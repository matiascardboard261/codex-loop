# Codex Loop Plugin

Codex plugin bundle for the `codex-loop` CLI.

The plugin contributes:

- `skills/codex-loop/SKILL.md`: setup and activation workflow.
- `hooks/hooks.json`: Codex lifecycle hooks for `UserPromptSubmit` and `Stop`.

The hook commands call the runtime binary installed by:

```bash
go install github.com/compozy/codex-loop/cmd/codex-loop@latest
codex-loop install
```

Existing installs can be updated with:

```bash
codex-loop upgrade              # latest GitHub release
codex-loop upgrade --version v0.1.1
```

`codex-loop install` also mirrors those managed hook registrations into `~/.codex/hooks.json` so current Codex builds execute them reliably without overwriting unrelated user hooks. The bundled Stop hook default timeout is 2700 seconds so goal confirmation can run slow reasoning models; user-specific timeout changes live in `~/.codex/codex-loop/config.toml` and require rerunning `codex-loop install`.

Activation supports exactly one limiter:

```text
[[CODEX_LOOP name="qa" min="6h"]]
[[CODEX_LOOP name="qa" rounds="3"]]
[[CODEX_LOOP name="qa" goal="finish only when verified"]]
```

Goal loops confirm completion with a configurable headless command that returns normal text. The default confirmation command is `codex exec --yolo`, `gpt-5.5`, and reasoning effort `high`; custom runners can be configured with `[goal].confirm_command` as a shell-like string that codex-loop parses to argv before direct execution. codex-loop then privately interprets the text with a fixed `codex exec --output-schema` step; the default interpreter uses `gpt-5.4-mini` and reasoning effort `low`.

After installing or updating the plugin or runtime, restart Codex.
