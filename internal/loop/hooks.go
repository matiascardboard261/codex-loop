package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RapidStopThreshold = 120 * time.Second
	RapidStopLimit     = 3
)

type UserPromptPayload struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	Prompt    string `json:"prompt"`
}

type StopPayload struct {
	SessionID            string  `json:"session_id"`
	CWD                  string  `json:"cwd"`
	LastAssistantMessage *string `json:"last_assistant_message"`
}

type HookResult map[string]any

func HandleUserPromptSubmit(paths Paths, payload UserPromptPayload, now time.Time) (HookResult, error) {
	if payload.Prompt == "" || !LooksLikeActivation(payload.Prompt) {
		return nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	activation, ok, err := ExtractActivation(payload.Prompt)
	if err != nil {
		return BlockWithReason(fmt.Sprintf("Codex loop activation failed: %v", err)), nil
	}
	if !ok {
		return nil, nil
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		return BlockWithReason("Codex loop activation failed: codex-loop activation requires a session_id from Codex"), nil
	}

	workingDir := payload.CWD
	if strings.TrimSpace(workingDir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve current working directory: %w", err)
		}
		workingDir = cwd
	}
	resolvedCWD, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("resolve cwd %q: %w", workingDir, err)
	}
	workspaceRoot, err := ResolveWorkspaceRoot(resolvedCWD)
	if err != nil {
		return nil, err
	}

	if err := SupersedeActiveLoops(paths, payload.SessionID, ""); err != nil {
		return nil, err
	}
	loopPath := CreateLoopPath(paths, activation.Slug, now)
	record := BuildLoopRecord(payload.SessionID, resolvedCWD, workspaceRoot, activation, now)
	if err := ReplaceLoopFile(loopPath, record); err != nil {
		return nil, err
	}
	return nil, nil
}

func HandleStop(paths Paths, payload StopPayload, now time.Time) (HookResult, error) {
	if strings.TrimSpace(payload.SessionID) == "" {
		return nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	active, err := ResolveActiveLoop(paths, payload.SessionID)
	if err != nil {
		return StopWarning(fmt.Sprintf("Codex loop stop hook failed: %v", err)), nil
	}
	if active == nil {
		return nil, nil
	}

	record := active.Record
	previousStop, err := ParseISO8601(record.LastStopAt)
	if err != nil {
		return StopWarning(fmt.Sprintf("Codex loop stop hook failed: %v", err)), nil
	}

	record.LastAssistantMessage = payload.LastAssistantMessage
	lastStopAt := ISOFormat(now)
	record.LastStopAt = &lastStopAt

	limitMode := ResolveLimitMode(record)
	var remainingSeconds *int
	if limitMode == LimitModeTime {
		deadlineAt, err := ParseISO8601(record.DeadlineAt)
		if err != nil {
			return StopWarning(fmt.Sprintf("Codex loop stop hook failed: %v", err)), nil
		}
		if deadlineAt == nil {
			return StopWarning("Codex loop stop hook failed: active codex-loop is missing deadline_at"), nil
		}
		if !now.Before(*deadlineAt) {
			record.Status = StatusCompleted
			if err := ReplaceLoopFile(active.Path, record); err != nil {
				return StopWarning(fmt.Sprintf("Codex loop stop hook failed: %v", err)), nil
			}
			return nil, nil
		}
		remaining := max(0, int(deadlineAt.Sub(now).Seconds()))
		remainingSeconds = &remaining
	} else {
		targetRounds := 0
		if record.TargetRounds != nil {
			targetRounds = *record.TargetRounds
		}
		if targetRounds <= 0 {
			return StopWarning("Codex loop stop hook failed: active codex-loop is missing a positive target_rounds"), nil
		}
		record.CompletedRounds++
		if record.CompletedRounds >= targetRounds {
			record.Status = StatusCompleted
			if err := ReplaceLoopFile(active.Path, record); err != nil {
				return StopWarning(fmt.Sprintf("Codex loop stop hook failed: %v", err)), nil
			}
			return nil, nil
		}
	}

	rapidCount := 0
	if previousStop != nil && now.Sub(*previousStop) <= RapidStopThreshold {
		rapidCount = record.RapidStopCount + 1
	}
	record.RapidStopCount = rapidCount

	if rapidCount >= RapidStopLimit && record.EscalationUsed {
		record.Status = StatusCutShort
		if err := ReplaceLoopFile(active.Path, record); err != nil {
			return StopWarning(fmt.Sprintf("Codex loop stop hook failed: %v", err)), nil
		}
		return StopWarning("Codex loop stopped after repeated rapid completions. Review the latest result manually before reactivating the loop."), nil
	}

	aggressive := rapidCount >= RapidStopLimit && !record.EscalationUsed
	if aggressive {
		record.EscalationUsed = true
	}

	record.ContinueCount++
	lastContinueAt := ISOFormat(now)
	record.LastContinueAt = &lastContinueAt
	if err := ReplaceLoopFile(active.Path, record); err != nil {
		return StopWarning(fmt.Sprintf("Codex loop stop hook failed: %v", err)), nil
	}

	return HookResult{
		"decision": "block",
		"reason":   ContinuationReason(paths, record, remainingSeconds, aggressive),
	}, nil
}

func ContinuationReason(paths Paths, record LoopRecord, remainingSeconds *int, aggressive bool) string {
	workspaceRoot := record.WorkspaceRoot
	if workspaceRoot == "" {
		workspaceRoot = record.CWD
	}
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err == nil {
		workspaceRoot = absWorkspaceRoot
	}
	continuationConfig := ResolveOptionalContinuationConfig(paths, workspaceRoot)
	originalTask := strings.TrimSpace(record.TaskPrompt)
	if originalTask == "" {
		originalTask = strings.TrimSpace(record.ActivationPrompt)
	}
	latestMessage := ""
	if record.LastAssistantMessage != nil {
		latestMessage = strings.TrimSpace(*record.LastAssistantMessage)
	}

	lines := []string{"Continue the active codex-loop task."}
	if ResolveLimitMode(record) == LimitModeTime {
		remaining := 0
		if remainingSeconds != nil {
			remaining = *remainingSeconds
		}
		lines = append(lines,
			fmt.Sprintf("The minimum work duration has not elapsed yet. Remaining time: %s.", FormatSeconds(remaining)),
			"Do not stop just because the primary request appears complete.",
		)
	} else {
		targetRounds := 0
		if record.TargetRounds != nil {
			targetRounds = *record.TargetRounds
		}
		nextRound := record.CompletedRounds + 1
		if targetRounds > 0 && nextRound > targetRounds {
			nextRound = targetRounds
		}
		lines = append(lines,
			fmt.Sprintf("Round %d of %d begins now.", nextRound, targetRounds),
			fmt.Sprintf("You have completed %d of %d required rounds.", record.CompletedRounds, targetRounds),
			"Treat this as a deliberate fresh pass. Do not just restate the previous conclusion.",
		)
	}

	if aggressive {
		lines = append(lines, "Several turns have ended too quickly. Broaden the scope materially before stopping again.")
	}

	lines = append(lines,
		"Expand the work with:",
		"- hardening and cleanup of the current solution",
		"- edge cases and larger scenarios",
		"- adjacent project areas that may share the same weakness",
		"- stronger validation with real commands, tests, or QA evidence",
		"- additional regression coverage where the same failure mode could recur",
	)
	if ResolveLimitMode(record) == LimitModeRounds {
		lines = append(lines, "- a fresh challenge to any earlier conclusion before you stop again")
	}
	if continuationConfig.SkillName != "" && continuationConfig.SkillPath != "" {
		lines = append(lines, fmt.Sprintf("- explicit use of the %s skill at %s", continuationConfig.SkillName, continuationConfig.SkillPath))
	}
	if continuationConfig.ExtraGuidance != "" {
		lines = append(lines, "", "Additional configured guidance:", continuationConfig.ExtraGuidance)
	}
	if originalTask != "" {
		lines = append(lines, "", "Original task:", originalTask)
	}
	if latestMessage != "" {
		lines = append(lines, "", "Latest assistant message before this continuation:", latestMessage)
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func FormatSeconds(seconds int) string {
	if seconds <= 0 {
		return "0s"
	}
	parts := make([]string, 0, 4)
	remainder := seconds
	for _, item := range []struct {
		suffix string
		unit   int
	}{
		{suffix: "d", unit: 86400},
		{suffix: "h", unit: 3600},
		{suffix: "m", unit: 60},
		{suffix: "s", unit: 1},
	} {
		amount := remainder / item.unit
		remainder %= item.unit
		if amount > 0 {
			parts = append(parts, fmt.Sprintf("%d%s", amount, item.suffix))
		}
	}
	return strings.Join(parts, " ")
}

func StopWarning(message string) HookResult {
	return HookResult{
		"continue":      false,
		"stopReason":    "codex-loop-cut-short",
		"systemMessage": message,
	}
}

func BlockWithReason(reason string) HookResult {
	return HookResult{
		"decision": "block",
		"reason":   reason,
	}
}
