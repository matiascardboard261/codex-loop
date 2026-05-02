Goal (incl. success criteria):
- Implement accepted plan for a synchronous `pre_loop_continue` runtime hook inside the codex-loop Stop handler.
- Success: structured config works, command runs synchronously before automatic continuation prompts, output is injected into `reason`, failures continue with warnings, docs/tests updated, README explains the hook with an example, and verification passes.
Constraints/Assumptions:
- Use `$openai-docs` / official OpenAI docs for Codex hook behavior.
- Follow workspace restrictions: no destructive git commands.
- Accepted plan must be persisted under `.codex/plans/`.
- Go changes must use `golang-pro`; final gate should include `make verify`.
Key decisions:
- Official hooks docs are the authority for Stop semantics.
- `pre_loop_continue` is an internal codex-loop runtime hook, not a second Codex lifecycle hook.
- Default configured execution cwd is `session_cwd`, with optional `workspace_root`.
- Use structured `[pre_loop_continue]` config, command+args without shell, JSON stdin, stdout-only prompt injection, and continue-with-warning on failure.
State:
- README improvement complete and verified; final response pending.
Done:
- Scanned existing ledgers for cross-agent awareness.
- Fetched official OpenAI Codex Hooks docs, including Stop and Common output fields.
- Inspected local `internal/loop/hooks.go`, `internal/loop/config.go`, README config docs, and hook registrations.
- User accepted/revised the plan.
- Persisted accepted plan to `.codex/plans/2026-05-02-pre-loop-continue.md`.
- Implemented `[pre_loop_continue]` runtime config with defaults.
- Added synchronous subprocess execution inside `HandleStop` with JSON stdin, stdout prompt injection, timeout, truncation, cwd selection, and warning-on-failure behavior.
- Wired CLI Stop hook to pass `cmd.Context()`.
- Added focused config, Stop handler, and CLI hook tests.
- Updated README and plugin skill docs.
- Verification passed: `go test ./internal/loop ./internal/cli`, `make fmt`, `go vet ./...`, and `make verify`.
- Expanded README `pre_loop_continue` documentation with purpose, lifecycle timing, runtime behavior, config example, and script example.
- Verification passed again after README update: `make verify`.
Now:
- Prepare final summary.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- `.codex/ledger/2026-05-02-MEMORY-pre-loop-hook.md`
- `.codex/plans/2026-05-02-pre-loop-continue.md`
- `internal/loop/hooks.go`
- `internal/loop/config.go`
- `internal/loop/pre_loop_continue.go`
- `internal/loop/config_test.go`
- `internal/loop/hooks_test.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `README.md`
- `plugins/codex-loop/skills/codex-loop/SKILL.md`
- Commands: `go test ./internal/loop ./internal/cli`, `make fmt`, `go vet ./...`, `make verify`
- OpenAI docs: `https://developers.openai.com/codex/hooks`
