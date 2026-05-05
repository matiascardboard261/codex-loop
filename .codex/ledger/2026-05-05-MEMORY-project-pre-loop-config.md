Goal (incl. success criteria):
- Implement project-local `codex-loop.toml` support so `[pre_loop_continue]` can be configured per project.
- Success: effective config precedence is defaults -> global runtime config -> nearest project `codex-loop.toml` within workspace; overlay is per field; Stop/pre-loop/goal paths use effective config; docs/tests updated; `make verify` passes.
Constraints/Assumptions:
- No destructive git commands.
- Use `golang-pro` for Go changes.
- Accepted plan must be persisted under `.codex/plans/2026-05-05-project-pre-loop-config.md`.
- Durable artifacts must be in English.
- Project config file is named exactly `codex-loop.toml`.
Key decisions:
- Discover project config from Stop payload CWD upward to resolved workspace root, never above it.
- Use nearest `codex-loop.toml` when multiple files exist.
- Apply project config as per-field overlay over global config/defaults.
- Explicit local `command = ""` disables global `pre_loop_continue.command` for that project.
State:
- Implementation complete; final response pending.
Done:
- Read existing ledgers for cross-agent awareness.
- Read `golang-pro` skill.
- Confirmed initial worktree is clean.
- User accepted the proposed plan by requesting implementation.
- Persisted accepted plan to `.codex/plans/2026-05-05-project-pre-loop-config.md`.
- Implemented project config discovery and effective runtime config overlay.
- Wired effective config into Stop goal confirmation, continuation guidance, `pre_loop_continue`, and goal-check logging.
- Added config, Stop, and CLI regressions for local config precedence.
- Focused tests passed: `go test ./internal/loop ./internal/cli`.
- Updated README and plugin skill documentation.
- Verification passed: `make fmt`, `go test ./internal/loop ./internal/cli`, `go vet ./...`, `make verify`.
Now:
- Prepare final response.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- `.codex/ledger/2026-05-05-MEMORY-project-pre-loop-config.md`
- `.codex/plans/2026-05-05-project-pre-loop-config.md`
- `internal/loop/config.go`, `internal/loop/hooks.go`, `internal/loop/pre_loop_continue.go`, `internal/loop/config_test.go`, `internal/loop/hooks_test.go`, `internal/cli/root_test.go`, `README.md`, `plugins/codex-loop/skills/codex-loop/SKILL.md`
- Commands: `make fmt`, `go test ./internal/loop ./internal/cli`, `go vet ./...`, `make verify`
