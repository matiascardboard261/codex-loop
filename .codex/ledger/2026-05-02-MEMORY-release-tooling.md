Goal (incl. success criteria):
- Finish release tooling for `codex-loop` using `/Users/pedronauck/dev/compozy/agh` as the reference, then update the official repository/module path to `github.com/compozy/codex-loop`.
- Success: CI, release, GoReleaser, and related project automation are configured consistently enough for a public release, all product repo references use `github.com/compozy/codex-loop`, and verification passes.
Constraints/Assumptions:
- Do not run destructive git commands (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) without explicit permission.
- Preserve unrelated user/agent changes.
- Go/tooling changes must follow `golang-pro` and final project gate should include `make verify`.
- Use local code/reference project first; external docs only if needed for current tooling behavior.
Key decisions:
- Adapt AGH tooling to this smaller OSS Go/plugin project instead of copying AGH's web/E2E/Homebrew/Linux package/release-PR complexity.
- Use standard OSS GoReleaser, not GoReleaser Pro.
- Publish on `v*` tags; pull requests and manual dispatch run dry-run snapshot builds.
- Include the Codex plugin bundle and local marketplace metadata in release archives alongside the CLI binary.
- Skip local snapshot signing/SBOM generation because those require CI-installed release tools; tagged GitHub releases still sign checksums and generate archive SBOMs.
State:
- Repo-path update complete and verified; final response pending.
Done:
- Scanned existing ledgers for cross-agent awareness.
- Read `golang-pro` and `context7` skill instructions.
- Confirmed current worktree initially has no visible `git status --short` output.
- Compared AGH Make/Mage, CI, release workflow, GoReleaser config, git-cliff config, release templates, and setup actions.
- Added GitHub CI workflow running `make verify`.
- Added tag-based GitHub release workflow with GoReleaser dry-run and publish jobs.
- Added reusable setup actions for Go, git-cliff, and release tools.
- Added `.goreleaser.yml`, release header/footer templates, and `cliff.toml`.
- Added `make release-check` and `make release-snapshot` Mage targets.
- Updated Mage build to inject version/commit/date ldflags and write `bin/codex-loop`.
- Updated README development/release tooling section.
- Verified GoReleaser schema with `make release-check`.
- Verified local snapshot archives with `make release-snapshot`; archives include CLI, README, LICENSE, plugin bundle, and marketplace metadata.
- Verified workflows with `go run github.com/rhysd/actionlint/cmd/actionlint@latest`.
- Final project gate passed: `make verify`.
- Replaced product module/import/doc/plugin/release references from the previous personal namespace to `github.com/compozy/codex-loop`.
- Updated GoReleaser release owner to `compozy`.
- Updated the accepted plan artifact under `.codex/plans/`.
- Left other agents' historical ledgers read-only; those are the only remaining previous-repo-path hits.
- Verified package list now reports `github.com/compozy/codex-loop/...`.
- Verification passed after repo path update: `make verify`, `make release-check`, `make release-snapshot`, and `go run github.com/rhysd/actionlint/cmd/actionlint@latest`.
Now:
- Prepare final handoff.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None currently.
Working set (files/ids/commands):
- Reference repo: `/Users/pedronauck/dev/compozy/agh`
- Target repo: `/Users/pedronauck/Dev/ai/codex-loop-plugin`
- Edited: `Makefile`, `magefile.go`, `README.md`, `.github/actions/setup-go/action.yml`, `.github/actions/setup-git-cliff/action.yml`, `.github/actions/setup-release/action.yml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.goreleaser.yml`, `.goreleaser.release-header.md.tmpl`, `.goreleaser.release-footer.md.tmpl`, `cliff.toml`
- Commands: `make release-check`, `make release-snapshot`, `make verify`, `go run github.com/rhysd/actionlint/cmd/actionlint@latest`
- Current repo path target: `github.com/compozy/codex-loop`
