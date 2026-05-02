Goal (incl. success criteria):
- Change codex-loop goal confirmation so the old Claude CLI confirm command can emit tool/prose chatter around the verdict without causing invalid_output.
- Success: codex-loop extracts a valid goal verdict JSON object from noisy output, keeps strict verdict validation, and verification passes.
Constraints/Assumptions:
- No destructive git commands.
- User asked to delete Bun wrapper experiment and keep old command; cleanup completed.
- Go changes require golang-pro.
- Do not touch unrelated dirty worktree changes.
Key decisions:
- Keep global `~/.codex/codex-loop/config.toml` on old `claude --dangerously-skip-permissions ... --output-format text` command.
- Implement tolerant parsing in production code, not a workaround command.
State:
- Implementation in progress.
Done:
- Removed `/Users/pedronauck/.codex/codex-loop/goal-confirm/`.
- Removed `.codex/plans/2026-05-02-claude-agent-sdk-confirm-command.md`.
- Removed `.codex/ledger/2026-05-02-MEMORY-claude-sdk-confirm.md`.
- Confirmed global config still uses old Claude CLI command.
- Read golang-pro skill.
- Implemented tolerant goal verdict decoding that can extract a matching JSON object from noisy output.
- Added focused decoder tests and an integration-style Stop hook test for noisy stdout.
- Updated README and plugin README with custom runner noisy-output behavior.
Now:
- Run gofmt and focused tests.
Next:
- Fix any test failures, then run `go vet ./...` and `make verify`.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- `.codex/ledger/2026-05-02-MEMORY-tolerant-confirm-output.md`
- `internal/loop/goal_confirm.go`
- `internal/loop/goal_confirm_test.go`
- `internal/loop/hooks_test.go`
- `README.md`
- `plugins/codex-loop/README.md`
