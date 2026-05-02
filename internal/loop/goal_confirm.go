package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	GoalCheckOutcomeCompleted     = "completed"
	GoalCheckOutcomeIncomplete    = "incomplete"
	GoalCheckOutcomeError         = "error"
	GoalCheckOutcomeTimeout       = "timeout"
	GoalCheckOutcomeInvalidOutput = "invalid_output"
)

type GoalCheckVerdict struct {
	Completed         bool     `json:"completed"`
	Confidence        float64  `json:"confidence"`
	Reason            string   `json:"reason"`
	MissingWork       []string `json:"missing_work"`
	NextRoundGuidance string   `json:"next_round_guidance"`
}

type goalCheckResult struct {
	Outcome         string
	Verdict         GoalCheckVerdict
	Model           string
	ReasoningEffort string
	CommandName     string
	CommandArgCount int
	Duration        time.Duration
	Warning         string
	ErrorSummary    string
}

type goalCheckLogEvent struct {
	EventName              string   `json:"event_name"`
	CheckedAt              string   `json:"checked_at"`
	SessionID              string   `json:"session_id"`
	LoopName               string   `json:"loop_name"`
	LoopSlug               string   `json:"loop_slug"`
	LoopPath               string   `json:"loop_path"`
	LimitMode              string   `json:"limit_mode"`
	ContinueCount          int      `json:"continue_count"`
	GoalCheckCount         int      `json:"goal_check_count"`
	ConfirmModel           string   `json:"confirm_model,omitempty"`
	ConfirmReasoning       string   `json:"confirm_reasoning_effort,omitempty"`
	ConfirmCommand         string   `json:"confirm_command,omitempty"`
	ConfirmCommandArgCount int      `json:"confirm_command_arg_count,omitempty"`
	DurationMilliseconds   int64    `json:"duration_ms"`
	Outcome                string   `json:"outcome"`
	Confidence             float64  `json:"confidence,omitempty"`
	Reason                 string   `json:"reason,omitempty"`
	MissingWork            []string `json:"missing_work,omitempty"`
	MissingWorkCount       int      `json:"missing_work_count"`
	Warning                string   `json:"warning,omitempty"`
	ErrorSummary           string   `json:"error_summary,omitempty"`
	ContinuationEmitted    bool     `json:"continuation_emitted"`
	PreLoopContinueActive  bool     `json:"pre_loop_continue_active"`
}

func runGoalCheck(ctx context.Context, paths Paths, cfg GoalConfig, payload StopPayload, record LoopRecord, now time.Time) (result goalCheckResult) {
	started := time.Now()
	resolved := resolveGoalCheckConfig(cfg, payload, record)
	result = goalCheckResult{
		Outcome:         GoalCheckOutcomeError,
		Model:           resolved.ConfirmModel,
		ReasoningEffort: resolved.ConfirmReasoningEffort,
	}
	defer func() {
		result.Duration = time.Since(started)
	}()

	workspaceRoot := strings.TrimSpace(record.WorkspaceRoot)
	if workspaceRoot == "" {
		workspaceRoot = strings.TrimSpace(record.CWD)
	}
	if workspaceRoot == "" {
		result.Warning = "cannot resolve workspace root for goal confirmation"
		result.ErrorSummary = result.Warning
		return result
	}
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		result.Warning = fmt.Sprintf("resolve workspace root: %v", err)
		result.ErrorSummary = result.Warning
		return result
	}

	if err := os.MkdirAll(paths.RuntimeRoot(), 0o755); err != nil {
		result.Warning = fmt.Sprintf("create runtime directory: %v", err)
		result.ErrorSummary = result.Warning
		return result
	}
	prompt := buildGoalCheckPrompt(record, now)
	schemaPath, outputPath, promptPath, cleanup, err := prepareGoalCheckFiles(paths, prompt)
	if err != nil {
		result.Warning = err.Error()
		result.ErrorSummary = result.Warning
		return result
	}
	defer cleanup()

	multi := map[string][]string{
		"MODEL_ARGV":     nil,
		"REASONING_ARGV": nil,
	}
	if resolved.ConfirmModel != "" {
		multi["MODEL_ARGV"] = []string{"--model", resolved.ConfirmModel}
	}
	if resolved.ConfirmReasoningEffort != "" {
		multi["REASONING_ARGV"] = []string{"--config", fmt.Sprintf(`model_reasoning_effort="%s"`, resolved.ConfirmReasoningEffort)}
	}
	values := map[string]string{
		"PROMPT":           prompt,
		"PROMPT_FILE":      promptPath,
		"MODEL":            resolved.ConfirmModel,
		"REASONING_EFFORT": resolved.ConfirmReasoningEffort,
		"WORKSPACE_ROOT":   absWorkspaceRoot,
		"CWD":              strings.TrimSpace(record.CWD),
		"SESSION_ID":       record.SessionID,
		"LOOP_NAME":        record.Name,
		"LOOP_SLUG":        record.Slug,
		"SCHEMA_PATH":      schemaPath,
		"OUTPUT_PATH":      outputPath,
		"RUNS_LOG_PATH":    paths.RunsLogPath(),
		"CODEX_HOME":       paths.CodexHome,
	}
	command, args, env, err := buildConfiguredCommand(resolved.ConfirmCommand, absWorkspaceRoot, commandExpansion{
		Values:    values,
		Multi:     multi,
		EnvPrefix: "CODEX_LOOP_CONFIRM_",
	})
	if err != nil {
		result.Warning = err.Error()
		result.ErrorSummary = result.Warning
		return result
	}
	result.CommandName = filepath.Base(command)
	result.CommandArgCount = len(args)

	timeout := time.Duration(resolved.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, command, args...)
	cmd.Dir = absWorkspaceRoot
	cmd.Env = env
	cmd.Stdin = strings.NewReader(prompt)
	var stdout limitedOutputBuffer
	var stderr limitedOutputBuffer
	stdout.limit = resolved.MaxOutputBytes
	stderr.limit = resolved.MaxOutputBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		result.Outcome = GoalCheckOutcomeTimeout
		result.Warning = fmt.Sprintf("goal confirmation timed out after %s", timeout)
		result.ErrorSummary = result.Warning
		return result
	}
	if err != nil {
		result.Warning = fmt.Sprintf("goal confirmation failed: %s", describeCommandError(err))
		result.ErrorSummary = result.Warning
		if stderrText := strings.TrimSpace(stderr.String()); stderrText != "" {
			result.ErrorSummary = strings.TrimSpace(result.ErrorSummary + ": " + truncateText(stderrText, resolved.MaxOutputBytes))
		}
		return result
	}

	output, err := readGoalCheckOutput(outputPath, stdout.String())
	if err != nil {
		result.Outcome = GoalCheckOutcomeInvalidOutput
		result.Warning = fmt.Sprintf("read goal confirmation output: %v", err)
		result.ErrorSummary = result.Warning
		return result
	}
	if len(output) > resolved.MaxOutputBytes {
		output = output[:resolved.MaxOutputBytes]
	}
	verdict, err := decodeGoalVerdict(output)
	if err != nil {
		result.Outcome = GoalCheckOutcomeInvalidOutput
		result.Warning = fmt.Sprintf("decode goal confirmation output: %v", err)
		result.ErrorSummary = result.Warning
		return result
	}
	result.Verdict = verdict
	if verdict.Completed {
		result.Outcome = GoalCheckOutcomeCompleted
	} else {
		result.Outcome = GoalCheckOutcomeIncomplete
	}
	return result
}

func resolveGoalCheckConfig(cfg GoalConfig, payload StopPayload, record LoopRecord) GoalConfig {
	resolved := cfg
	if record.ConfirmModel != nil && strings.TrimSpace(*record.ConfirmModel) != "" {
		resolved.ConfirmModel = strings.TrimSpace(*record.ConfirmModel)
	} else if strings.TrimSpace(resolved.ConfirmModel) == "" && strings.TrimSpace(payload.Model) != "" {
		resolved.ConfirmModel = strings.TrimSpace(payload.Model)
	}
	if record.ConfirmReasoning != nil && strings.TrimSpace(*record.ConfirmReasoning) != "" {
		resolved.ConfirmReasoningEffort = strings.TrimSpace(*record.ConfirmReasoning)
	}
	if strings.TrimSpace(resolved.ConfirmCommand) == "" {
		resolved.ConfirmCommand = DefaultGoalConfirmCommand()
	}
	if resolved.TimeoutSeconds <= 0 {
		resolved.TimeoutSeconds = DefaultGoalTimeoutSeconds
	}
	if resolved.MaxOutputBytes <= 0 {
		resolved.MaxOutputBytes = DefaultGoalMaxOutputBytes
	}
	if !ValidReasoningEffort(resolved.ConfirmReasoningEffort) {
		resolved.ConfirmReasoningEffort = DefaultGoalConfirmReasoningEffort
	}
	return resolved
}

func prepareGoalCheckFiles(paths Paths, prompt string) (string, string, string, func(), error) {
	schemaFile, err := os.CreateTemp(paths.RuntimeRoot(), "goal-schema-*.json")
	if err != nil {
		return "", "", "", func() {}, fmt.Errorf("create goal schema file: %w", err)
	}
	outputFile, err := os.CreateTemp(paths.RuntimeRoot(), "goal-output-*.json")
	if err != nil {
		_ = schemaFile.Close()
		_ = os.Remove(schemaFile.Name())
		return "", "", "", func() {}, fmt.Errorf("create goal output file: %w", err)
	}
	promptFile, err := os.CreateTemp(paths.RuntimeRoot(), "goal-prompt-*.txt")
	if err != nil {
		_ = schemaFile.Close()
		_ = outputFile.Close()
		_ = os.Remove(schemaFile.Name())
		_ = os.Remove(outputFile.Name())
		return "", "", "", func() {}, fmt.Errorf("create goal prompt file: %w", err)
	}
	cleanup := func() {
		_ = os.Remove(schemaFile.Name())
		_ = os.Remove(outputFile.Name())
		_ = os.Remove(promptFile.Name())
	}
	if _, err := schemaFile.WriteString(goalCheckOutputSchema); err != nil {
		_ = schemaFile.Close()
		_ = outputFile.Close()
		_ = promptFile.Close()
		cleanup()
		return "", "", "", func() {}, fmt.Errorf("write goal schema file: %w", err)
	}
	if _, err := promptFile.WriteString(prompt); err != nil {
		_ = schemaFile.Close()
		_ = outputFile.Close()
		_ = promptFile.Close()
		cleanup()
		return "", "", "", func() {}, fmt.Errorf("write goal prompt file: %w", err)
	}
	if err := schemaFile.Close(); err != nil {
		_ = outputFile.Close()
		_ = promptFile.Close()
		cleanup()
		return "", "", "", func() {}, fmt.Errorf("close goal schema file: %w", err)
	}
	if err := outputFile.Close(); err != nil {
		_ = promptFile.Close()
		cleanup()
		return "", "", "", func() {}, fmt.Errorf("close goal output file: %w", err)
	}
	if err := promptFile.Close(); err != nil {
		cleanup()
		return "", "", "", func() {}, fmt.Errorf("close goal prompt file: %w", err)
	}
	return schemaFile.Name(), outputFile.Name(), promptFile.Name(), cleanup, nil
}

func buildGoalCheckPrompt(record LoopRecord, now time.Time) string {
	goalText := ""
	if record.GoalText != nil {
		goalText = strings.TrimSpace(*record.GoalText)
	}
	task := strings.TrimSpace(record.TaskPrompt)
	if task == "" {
		task = strings.TrimSpace(record.ActivationPrompt)
	}
	latestMessage := ""
	if record.LastAssistantMessage != nil {
		latestMessage = strings.TrimSpace(*record.LastAssistantMessage)
	}
	lines := []string{
		"You are the codex-loop goal confirmation reviewer.",
		"Inspect the current workspace in read-only mode and decide whether the original task is actually complete.",
		"Do not edit files, run formatters that write files, apply patches, or perform destructive actions.",
		"Return only JSON matching the provided schema.",
		"",
		fmt.Sprintf("Checked at: %s", ISOFormat(now)),
		fmt.Sprintf("Loop name: %s", record.Name),
		fmt.Sprintf("Loop slug: %s", record.Slug),
		fmt.Sprintf("Workspace root: %s", record.WorkspaceRoot),
	}
	if goalText != "" {
		lines = append(lines, "", "Additional goal:", goalText)
	}
	if task != "" {
		lines = append(lines, "", "Original task:", task)
	}
	if latestMessage != "" {
		lines = append(lines, "", "Latest assistant message:", latestMessage)
	}
	lines = append(lines,
		"",
		"Decision rules:",
		"- Set completed to true only when the requested work is actually done and no required follow-up remains.",
		"- Use confidence from 0 to 1.",
		"- If incomplete, list concrete missing work and provide next_round_guidance.",
		"- If uncertain, set completed to false.",
	)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func decodeGoalVerdict(content []byte) (GoalCheckVerdict, error) {
	trimmed := bytes.TrimSpace(content)
	trimmed = bytes.TrimPrefix(trimmed, []byte("```json"))
	trimmed = bytes.TrimPrefix(trimmed, []byte("```"))
	trimmed = bytes.TrimSuffix(trimmed, []byte("```"))
	trimmed = bytes.TrimSpace(trimmed)
	if len(trimmed) == 0 {
		return GoalCheckVerdict{}, fmt.Errorf("empty output")
	}
	var verdict GoalCheckVerdict
	if err := json.Unmarshal(trimmed, &verdict); err != nil {
		return GoalCheckVerdict{}, err
	}
	verdict.Reason = strings.TrimSpace(verdict.Reason)
	verdict.NextRoundGuidance = strings.TrimSpace(verdict.NextRoundGuidance)
	for index, item := range verdict.MissingWork {
		verdict.MissingWork[index] = strings.TrimSpace(item)
	}
	if verdict.Confidence < 0 {
		verdict.Confidence = 0
	}
	if verdict.Confidence > 1 {
		verdict.Confidence = 1
	}
	if verdict.Reason == "" {
		return GoalCheckVerdict{}, fmt.Errorf("missing reason")
	}
	return verdict, nil
}

func readGoalCheckOutput(outputPath string, stdout string) ([]byte, error) {
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read output file: %w", err)
	}
	if len(bytes.TrimSpace(output)) > 0 {
		return output, nil
	}
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return nil, fmt.Errorf("empty output file and stdout")
	}
	return []byte(stdout), nil
}

func AppendGoalCheckLog(paths Paths, loopPath string, record LoopRecord, result goalCheckResult, continuationEmitted bool, preLoopContinueActive bool, checkedAt time.Time) error {
	event := goalCheckLogEvent{
		EventName:              "goal_check",
		CheckedAt:              ISOFormat(checkedAt),
		SessionID:              record.SessionID,
		LoopName:               record.Name,
		LoopSlug:               record.Slug,
		LoopPath:               loopPath,
		LimitMode:              ResolveLimitMode(record),
		ContinueCount:          record.ContinueCount,
		GoalCheckCount:         record.GoalCheckCount,
		ConfirmModel:           result.Model,
		ConfirmReasoning:       result.ReasoningEffort,
		ConfirmCommand:         result.CommandName,
		ConfirmCommandArgCount: result.CommandArgCount,
		DurationMilliseconds:   result.Duration.Milliseconds(),
		Outcome:                result.Outcome,
		Confidence:             result.Verdict.Confidence,
		Reason:                 truncateText(result.Verdict.Reason, 1000),
		MissingWork:            truncateStringSlice(result.Verdict.MissingWork, 20, 500),
		MissingWorkCount:       len(result.Verdict.MissingWork),
		Warning:                truncateText(result.Warning, 1000),
		ErrorSummary:           truncateText(result.ErrorSummary, 1000),
		ContinuationEmitted:    continuationEmitted,
		PreLoopContinueActive:  preLoopContinueActive,
	}
	return AppendJSONL(paths.RunsLogPath(), event)
}

func AppendJSONL(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create JSONL parent directory: %w", err)
	}
	handle, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open JSONL log %q: %w", path, err)
	}
	defer handle.Close()
	encoder := json.NewEncoder(handle)
	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("append JSONL log %q: %w", path, err)
	}
	return nil
}

func truncateText(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}

func truncateStringSlice(values []string, maxItems int, maxItemBytes int) []string {
	if maxItems <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) > maxItems {
		values = values[:maxItems]
	}
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		output = append(output, truncateText(value, maxItemBytes))
	}
	return output
}

const goalCheckOutputSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["completed", "confidence", "reason", "missing_work", "next_round_guidance"],
  "properties": {
    "completed": { "type": "boolean" },
    "confidence": { "type": "number", "minimum": 0, "maximum": 1 },
    "reason": { "type": "string" },
    "missing_work": {
      "type": "array",
      "items": { "type": "string" }
    },
    "next_round_guidance": { "type": "string" }
  }
}
`
