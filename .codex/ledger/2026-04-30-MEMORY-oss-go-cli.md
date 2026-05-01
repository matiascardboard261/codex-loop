Goal (incl. success criteria):
- Implement accepted plan to turn current codex-loop-plugin repo into OSS `codex-loop` Go/Cobra CLI and Codex plugin.
- Success: Go runtime replaces Python, plugin remains installable via Codex marketplace/lifecycle hooks, docs/license updated, tests and `make verify` pass.
Constraints/Assumptions:
- Follow workspace AGENTS prompt: no destructive git commands; preserve unrelated changes; tests must find bugs, not hide them.
- Accepted plan must be persisted under `.codex/plans/`.
- Public name: `codex-loop`; module: `github.com/pedronauck/codex-loop`; license: MIT.
- Distribution primary: `go install github.com/pedronauck/codex-loop/cmd/codex-loop@latest`.
- Greenfield product: no compatibility, migration, or public mention of prior names/artifacts.
Key decisions:
- Use Go/Cobra for CLI and hook runtime.
- Use Codex plugin lifecycle config `hooks/hooks.json` instead of primary mutation of global `~/.codex/hooks.json`.
- Keep state JSON stable under `~/.codex/codex-loop/loops/`.
State:
- Implementation complete and verified; final response pending.
Done:
- Explored current Python plugin/runtime and go-devstack boilerplate.
- Confirmed official Codex plugin/hooks docs and Cobra package docs during planning.
- Persisted accepted plan under `.codex/plans/2026-04-30-codex-loop-go-cli.md`.
- Added Go module/build scaffolding and removed tracked Python plugin runtime files.
- Updated plan direction to remove migration/compatibility requirements.
- Implemented greenfield Go/Cobra CLI, hook runtime, installer, plugin bundle, docs, MIT license, and tests.
- Removed residual old plugin directory from the workspace.
- Verification passed: `make verify`; CODEX_HOME smoke test passed.
Now:
- Prepare final summary.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None blocking.
Working set (files/ids/commands):
- `.codex/plans/2026-04-30-codex-loop-go-cli.md`
- `.codex/ledger/2026-04-30-MEMORY-oss-go-cli.md`
- `cmd/codex-loop/`
- `internal/cli/`, `internal/installer/`, `internal/loop/`, `internal/logger/`, `internal/version/`
- `plugins/codex-loop/`
