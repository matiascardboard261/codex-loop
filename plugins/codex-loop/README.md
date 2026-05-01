# Codex Loop Plugin

Codex plugin bundle for the `codex-loop` CLI.

The plugin contributes:

- `skills/codex-loop/SKILL.md`: setup and activation workflow.
- `hooks/hooks.json`: Codex lifecycle hooks for `UserPromptSubmit` and `Stop`.

The hook commands call the runtime binary installed by:

```bash
go install github.com/pedronauck/codex-loop/cmd/codex-loop@latest
codex-loop install
```

`codex-loop install` also mirrors those managed hook registrations into `~/.codex/hooks.json` so current Codex builds execute them reliably without overwriting unrelated user hooks.

After installing or updating the plugin or runtime, restart Codex.
