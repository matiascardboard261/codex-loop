Goal (incl. success criteria):
- Add/adapt this repo's AGENTS.md from ~/dev/compozy/agh/AGENTS.md for codex-loop-plugin.
- Success: project instructions match this repo's Go/Cobra CLI + Codex plugin shape and align with golang-pro requirements.
Constraints/Assumptions:
- Follow workspace prompt: no destructive git commands; preserve unrelated changes.
- Use golang-pro guidance for Go production/testing/linting rules.
- Source AGENTS.md is AGH-specific and must be adapted, not copied verbatim.
Key decisions:
- Treat existing .codex/ledger/2026-04-30-MEMORY-oss-go-cli.md as read-only cross-agent context.
State:
- Completed; final response pending.
Done:
- Read golang-pro skill.
- Read source AGH AGENTS.md.
- Confirmed current repo has no root AGENTS.md yet.
- Inspected Makefile, magefile, go.mod, README, plugin metadata, and existing plan.
- Added adapted root AGENTS.md for codex-loop.
- Added `vet` target and included `go vet ./...` in `make verify`.
- Updated README development summary to mention vetting.
- Verified no AGH-specific strings remained in AGENTS.md.
- Verification passed: `make verify`.
Now:
- Prepare final summary.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- Target: AGENTS.md
- Source: /Users/pedronauck/dev/compozy/agh/AGENTS.md
- Commands read: rg --files, sed, find, go version, git status --short, git diff
- Commands run: make verify
- Edited: AGENTS.md, Makefile, magefile.go, README.md
