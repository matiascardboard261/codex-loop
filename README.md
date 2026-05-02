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

## Continuation Customization

`~/.codex/codex-loop/config.toml` supports:

```toml
optional_skill_name = ""
optional_skill_path = ""
extra_continuation_guidance = ""

[pre_loop_continue]
command = ""
args = []
cwd = "session_cwd"
timeout_seconds = 60
max_output_bytes = 12000
```

- `optional_skill_name` and `optional_skill_path` are used only when the path resolves inside the active workspace.
- `optional_skill_path` may point to a skill directory or directly to `SKILL.md`.
- `extra_continuation_guidance` appends extra text to every automatic continuation.

### `pre_loop_continue`

`pre_loop_continue` is a codex-loop runtime hook that runs inside the managed `Stop` hook, immediately before codex-loop asks Codex to continue the active loop. It is not a separate Codex lifecycle hook, which matters because independent Codex `Stop` hooks may run concurrently and cannot guarantee ordering.

Use it when the next continuation prompt should include fresh local context computed at stop time, such as a test summary, changed-file summary, issue tracker snapshot, custom QA checklist, or local build status.

Runtime behavior:

- It runs only when codex-loop has decided to continue the task.
- It does not run when there is no active loop, when the loop has completed, or when codex-loop cuts the loop short.
- `command` and `args` execute without an implicit shell. Point `command` at a script if you need shell features such as pipes or redirects.
- `cwd = "session_cwd"` runs from the same directory Codex reported in the `Stop` hook payload. This is the default.
- `cwd = "workspace_root"` runs from the root codex-loop resolved from `.git` or `.codex`.
- The command receives structured JSON on stdin.
- Only stdout is appended to the next continuation prompt under `pre_loop_continue output:`.
- Stderr is not injected into the prompt. Failures and timeouts keep the loop running and append a short `pre_loop_continue warning:` instead.
- `max_output_bytes` caps captured stdout before prompt injection.

Example config:

```toml
[pre_loop_continue]
command = ".codex/scripts/loop-context.sh"
args = ["--format", "markdown"]
cwd = "session_cwd"
timeout_seconds = 30
max_output_bytes = 8000
```

Example script:

```bash
#!/usr/bin/env bash
set -euo pipefail

payload="$(cat)"
session_id="$(printf '%s' "$payload" | jq -r '.session_id')"
round="$(printf '%s' "$payload" | jq -r '.loop.continue_count')"

printf 'Session: %s\n' "$session_id"
printf 'Continuation round: %s\n\n' "$round"

printf 'Changed files:\n'
git status --short

printf '\nRecent test signal:\n'
if [ -f .codex/last-test.log ]; then
  tail -n 40 .codex/last-test.log
else
  printf 'No .codex/last-test.log found.\n'
fi
```

With that config, every automatic continuation prompt will include the script stdout after codex-loop's normal continuation instructions. If the script exits non-zero or exceeds `timeout_seconds`, codex-loop still continues the loop and injects a short warning instead of the script output.

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
