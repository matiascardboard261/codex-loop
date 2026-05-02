# codex-loop

`codex-loop` is a local-first Codex CLI and plugin that keeps explicitly activated Codex tasks running until they satisfy either a minimum duration or a target number of deliberate rounds.

It is designed for release-grade QA, long-running hardening passes, and repeated review loops where stopping after the first apparently complete answer is not enough.

## What It Does

- Activates only from a structured prompt header on the first line.
- Supports time limits such as `min="6h"` and round limits such as `rounds="3"`.
- Persists loop state under `~/.codex/codex-loop/loops/`.
- Defines Codex lifecycle hooks in the plugin bundle and mirrors managed registrations into `~/.codex/hooks.json` during install.
- Runs as a Go CLI with no Python runtime dependency.
- Stores all state locally in the user's Codex home directory.

## Install CLI

```bash
go install github.com/compozy/codex-loop/cmd/codex-loop@latest
codex-loop install
```

`codex-loop install` creates or updates:

- `~/.codex/codex-loop/bin/codex-loop`
- `~/.codex/codex-loop/loops/`
- `~/.codex/codex-loop/config.toml`
- `~/.codex/hooks.json` with managed `UserPromptSubmit` and `Stop` hook registrations
- `~/.codex/config.toml` with `features.codex_hooks = true`

## Install Plugin

Register this repo as a Codex plugin marketplace:

```bash
codex plugin marketplace add github.com/compozy/codex-loop
```

For local development from this checkout:

```bash
codex plugin marketplace add /path/to/codex-loop
```

Then restart Codex, open the plugin directory, and install `codex-loop` from the `Codex Loop Plugins` marketplace.

The plugin contributes lifecycle hooks from `plugins/codex-loop/hooks/hooks.json`. `codex-loop install` mirrors the same managed hook commands into `~/.codex/hooks.json` so current Codex builds execute them reliably while preserving unrelated user hooks.

## Activation

The activation header must be the first line of the prompt.

Minimum-duration loop:

```text
[[CODEX_LOOP name="release-stress-qa" min="6h"]]
Run release-grade QA for this repository and keep expanding scope until the minimum duration is met.
```

Rounds-based loop:

```text
[[CODEX_LOOP name="release-stress-qa" rounds="3"]]
Run three deliberate QA passes for this repository and treat each stop as the end of one round.
```

Rules:

- The task prompt starts on the line after the header.
- `name="..."` is required.
- Exactly one limiter is required: `min="..."` or `rounds="..."`.
- `rounds` must be a positive integer.
- Duration parsing accepts forms such as `30m`, `30min`, `1h 30m`, `2 hours`, and `45sec`.
- Loop state is isolated by Codex `session_id`.

## Commands

```bash
codex-loop install
codex-loop status
codex-loop status --all
codex-loop status --session-id <id>
codex-loop status --workspace-root <path>
codex-loop uninstall
codex-loop version
```

`status` prints JSON. By default it shows only active loops; use `--all` to include completed, superseded, and cut-short loops.

## Optional Continuation Guidance

`~/.codex/codex-loop/config.toml` supports:

```toml
optional_skill_name = ""
optional_skill_path = ""
extra_continuation_guidance = ""
```

- `optional_skill_name` and `optional_skill_path` are used only when the path resolves inside the active workspace.
- `optional_skill_path` may point to a skill directory or directly to `SKILL.md`.
- `extra_continuation_guidance` appends extra text to every automatic continuation.

## Uninstall

```bash
codex-loop uninstall
```

This removes only `~/.codex/codex-loop/`. It leaves `~/.codex/config.toml` and Codex plugin install state unchanged.
It also removes only the `codex-loop`-managed hook registrations from `~/.codex/hooks.json`, preserving unrelated user hooks.

## Development

```bash
make deps
make verify
```

`make verify` runs formatting, vetting, linting, race-enabled tests, and build checks.

Release tooling mirrors the project CI:

```bash
make release-check
make release-snapshot
```

- `make release-check` validates `.goreleaser.yml` with the current GoReleaser v2 CLI.
- `make release-snapshot` builds local snapshot artifacts under `dist/` without publishing, signing, or SBOM generation.
- GitHub Actions runs `make verify` on pushes and pull requests to `main`.
- Pushing a `v*` tag publishes a GitHub release through GoReleaser.

## Privacy

`codex-loop` does not send data to a network service. It reads Codex hook JSON from stdin, writes hook decisions to stdout, and stores loop state locally under `~/.codex/codex-loop/`.
