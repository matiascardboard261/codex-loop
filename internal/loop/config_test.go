package loop

import (
	"strings"
	"testing"
)

func TestRuntimeConfigDefaultsPreLoopContinue(t *testing.T) {
	t.Parallel()

	cfg := LoadRuntimeConfig(mustPaths(t))

	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDSession {
		t.Fatalf("expected default cwd %q, got %q", PreLoopContinueCWDSession, cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != DefaultPreLoopContinueTimeoutSeconds {
		t.Fatalf("expected default timeout %d, got %d", DefaultPreLoopContinueTimeoutSeconds, cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != DefaultPreLoopContinueMaxOutputBytes {
		t.Fatalf("expected default max output %d, got %d", DefaultPreLoopContinueMaxOutputBytes, cfg.PreLoopContinue.MaxOutputBytes)
	}
	if cfg.PreLoopContinue.Command != "" {
		t.Fatalf("expected disabled pre_loop_continue command, got %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.Hooks.StopTimeoutSeconds != DefaultStopHookTimeoutSeconds {
		t.Fatalf("expected stop timeout %d, got %d", DefaultStopHookTimeoutSeconds, cfg.Hooks.StopTimeoutSeconds)
	}
	if cfg.Goal.ConfirmModel != DefaultGoalConfirmModel {
		t.Fatalf("expected goal model %q, got %q", DefaultGoalConfirmModel, cfg.Goal.ConfirmModel)
	}
	if cfg.Goal.ConfirmReasoningEffort != DefaultGoalConfirmReasoningEffort {
		t.Fatalf("expected goal reasoning %q, got %q", DefaultGoalConfirmReasoningEffort, cfg.Goal.ConfirmReasoningEffort)
	}
}

func TestRuntimeConfigParsesPreLoopContinue(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "scripts/pre-loop --json"
cwd = "workspace_root"
timeout_seconds = 7
max_output_bytes = 42
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.PreLoopContinue.Command != "scripts/pre-loop --json" {
		t.Fatalf("unexpected command %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDWorkspace {
		t.Fatalf("unexpected cwd %q", cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != 7 {
		t.Fatalf("unexpected timeout %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != 42 {
		t.Fatalf("unexpected max output %d", cfg.PreLoopContinue.MaxOutputBytes)
	}
}

func TestRuntimeConfigParsesGoalAndHookSettings(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[hooks]
stop_timeout_seconds = 120

[goal]
confirm_model = "gpt-5.4"
confirm_reasoning_effort = "xhigh"
confirm_command = "bin/codex exec $MODEL_ARGV $REASONING_ARGV $PROMPT_FILE"
timeout_seconds = 90
max_output_bytes = 77
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.Hooks.StopTimeoutSeconds != 120 {
		t.Fatalf("unexpected stop timeout %d", cfg.Hooks.StopTimeoutSeconds)
	}
	if cfg.Goal.ConfirmModel != "gpt-5.4" {
		t.Fatalf("unexpected model %q", cfg.Goal.ConfirmModel)
	}
	if cfg.Goal.ConfirmReasoningEffort != "xhigh" {
		t.Fatalf("unexpected reasoning %q", cfg.Goal.ConfirmReasoningEffort)
	}
	if cfg.Goal.ConfirmCommand != "bin/codex exec $MODEL_ARGV $REASONING_ARGV $PROMPT_FILE" {
		t.Fatalf("unexpected confirm command %#v", cfg.Goal.ConfirmCommand)
	}
	if cfg.Goal.TimeoutSeconds != 90 {
		t.Fatalf("unexpected goal timeout %d", cfg.Goal.TimeoutSeconds)
	}
	if cfg.Goal.MaxOutputBytes != 77 {
		t.Fatalf("unexpected max output %d", cfg.Goal.MaxOutputBytes)
	}
}

func TestRuntimeConfigAllowsBlankGoalModelAndReasoning(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[goal]
confirm_model = ""
confirm_reasoning_effort = ""
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.Goal.ConfirmModel != "" {
		t.Fatalf("expected blank goal model, got %q", cfg.Goal.ConfirmModel)
	}
	if cfg.Goal.ConfirmReasoningEffort != "" {
		t.Fatalf("expected blank goal reasoning, got %q", cfg.Goal.ConfirmReasoningEffort)
	}
}

func TestRuntimeConfigDefaultGoalConfirmCommandUsesYolo(t *testing.T) {
	t.Parallel()

	cfg := LoadRuntimeConfig(mustPaths(t))

	if !strings.Contains(cfg.Goal.ConfirmCommand, "--yolo") {
		t.Fatalf("expected default confirm command to include --yolo, got %#v", cfg.Goal.ConfirmCommand)
	}
	if !strings.Contains(cfg.Goal.ConfirmCommand, "$MODEL_ARGV") {
		t.Fatalf("expected default confirm command to include model argv placeholder, got %#v", cfg.Goal.ConfirmCommand)
	}
	if !strings.Contains(cfg.Goal.ConfirmCommand, "$REASONING_ARGV") {
		t.Fatalf("expected default confirm command to include reasoning argv placeholder, got %#v", cfg.Goal.ConfirmCommand)
	}
}

func TestRuntimeConfigNormalizesInvalidGoalValues(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[hooks]
stop_timeout_seconds = 40

[goal]
confirm_reasoning_effort = "invalid"
timeout_seconds = 100
max_output_bytes = 0
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.Goal.ConfirmReasoningEffort != DefaultGoalConfirmReasoningEffort {
		t.Fatalf("expected default reasoning, got %q", cfg.Goal.ConfirmReasoningEffort)
	}
	if cfg.Goal.TimeoutSeconds != 10 {
		t.Fatalf("expected goal timeout normalized to 10, got %d", cfg.Goal.TimeoutSeconds)
	}
	if cfg.Goal.MaxOutputBytes != DefaultGoalMaxOutputBytes {
		t.Fatalf("expected default max output, got %d", cfg.Goal.MaxOutputBytes)
	}
}

func TestRuntimeConfigNormalizesPreLoopContinueDefaults(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/echo"
timeout_seconds = 0
max_output_bytes = -1
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDSession {
		t.Fatalf("expected default cwd, got %q", cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != DefaultPreLoopContinueTimeoutSeconds {
		t.Fatalf("expected default timeout, got %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != DefaultPreLoopContinueMaxOutputBytes {
		t.Fatalf("expected default max output, got %d", cfg.PreLoopContinue.MaxOutputBytes)
	}
}
