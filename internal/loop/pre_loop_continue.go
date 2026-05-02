package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	PreLoopContinueCWDSession   = "session_cwd"
	PreLoopContinueCWDWorkspace = "workspace_root"

	DefaultPreLoopContinueTimeoutSeconds = 60
	DefaultPreLoopContinueMaxOutputBytes = 12000
)

type PreLoopContinueInput struct {
	EventName          string      `json:"event_name"`
	SessionID          string      `json:"session_id"`
	CWD                string      `json:"cwd"`
	WorkspaceRoot      string      `json:"workspace_root"`
	Loop               LoopRecord  `json:"loop"`
	Stop               StopPayload `json:"stop"`
	ContinuationReason string      `json:"continuation_reason"`
	RemainingSeconds   *int        `json:"remaining_seconds,omitempty"`
	Aggressive         bool        `json:"aggressive"`
	ContinuingAt       string      `json:"continuing_at"`
}

type preLoopContinueRunResult struct {
	Output    string
	Warning   string
	Truncated bool
}

type limitedOutputBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedOutputBuffer) Write(payload []byte) (int, error) {
	if b.limit <= 0 {
		b.truncated = true
		return len(payload), nil
	}
	remaining := b.limit - b.buffer.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(payload), nil
	}
	if len(payload) > remaining {
		if _, err := b.buffer.Write(payload[:remaining]); err != nil {
			return 0, fmt.Errorf("write limited output: %w", err)
		}
		b.truncated = true
		return len(payload), nil
	}
	if _, err := b.buffer.Write(payload); err != nil {
		return 0, fmt.Errorf("write limited output: %w", err)
	}
	return len(payload), nil
}

func (b *limitedOutputBuffer) String() string {
	return b.buffer.String()
}

func appendPreLoopContinue(ctx context.Context, paths Paths, payload StopPayload, record LoopRecord, remainingSeconds *int, aggressive bool, reason string, now time.Time) string {
	cfg := normalizeRuntimeConfig(LoadRuntimeConfig(paths)).PreLoopContinue
	if strings.TrimSpace(cfg.Command) == "" {
		return reason
	}

	result := runPreLoopContinue(ctx, cfg, payload, record, remainingSeconds, aggressive, reason, now)
	if result.Warning != "" {
		return strings.TrimSpace(reason + "\n\npre_loop_continue warning:\n" + result.Warning)
	}

	output := strings.TrimSpace(result.Output)
	if output == "" {
		return reason
	}
	if result.Truncated {
		output = strings.TrimSpace(fmt.Sprintf("%s\n[output truncated after %d bytes]", output, cfg.MaxOutputBytes))
	}
	return strings.TrimSpace(reason + "\n\npre_loop_continue output:\n" + output)
}

func runPreLoopContinue(ctx context.Context, cfg PreLoopContinueConfig, payload StopPayload, record LoopRecord, remainingSeconds *int, aggressive bool, reason string, now time.Time) preLoopContinueRunResult {
	cwd, err := resolvePreLoopContinueCWD(cfg, payload, record)
	if err != nil {
		return preLoopContinueRunResult{Warning: err.Error()}
	}

	command := strings.TrimSpace(cfg.Command)
	if strings.ContainsAny(command, `/\`) && !filepath.IsAbs(command) {
		command = filepath.Join(cwd, command)
	}

	input := PreLoopContinueInput{
		EventName:          "pre_loop_continue",
		SessionID:          record.SessionID,
		CWD:                cwd,
		WorkspaceRoot:      record.WorkspaceRoot,
		Loop:               record,
		Stop:               payload,
		ContinuationReason: reason,
		RemainingSeconds:   remainingSeconds,
		Aggressive:         aggressive,
		ContinuingAt:       ISOFormat(now),
	}
	stdin, err := json.Marshal(input)
	if err != nil {
		return preLoopContinueRunResult{Warning: fmt.Sprintf("encode input JSON: %v", err)}
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout limitedOutputBuffer
	stdout.limit = cfg.MaxOutputBytes
	cmd := exec.CommandContext(runCtx, command, cfg.Args...)
	cmd.Dir = cwd
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard

	err = cmd.Run()
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return preLoopContinueRunResult{Warning: fmt.Sprintf("command timed out after %s", timeout)}
	}
	if err != nil {
		return preLoopContinueRunResult{Warning: fmt.Sprintf("command failed: %s", describeCommandError(err))}
	}

	return preLoopContinueRunResult{
		Output:    stdout.String(),
		Truncated: stdout.truncated,
	}
}

func resolvePreLoopContinueCWD(cfg PreLoopContinueConfig, payload StopPayload, record LoopRecord) (string, error) {
	sessionCWD := strings.TrimSpace(payload.CWD)
	if sessionCWD == "" {
		sessionCWD = strings.TrimSpace(record.CWD)
	}
	workspaceRoot := strings.TrimSpace(record.WorkspaceRoot)

	var selected string
	switch strings.TrimSpace(cfg.CWD) {
	case "", PreLoopContinueCWDSession:
		selected = sessionCWD
	case PreLoopContinueCWDWorkspace:
		selected = workspaceRoot
		if selected == "" {
			selected = sessionCWD
		}
	default:
		return "", fmt.Errorf("invalid cwd %q; expected %q or %q", cfg.CWD, PreLoopContinueCWDSession, PreLoopContinueCWDWorkspace)
	}
	if selected == "" {
		return "", fmt.Errorf("cannot resolve cwd for pre_loop_continue")
	}
	abs, err := filepath.Abs(selected)
	if err != nil {
		return "", fmt.Errorf("resolve cwd %q: %w", selected, err)
	}
	return abs, nil
}

func describeCommandError(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Sprintf("exit status %d", exitErr.ExitCode())
	}
	return err.Error()
}
