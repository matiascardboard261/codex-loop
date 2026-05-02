package loop

import "testing"

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
		t.Fatalf("expected disabled pre_loop_continue command, got %q", cfg.PreLoopContinue.Command)
	}
}

func TestRuntimeConfigParsesPreLoopContinue(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "scripts/pre-loop"
args = ["--json"]
cwd = "workspace_root"
timeout_seconds = 7
max_output_bytes = 42
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.PreLoopContinue.Command != "scripts/pre-loop" {
		t.Fatalf("unexpected command %q", cfg.PreLoopContinue.Command)
	}
	if len(cfg.PreLoopContinue.Args) != 1 || cfg.PreLoopContinue.Args[0] != "--json" {
		t.Fatalf("unexpected args %#v", cfg.PreLoopContinue.Args)
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
