# Plan: Synchronous `pre_loop_continue` in Stop Hook

## Summary

Add an internal configurable `pre_loop_continue` hook to `codex-loop`. It runs synchronously inside the managed `Stop` handler only when `codex-loop` is about to emit an automatic continuation prompt.

The Codex `Stop` hook stdout remains protocol JSON. The configured command output is captured by `codex-loop` and appended to the JSON `reason` string that Codex uses as the continuation prompt.

## Key Changes

- Add `~/.codex/codex-loop/config.toml` support:

```toml
[pre_loop_continue]
command = ""
args = []
cwd = "session_cwd"
timeout_seconds = 60
max_output_bytes = 12000
```

- Public behavior:
  - `command = ""` disables the feature.
  - `command` and `args` execute without shell expansion.
  - `cwd` accepts `session_cwd` and `workspace_root`; default is `session_cwd`.
  - The command receives structured JSON on stdin with loop, Stop payload, and continuation context.
  - Only stdout is appended to the continuation prompt.
  - stderr is not injected; failures become short continuation warnings.
  - stdout above `max_output_bytes` is truncated with an explicit marker.

- Runtime behavior:
  - Run only after `HandleStop` decides to continue.
  - Do not run for completed, inactive, cut-short, or `continue: false` outcomes.
  - Persist the updated loop record before running the command.
  - Append successful output under a stable `pre_loop_continue output:` section.
  - Use `context.Context` and `exec.CommandContext` for timeout/cancellation.

## Failure Behavior

- On start failure, non-zero exit, timeout, invalid `cwd`, or invalid command config:
  - continue the loop with `decision: "block"`;
  - append a short `pre_loop_continue warning:` section to the continuation reason;
  - do not inject raw stderr.

## Test Plan

- Config tests for defaults, parsing, disabled command, and invalid `cwd`.
- Stop handler tests for successful output, JSON stdin, default `session_cwd`, `workspace_root`, failures, timeout, truncation, and non-execution on completion/cut-short.
- CLI hook test confirming final stdout remains valid JSON and includes captured output.
- Final gate: `make fmt`, `go vet ./...`, `make verify`.

## Assumptions

- The configured command is trusted local code.
- Shell mode is out of scope for v1; users can point to scripts for shell behavior.
- This is a `codex-loop` runtime hook, not an additional Codex lifecycle hook.
