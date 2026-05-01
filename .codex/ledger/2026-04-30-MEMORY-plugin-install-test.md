Goal (incl. success criteria):
- Verify why codex-loop activation was not working in this Codex environment, compare with the original Python implementation, and restore the working behavior in code and local install state.
Constraints/Assumptions:
- Do not modify unrelated files or run destructive git commands.
- Prefer inspection and safe runtime checks before any install/refresh action.
Key decisions:
- Treat this as an environment validation task first, not an installation task.
- Treat the first Python commit as the canonical explanation for why activation used to work: managed config-layer hooks in `~/.codex/hooks.json`, not plugin-only hook dispatch.
- Keep `plugins/codex-loop/hooks/hooks.json` as the bundled hook definition, but mirror the same managed hook commands into `~/.codex/hooks.json` during `codex-loop install`.
State:
- completed
Done:
- Read project and skill instructions.
- Read other agent ledgers for cross-session context.
- Confirmed current Codex config has `features.codex_hooks = true`.
- Confirmed current Codex config has `[plugins."codex-loop@codex-loop-plugins"] enabled = true`.
- Confirmed the plugin cache exists under `~/.codex/plugins/cache/codex-loop-plugins/codex-loop/1.0.0/` with `.codex-plugin/plugin.json`, `hooks/hooks.json`, and the packaged skill.
- Confirmed the managed runtime exists under `~/.codex/codex-loop/` with `bin/codex-loop`, `config.toml`, and `loops/`.
- Confirmed `codex-loop` is on PATH and the PATH binary matches the managed runtime binary by SHA-256.
- Confirmed `codex-loop status --all` currently returns `[]`, meaning no active/completed loops are recorded right now.
- Ran a nested `codex exec --ephemeral --json` smoke test with `[[CODEX_LOOP ...]]`; the child session completed successfully but did not create a loop record.
- Confirmed the installed runtime can execute `hook user-prompt-submit` directly in an isolated temporary `CODEX_HOME` and create a valid active loop record.
- Confirmed the plugin hook command uses an absolute managed-runtime path (`${CODEX_HOME:-$HOME/.codex}/codex-loop/bin/codex-loop`), so this is not a shell `PATH` issue.
- Confirmed `~/.codex/hooks.json` is neutral (`{"hooks": {}}`) and there are no lingering `codex-timed-loop` references in `~/.codex/hooks.json` or `~/.codex/config.toml`.
- Observed that a real prompt in the current interactive Codex session using `[[CODEX_LOOP name="smoke" rounds="1"]]` still left `codex-loop status --all` as `[]`.
- Confirmed `codex-cli 0.128.0` is installed in this environment.
- Confirmed config-layer hooks still execute in fresh Codex processes by launching `codex exec` with an isolated `CODEX_HOME/hooks.json` that wrote a marker file.
- Inspected `~/.codex/log/codex-tui.log` and found no `UserPromptSubmit`, `Preparing the codex loop`, or `Evaluating the codex loop` entries for this session.
- Verified upstream evidence from the official `openai/codex` issue tracker that plugin-local `hooks/hooks.json` is currently not executed while config-layer `hooks.json` does execute (`openai/codex` issue `#16430`, opened 2026-04-01).
- Inspected the root commit `504f9e4` and confirmed the Python implementation installed managed hooks by merging a template into `~/.codex/hooks.json` and removing only managed registrations on uninstall.
- Ported that behavior to the Go installer:
  - `codex-loop install` now syncs managed hook registrations into `~/.codex/hooks.json`
  - `codex-loop uninstall` now removes only those managed registrations while preserving unrelated hooks
  - added installer regression tests and a parity test that the installer template matches `plugins/codex-loop/hooks/hooks.json`
- Updated README and plugin skill/docs to reflect the working install path.
- Verification passed: `make verify`.
- Ran an isolated end-to-end smoke test:
  - `install` preserved a custom hook while adding managed `Stop` and `UserPromptSubmit`
  - `hook user-prompt-submit` created an active loop record
  - `uninstall` removed only managed hook registrations and preserved the custom hook
- Installed the updated local binary with `go install ./cmd/codex-loop`.
- Ran `codex-loop install` against the real `~/.codex` and confirmed `~/.codex/hooks.json` now contains the managed `Stop` and `UserPromptSubmit` commands again.
Now:
- Report that the code fix and real install fix are complete; the remaining user action is to restart the running Codex desktop app so it reloads the updated `~/.codex/hooks.json`.
Next:
- After restart, re-run a live `[[CODEX_LOOP ...]]` prompt in the GUI session if the user wants one more end-to-end confirmation.
Open questions (UNCONFIRMED if needed):
- UNCONFIRMED: whether a future Codex release will restore plugin-local hook dispatch or whether the docs/plugin examples will be changed instead.
Working set (files/ids/commands):
- ~/.codex/config.toml
- ~/.codex/codex-loop/config.toml
- ~/.codex/hooks.json
- plugins/codex-loop/.codex-plugin/plugin.json
- plugins/codex-loop/hooks/hooks.json
- ~/.codex/plugins/cache/codex-loop-plugins/codex-loop/1.0.0/.codex-plugin/plugin.json
- ~/.codex/plugins/cache/codex-loop-plugins/codex-loop/1.0.0/hooks/hooks.json
- ~/.codex/log/codex-tui.log
- internal/installer/installer.go
- internal/installer/hooksjson.go
- internal/installer/installer_test.go
- README.md
- plugins/codex-loop/README.md
- plugins/codex-loop/skills/codex-loop/SKILL.md
- CODEX_HOME=<tmp> codex exec --dangerously-bypass-approvals-and-sandbox --json 'hello'
- make verify
- go install ./cmd/codex-loop
- codex-loop install
- codex-loop status --all
