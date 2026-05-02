---
name: codex-loop
description: Install or refresh the Codex Loop runtime for structured [[CODEX_LOOP ...]] activation with either min="..." or rounds="...". Use only for codex-loop setup, status, uninstall, or activation guidance.
---

# Codex Loop

## Procedures

**Step 1: Inspect current global Codex state**
1. Read `~/.codex/config.toml` only if it exists so you can explain whether `features.codex_hooks` was already enabled.
2. Read `~/.codex/codex-loop/config.toml` only if it exists so you can describe optional continuation guidance and any `pre_loop_continue` command.
3. Do not hand-edit global hook files for normal setup; `codex-loop install` syncs the bundled hook commands into `~/.codex/hooks.json` while preserving unrelated hooks.

**Step 2: Install or refresh the runtime**
1. If `codex-loop` is not on `PATH`, install it:
   - `go install github.com/compozy/codex-loop/cmd/codex-loop@latest`
2. Execute:
   - `codex-loop install`
3. Read the command output and report the runtime path, loop state path, managed hook config path, and config path.
4. Tell the user to restart Codex so plugin lifecycle hooks and `features.codex_hooks` are reloaded.

**Step 3: Explain activation**
1. Tell the user that loop activation must be the first line of the prompt.
2. Tell the user that the header must contain exactly one limiter:
   - `[[CODEX_LOOP name="release-stress-qa" min="6h"]]`
   - `[[CODEX_LOOP name="release-stress-qa" rounds="3"]]`
3. Tell the user that the task prompt starts on the next line and remains the source task for every continuation.
4. Tell the user that loop state is persisted under `~/.codex/codex-loop/loops/`.

## Commands

- `codex-loop install`: install or refresh the local runtime under `~/.codex/codex-loop/` and sync managed hook registrations into `~/.codex/hooks.json`.
- `codex-loop status`: print active loop state as JSON.
- `codex-loop status --all`: include completed, superseded, and cut-short loops.
- `codex-loop uninstall`: remove the managed `~/.codex/codex-loop/` runtime directory and only the `codex-loop`-managed hook registrations.
- `[pre_loop_continue]`: optional codex-loop runtime hook that runs synchronously inside the Stop handler before an automatic continuation prompt is emitted.

## Error Handling

- If `go install` fails, confirm Go is installed and available on `PATH`.
- If `codex-loop install` fails while updating `~/.codex/config.toml`, inspect that file for malformed TOML or filesystem permission problems.
- If activation does nothing after install, restart Codex and confirm the `codex-loop` plugin is installed and enabled.
