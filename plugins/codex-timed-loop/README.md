# Codex Timed Loop Plugin

Global Codex plugin that installs managed hooks in `~/.codex` and keeps selected tasks running until they satisfy either a minimum duration or a target number of rounds.

## What it does

- Activates only from a structured prompt header on the first line:
  - `[[CODEX_LOOP name="release-stress-qa" min="6h"]]`
  - `[[CODEX_LOOP name="release-stress-qa" rounds="3"]]`
- Persists loop state under `~/.codex/codex-timed-loop/loops/`.
- Reopens work automatically on `Stop` until the chosen limit is met.
- Expands scope before continuing with hardening, edge cases, adjacent coverage, and stronger validation.
- Supports optional extra continuation guidance from `~/.codex/codex-timed-loop/config.toml`.

## Layout

- `.codex-plugin/plugin.json`: Codex plugin manifest.
- `skills/codex-timed-loop/SKILL.md`: bootstrap workflow used after plugin installation.
- `scripts/install.py`: installs managed global hook files under `~/.codex/`.
- `scripts/uninstall.py`: removes only the managed hook registrations and runtime files.
- `templates/hooks.json`: managed global hook registration template.
- `templates/config.toml`: optional continuation guidance template.
- `templates/hooks/*.py`: hook runtime implementation.
- `tests/*.py`: installer, uninstall, parser, and hook tests.

## Install

1. Register the marketplace:
   - `codex plugin marketplace add /Users/pedronauck/dev/ai/codex-loop-plugin`
2. Restart Codex.
3. Open `/plugins`.
4. Install `codex-timed-loop` from `Codex Loop Plugins`.
5. Run the bootstrap skill, or execute:
   - `python3 /Users/pedronauck/dev/ai/codex-loop-plugin/plugins/codex-timed-loop/scripts/install.py`

The installer creates or updates:

- `~/.codex/hooks.json`
- `~/.codex/config.toml` with `features.codex_hooks = true`
- `~/.codex/codex-timed-loop/hooks/*.py`
- `~/.codex/codex-timed-loop/loops/`
- `~/.codex/codex-timed-loop/config.toml`

The installer refuses to proceed when `~/.codex/config.toml` already contains inline `[hooks]` tables.

## Activation

The first line of the prompt must be the activation header:

```text
[[CODEX_LOOP name="release-stress-qa" min="6h"]]
Run release-grade QA for this repository and keep expanding scope until the minimum duration is met.
```

Or:

```text
[[CODEX_LOOP name="release-stress-qa" rounds="3"]]
Run three deliberate QA passes for this repository and treat each stop as the end of one round.
```

Rules:

- The header must be on the first line.
- The task prompt must start on the next line.
- The header must contain exactly one limiter: `min="..."` or `rounds="..."`.
- `rounds` must be a positive integer.
- Duration parsing accepts compact or human-friendly forms such as `30m`, `30min`, `1h 30m`, `2 hours`, and `45sec`.
- The loop is isolated by Codex `session_id`, not by repository or plugin installation.

## Parallel sessions

- Multiple Codex conversations in the same repository can coexist.
- Only sessions that send the activation header create a loop.
- Each active loop is scoped to its own `session_id`.
- Re-activating the loop in the same conversation supersedes the previous active loop for that conversation.

## Optional continuation guidance

`~/.codex/codex-timed-loop/config.toml` supports:

```toml
optional_skill_name = ""
optional_skill_path = ""
extra_continuation_guidance = ""
```

- `optional_skill_name` and `optional_skill_path` are used only when the path resolves inside the current workspace.
- `optional_skill_path` may point to a skill directory or directly to `SKILL.md`.
- `extra_continuation_guidance` appends extra text to every automatic continuation.

## Uninstall

Run:

```bash
python3 /Users/pedronauck/dev/ai/codex-loop-plugin/plugins/codex-timed-loop/scripts/uninstall.py
```

This removes only the managed hook registrations and the runtime directory `~/.codex/codex-timed-loop/`. It leaves other global hooks and `features.codex_hooks` unchanged.

## Verification

Run:

```bash
python3 -m unittest discover -s /Users/pedronauck/dev/ai/codex-loop-plugin/plugins/codex-timed-loop/tests -v
```
