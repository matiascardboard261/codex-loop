<div align="center">
  <h1>Codex Loop</h1>
  <p><strong>Keep Codex tasks running until they actually finish — by time, rounds, or independently confirmed goal.</strong></p>
  <p>
    <a href="https://github.com/compozy/codex-loop/actions/workflows/ci.yml">
      <img src="https://github.com/compozy/codex-loop/actions/workflows/ci.yml/badge.svg" alt="CI">
    </a>
    <a href="https://pkg.go.dev/github.com/compozy/codex-loop">
      <img src="https://pkg.go.dev/badge/github.com/compozy/codex-loop.svg" alt="Go Reference">
    </a>
    <a href="https://goreportcard.com/report/github.com/compozy/codex-loop">
      <img src="https://goreportcard.com/badge/github.com/compozy/codex-loop" alt="Go Report Card">
    </a>
    <a href="LICENSE">
      <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT">
    </a>
    <a href="https://github.com/compozy/codex-loop/releases">
      <img src="https://img.shields.io/github/v/release/compozy/codex-loop?include_prereleases" alt="Release">
    </a>
  </p>
</div>

`codex-loop` is a local-first Codex CLI and plugin that keeps explicitly activated Codex tasks running until they satisfy a minimum duration, a target number of deliberate rounds, or an independently confirmed goal.

It is designed for release-grade QA, long-running hardening passes, and repeated review loops where stopping after the first apparently complete answer is not enough.

## ✨ Highlights

- **Three loop modes.** Pick the limiter that fits the work: minimum duration (`min="6h"`), deliberate round count (`rounds="3"`), or independently confirmed goal (`goal="ship only after verification"`).
- **Activation by header, not flag.** Loops only start when the prompt's first line contains a structured `[[CODEX_LOOP ...]]` header, so day-to-day Codex use is untouched.
- **Independent goal confirmation.** Goal loops invoke a configurable headless reviewer that returns normal text, then codex-loop privately interprets that text into a structured verdict.
- **Pre-continuation context hook.** `pre_loop_continue` runs right before each automatic continuation so the next prompt can carry fresh local context — test summaries, changed files, build status, custom checklists.
- **Codex lifecycle integration.** Ships as a Codex plugin, contributing `UserPromptSubmit` and `Stop` hooks, and mirrors managed registrations into `~/.codex/hooks.json` for current Codex builds.
- **Local-first state.** Loop state lives under `~/.codex/codex-loop/`, isolated by Codex `session_id`. Compact verdict metadata lands in `~/.codex/codex-loop/runs.jsonl`.
- **Single Go binary.** No Python runtime, no daemon, no network calls from `codex-loop` itself.

## 📦 Installation

#### Go

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

#### Codex Plugin

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

## 🚀 Activation

The activation header must be the first line of the prompt. The task prompt starts on the line after the header.

#### Minimum-duration loop

```text
[[CODEX_LOOP name="release-stress-qa" min="6h"]]
Run release-grade QA for this repository and keep expanding scope until the minimum duration is met.
```

#### Rounds-based loop

```text
[[CODEX_LOOP name="release-stress-qa" rounds="3"]]
Run three deliberate QA passes for this repository and treat each stop as the end of one round.
```

#### Goal-based loop

```text
[[CODEX_LOOP name="release-stress-qa" goal="ship only after real verification"]]
Run release-grade QA for this repository and keep going until the work is actually complete.
```

#### Goal-based loop with custom confirmation model

```text
[[CODEX_LOOP name="release-stress-qa" goal="ship only after real verification" confirm_model="gpt-5.5" confirm_reasoning_effort="xhigh"]]
Run release-grade QA for this repository and keep going until the work is actually complete.
```

#### Header rules

| Field                       | Required                | Notes                                                                       |
| --------------------------- | ----------------------- | --------------------------------------------------------------------------- |
| `name`                      | yes                     | Loop identifier                                                             |
| `min`                       | one of                  | Duration: `30m`, `30min`, `1h 30m`, `2 hours`, `45sec`, etc.                |
| `rounds`                    | one of                  | Positive integer                                                            |
| `goal`                      | one of                  | Free-form verification goal; `goal=""` reuses the task prompt as the goal   |
| `confirm_model`             | only with `goal`        | Model for the goal confirmation run                                         |
| `confirm_reasoning_effort`  | only with `goal`        | One of `minimal`, `low`, `medium`, `high`, `xhigh`                          |

Loop state is isolated by Codex `session_id`. Exactly one limiter — `min`, `rounds`, or `goal` — is required.

## 🧰 Commands

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

## ⚙️ Continuation Customization

`~/.codex/codex-loop/config.toml` supports:

```toml
optional_skill_name = ""
optional_skill_path = ""
extra_continuation_guidance = ""

[hooks]
stop_timeout_seconds = 2700

[goal]
confirm_model = "gpt-5.5"
confirm_reasoning_effort = "high"
confirm_command = "codex exec --cd $WORKSPACE_ROOT --ephemeral --yolo --output-last-message $CONFIRM_OUTPUT_PATH $MODEL_ARGV $REASONING_ARGV --skip-git-repo-check -"
timeout_seconds = 2400
interpret_model = "gpt-5.4-mini"
interpret_reasoning_effort = "low"
interpret_timeout_seconds = 120
max_output_bytes = 12000

[pre_loop_continue]
command = ""
cwd = "session_cwd"
timeout_seconds = 60
max_output_bytes = 12000
```

- `optional_skill_name` and `optional_skill_path` are used only when the path resolves inside the active workspace.
- `optional_skill_path` may point to a skill directory or directly to `SKILL.md`.
- `extra_continuation_guidance` appends extra text to every automatic continuation.
- `hooks.stop_timeout_seconds` controls the managed Codex `Stop` hook timeout written by `codex-loop install`; rerun `codex-loop install` and restart Codex after changing it.

### 🎯 Goal Confirmation

Goal loops run a configurable headless confirmation command inside the `Stop` hook before deciding whether to continue. The public confirmation command returns normal text. codex-loop then runs a private interpreter command that converts that text into the internal verdict JSON.

The default confirmation run uses `codex exec --yolo`, `gpt-5.5`, and `model_reasoning_effort = "high"`. Model and reasoning are separate settings; use `confirm_model = "gpt-5.5"` plus `confirm_reasoning_effort = "xhigh"` rather than a combined model string. Codex documents `--yolo` as full access without sandboxing or approvals; use a custom `confirm_command` when you need a different safety profile or runner.

The interpreter always uses `codex exec --sandbox read-only --output-schema`. Its model defaults to `gpt-5.4-mini` and reasoning effort defaults to `low`; users can change only the interpreter model, reasoning effort, and timeout. The interpreter command itself is intentionally not configurable so codex-loop can rely on Codex structured output and the user's existing Codex auth.

**Runtime behavior:**

- `confirm_command` is a shell-like string parsed to argv and executed without an implicit shell.
- Placeholder values are shell-quoted before parsing so injected values remain literal arguments.
- The default confirmation command runs with `--yolo`, `--ephemeral`, and `--output-last-message`; it does not receive an output schema.
- Custom confirmation commands may write normal prose to `$CONFIRM_OUTPUT_PATH` or stdout.
- The interpreter command is fixed to `codex exec` and produces the structured verdict internally.
- If the interpreted verdict is complete, codex-loop marks the loop completed and emits no continuation prompt.
- If the interpreted verdict is incomplete, invalid, timed out, or either command fails, codex-loop continues with a warning and keeps the loop active.
- `goal.timeout_seconds` controls the confirmation timeout; `goal.interpret_timeout_seconds` controls the interpreter timeout. Both are normalized to leave time before the outer Stop hook timeout.
- `goal.max_output_bytes` caps captured confirmation and interpreter output used for prompts and diagnostics.
- Each confirmation attempt appends compact metadata to `~/.codex/codex-loop/runs.jsonl`.

**Confirmation command variables:**

- `$PROMPT`, `$PROMPT_FILE`, and `$CONFIRM_OUTPUT_PATH` expose the reviewer prompt and plain-text output file.
- `$MODEL_ARGV` expands to `--model <model>` when a model is configured; `$REASONING_ARGV` expands to `--config model_reasoning_effort="<effort>"` when reasoning is configured.
- `$MODEL`, `$REASONING_EFFORT`, `$WORKSPACE_ROOT`, `$CWD`, `$SESSION_ID`, `$LOOP_NAME`, `$LOOP_SLUG`, `$RUNS_LOG_PATH`, and `$CODEX_HOME` are also available.
- The same values are exported with the `CODEX_LOOP_CONFIRM_` prefix.

**Custom runner example:**

```toml
[goal]
confirm_model = "opus"
confirm_reasoning_effort = ""
confirm_command = "compozy exec --ide claude --model $MODEL $PROMPT"
```

### 🪝 `pre_loop_continue`

`pre_loop_continue` is a codex-loop runtime hook that runs inside the managed `Stop` hook, immediately before codex-loop asks Codex to continue the active loop. For goal loops, it runs only after goal confirmation decides another round is needed. It is not a separate Codex lifecycle hook, which matters because independent Codex `Stop` hooks may run concurrently and cannot guarantee ordering.

Use it when the next continuation prompt should include fresh local context computed at stop time, such as a test summary, changed-file summary, issue tracker snapshot, custom QA checklist, or local build status.

**Runtime behavior:**

- It runs only when codex-loop has decided to continue the task.
- It does not run when there is no active loop, when the loop has completed, or when codex-loop cuts the loop short.
- `command` is a shell-like string parsed to argv and executed without an implicit shell. Point it at a script, or explicitly use `bash -lc '...'`, if you need shell features such as pipes or redirects.
- `cwd = "session_cwd"` runs from the same directory Codex reported in the `Stop` hook payload. This is the default.
- `cwd = "workspace_root"` runs from the root codex-loop resolved from `.git` or `.codex`.
- The command receives structured JSON on stdin.
- The same JSON is available through `$INPUT_JSON` and `$INPUT_FILE`.
- Only stdout is appended to the next continuation prompt under `pre_loop_continue output:`.
- Stderr is not injected into the prompt. Failures and timeouts keep the loop running and append a short `pre_loop_continue warning:` instead.
- `max_output_bytes` caps captured stdout before prompt injection.

**Example config:**

```toml
[pre_loop_continue]
command = ".codex/scripts/loop-context.sh --format markdown --input $INPUT_FILE"
cwd = "session_cwd"
timeout_seconds = 30
max_output_bytes = 8000
```

**Example script:**

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

## 🗑️ Uninstall

```bash
codex-loop uninstall
```

This removes only `~/.codex/codex-loop/`. It leaves `~/.codex/config.toml` and Codex plugin install state unchanged. It also removes only the `codex-loop`-managed hook registrations from `~/.codex/hooks.json`, preserving unrelated user hooks.

## 🛠️ Development

```bash
make deps        # Tidy and verify modules
make verify      # Full pipeline: fmt → vet → lint → race tests → build
```

Release tooling mirrors the project CI:

```bash
make release-check       # Validate .goreleaser.yml with current GoReleaser v2 CLI
make release-snapshot    # Build local snapshot artifacts under dist/ (no publish/sign/SBOM)
```

- GitHub Actions runs `make verify` on pushes and pull requests to `main`.
- Pushing normal changes to `main` runs the release workflow in release-PR mode. It uses `github.com/compozy/releasepr@v0.0.21` to calculate the next semantic version, generate `CHANGELOG.md`, generate the current `RELEASE_BODY.md`, update historical `RELEASE_NOTES.md`, and open or update a `release/vX.Y.Z` pull request.
- Release pull requests run CI plus a GoReleaser dry-run check before merge.
- Merging a release pull request to `main` creates the `vX.Y.Z` tag and publishes the GitHub release through GoReleaser using `RELEASE_BODY.md`.
- The release workflow requires a `RELEASE_TOKEN` secret with permission to push release branches/tags, open pull requests, and dispatch workflows.

Manual release notes can be staged before the release PR is generated:

```bash
go run github.com/compozy/releasepr@v0.0.21 add-note --title "Short title" --type feature
```

Generated note files live in `.release-notes/`. The release PR archives consumed notes under `.release-notes/archive/vX.Y.Z/` and keeps `.release-notes/.gitkeep` in place for future notes.

## 🔒 Privacy

`codex-loop` itself does not send data to a network service. It reads Codex hook JSON from stdin, writes hook decisions to stdout, and stores loop state locally under `~/.codex/codex-loop/`.

Goal loops intentionally invoke the local `codex exec` command for default confirmation and interpretation, which uses the user's configured Codex provider/auth. The local JSONL log stores compact verdict metadata only; it does not store the full original task prompt, full assistant message, confirmation prompt, confirmation review text, interpreter prompt, or `pre_loop_continue` output.

## 📄 License

[MIT](LICENSE)
