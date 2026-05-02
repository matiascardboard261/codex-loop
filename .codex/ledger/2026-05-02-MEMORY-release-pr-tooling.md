Goal (incl. success criteria):
- Implement accepted plan for release PR tooling using local `pr-release` and `looper` references.
- Success: pushes to `main` create/update release PRs, release PR dry-runs run, merges publish via GoReleaser, docs/tests updated, and verification passes.
Constraints/Assumptions:
- No destructive git commands.
- Accepted Plan Mode plan must be persisted under `.codex/plans/`.
- Go/test work uses `golang-pro`; final gate should include `make verify`.
- Use `github.com/compozy/releasepr@v0.0.21` and initial release `v0.1.0`.
- Keep OSS GoReleaser; no Pro/NPM/Homebrew/Docker/AUR additions.
Key decisions:
- Replace tag-only release publishing with `main -> release PR -> dry-run -> merge -> tag/publish`.
- Keep `RELEASE_BODY.md` as the file consumed by GoReleaser and `RELEASE_NOTES.md` as historical generated notes.
State:
- Implementation complete and verified; final response pending.
Done:
- Read existing session ledgers and `golang-pro` skill.
- Confirmed initial worktree had no visible `git status --short` output.
- Persisted accepted plan to `.codex/plans/2026-05-02-release-pr-tooling.md`.
- Replaced tag-only release workflow with release PR, dry-run, and merge-publish jobs.
- Pinned release PR orchestration to `github.com/compozy/releasepr@v0.0.21`.
- Set first release baseline to `v0.1.0` in workflow and `cliff.toml`.
- Kept standard OSS GoReleaser and `RELEASE_BODY.md` publishing.
- Added `.release-notes/.gitkeep` and README release PR documentation.
- Added release config regression tests under `internal/releaseconfig`.
- Fixed Makefile Mage fallback so broken `mage` shims do not block `make` targets.
- Verification passed: `go test ./...`, `go vet ./...`, `make lint`, `make release-check`, `make release-snapshot`, `make verify`, `actionlint`, `git diff --check`.
- Confirmed `git cliff --bumped-version` returns `v0.1.0` with no tags.
Now:
- Prepare final handoff.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- `.codex/plans/2026-05-02-release-pr-tooling.md`
- `.codex/ledger/2026-05-02-MEMORY-release-pr-tooling.md`
- Edited: `.github/workflows/release.yml`, `.goreleaser.yml`, `Makefile`, `README.md`, `cliff.toml`, `.release-notes/.gitkeep`, `internal/releaseconfig/releaseconfig_test.go`
- Commands: `go test ./...`, `go vet ./...`, `make lint`, `make release-check`, `make release-snapshot`, `make verify`, `go run github.com/rhysd/actionlint/cmd/actionlint@latest`, `git cliff --bumped-version`, `git diff --check`
