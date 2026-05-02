# Accepted Plan: Release PR Tooling

Accepted on 2026-05-02 after the user asked to implement the proposed plan.

## Summary
- Replace the current tag-only release workflow with a `pr-release` workflow that opens or updates release PRs from `main`, validates release PRs with dry-run checks, and publishes after the release PR is merged.
- Pin release PR orchestration to `github.com/compozy/releasepr@v0.0.21`, matching the healthy local `looper` setup and latest local `pr-release` tag.
- Keep `codex-loop` on standard OSS GoReleaser; do not add NPM, Homebrew, Docker, AUR, or GoReleaser Pro requirements.
- Use `v0.1.0` as the first public release baseline.

## Key Changes
- Update `.github/workflows/release.yml` with three jobs:
  - `release-pr`: runs on eligible pushes to `main` and manual `workflow_dispatch` in `release-pr` mode, invokes `go run "${PR_RELEASE_MODULE}" pr-release --enable-rollback --ci-output`, and dispatches CI plus release dry-run checks for the generated `release/vX.Y.Z` PR.
  - `dry-run`: runs for release PRs and manual `dry-run` dispatches, checks out the release branch, installs release tools, and invokes `go run "${PR_RELEASE_MODULE}" dry-run --ci-output`.
  - `release`: runs only when a release commit lands on `main`, creates and pushes the version tag, then publishes with GoReleaser using `RELEASE_BODY.md`.
- Add workflow inputs `mode`, `force_release`, `head_ref`, and `pr_number`, plus permissions required for branch/PR/tag creation and workflow dispatch.
- Set workflow env `GO_VERSION=1.26.2`, `INITIAL_VERSION=v0.1.0`, and `PR_RELEASE_MODULE=github.com/compozy/releasepr@v0.0.21`.
- Align `cliff.toml` and `.goreleaser.yml` with release automation commits and first-release behavior.
- Add `.release-notes/.gitkeep` for optional manual release notes.
- Update README release docs to describe the release PR flow, generated release files, required `RELEASE_TOKEN`, and validation commands.
- Add release tooling regression tests that assert the workflow/config contract.

## Test Plan
- Run focused checks:
  - `go test ./...`
  - `go vet ./...`
  - `make lint`
  - `make release-check`
  - `make release-snapshot`
  - `go run github.com/rhysd/actionlint/cmd/actionlint@latest`
  - `git cliff --bumped-version`
- Run final gate:
  - `make verify`
  - `git diff --check`

## Assumptions
- The GitHub repo has a `RELEASE_TOKEN` secret with permission to push release branches/tags, open PRs, and dispatch workflows.
- `codex-loop` remains a Go CLI plus Codex plugin release only.
- No destructive git commands are needed.
