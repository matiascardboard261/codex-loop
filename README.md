# codex-loop-plugin

Standalone marketplace repo for the `codex-timed-loop` Codex plugin.

## What this project contains

- `.agents/plugins/marketplace.json`: personal/global Codex marketplace definition.
- `plugins/codex-timed-loop/`: publishable plugin bundle.
- Global hook installer and uninstaller for `~/.codex`.
- Tests for activation parsing, loop behavior, and installer lifecycle.

## Install

1. Register this repo as a personal marketplace:

   ```bash
   codex plugin marketplace add /Users/pedronauck/dev/ai/codex-loop-plugin
   ```

2. Restart Codex.
3. Open `/plugins` and install `codex-timed-loop` from `Codex Loop Plugins`.
4. Run the global installer once:

   ```bash
   python3 /Users/pedronauck/dev/ai/codex-loop-plugin/plugins/codex-timed-loop/scripts/install.py
   ```

The installer writes the managed runtime into:

- `~/.codex/hooks.json`
- `~/.codex/config.toml`
- `~/.codex/codex-timed-loop/`

## Activation

Use one of these headers on the first line of the prompt:

```text
[[CODEX_LOOP name="release-stress-qa" min="6h"]]
Run release-grade QA for this repository and keep expanding scope until the minimum duration is met.
```

```text
[[CODEX_LOOP name="release-stress-qa" rounds="3"]]
Run three deliberate QA rounds for this repository.
```

Only one limiter is allowed per header:

- `min="..."`
- `rounds="..."`

Duration parsing accepts:

- `30m`
- `30min`
- `1h 30m`
- `2 hours`
- `45sec`

## Uninstall

```bash
python3 /Users/pedronauck/dev/ai/codex-loop-plugin/plugins/codex-timed-loop/scripts/uninstall.py
```

## Verify

```bash
python3 -m unittest discover -s /Users/pedronauck/dev/ai/codex-loop-plugin/plugins/codex-timed-loop/tests -v
```
