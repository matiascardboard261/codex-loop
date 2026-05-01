Goal (incl. success criteria):
- Ensure the local `codex-loop` project from this checkout is installed globally on the user's machine.
- Success: the local CLI binary is globally invokable and `codex-loop install` has installed/refreshed the managed runtime under the user's Codex home.
- Follow-up success: remove the stale global hook error caused by legacy `~/.codex/hooks.json` entries that still referenced `codex-timed-loop`.
Constraints/Assumptions:
- Follow workspace restrictions: no destructive git commands; do not touch unrelated changes.
- Prefer the project's official install flow from README and plugin skill.
- User asked for machine-wide/global availability on this computer, so prefer installing from the local checkout instead of fetching a remote release.
Key decisions:
- The user-facing "global install" requires both the local Go binary on PATH and the managed runtime under `~/.codex/codex-loop/`.
- The Codex marketplace registration should point at this local checkout so the plugin can be installed from the repo rather than a remote source.
State:
- Completed; final response pending.
Done:
- Read root AGENTS.md and relevant project install docs.
- Read existing cross-agent ledgers for awareness.
- Read the local codex-loop skill and golang-pro guidance.
- Verified the checkout with `make verify`.
- Installed the local binary with `go install ./cmd/codex-loop`.
- Confirmed `codex-loop` is now on PATH at `/Users/pedronauck/.local/share/mise/installs/go/1.26.1/bin/codex-loop`.
- Ran `codex-loop install` and created/refreshed:
  - `/Users/pedronauck/.codex/codex-loop/bin/codex-loop`
  - `/Users/pedronauck/.codex/codex-loop/config.toml`
  - `/Users/pedronauck/.codex/codex-loop/loops/`
- Confirmed `features.codex_hooks` was already enabled in `/Users/pedronauck/.codex/config.toml`.
- Added the local Codex marketplace with `codex plugin marketplace add /Users/pedronauck/Dev/ai/codex-loop-plugin`.
- Confirmed marketplace persistence under `[marketplaces.codex-loop-plugins]` in `/Users/pedronauck/.codex/config.toml`.
- Confirmed the PATH binary and managed runtime binary have the same SHA-256.
- Enabled the plugin in `/Users/pedronauck/.codex/config.toml` with `[plugins."codex-loop@codex-loop-plugins"] enabled = true`.
- Confirmed a fresh `codex exec` session reported `hook: UserPromptSubmit`, consistent with the plugin hook being loaded in new Codex processes.
- Investigated the reported stop-hook error and confirmed it came from legacy global hooks in `/Users/pedronauck/.codex/hooks.json` still referencing missing `codex-timed-loop` Python scripts.
- Replaced `/Users/pedronauck/.codex/hooks.json` with a neutral empty hook config:
  - `{ "hooks": {} }`
- Validated with a fresh `codex exec --ephemeral` smoke test that a new Codex process exits cleanly without the legacy Python hook error.
- Confirmed no active `codex-timed-loop` references remain in `/Users/pedronauck/.codex/hooks.json` or `/Users/pedronauck/.codex/config.toml`.
- Re-checked after the user reported the same hook prompt again:
  - `/Users/pedronauck/.codex/hooks.json` still contains only `{ "hooks": {} }`
  - no `codex-timed-loop` references exist in `/Users/pedronauck/.codex/hooks.json` or `/Users/pedronauck/.codex/config.toml`
  - the running `/Applications/Codex.app` process started at `2026-04-30 20:56:58 -0300`
  - `/Users/pedronauck/.codex/hooks.json` was rewritten at `2026-04-30 21:00:59 -0300`
  - therefore the current GUI session is still using an in-memory copy of the old hook config loaded before the fix
- After the hook prompt appeared again, added legacy file-path compatibility wrappers so stale GUI sessions no longer fail:
  - `/Users/pedronauck/.codex/codex-timed-loop/hooks/loop_stop.py`
  - `/Users/pedronauck/.codex/codex-timed-loop/hooks/loop_user_prompt_submit.py`
- Those wrappers `execv` into `/Users/pedronauck/.codex/codex-loop/bin/codex-loop hook stop|user-prompt-submit`.
- Validated both wrappers directly with sample JSON payloads; both exit successfully with no Python file-not-found error.
- User explicitly requested full removal of the legacy path afterwards.
- Deleted `/Users/pedronauck/.codex/codex-timed-loop/` completely.
- Verified the legacy directory is gone, `/Users/pedronauck/.codex/hooks.json` remains neutral, and `/Users/pedronauck/.codex/codex-loop/` remains installed.
Now:
- Prepare the final handoff with the confirmed removal of the legacy directory.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- If the current Codex desktop session still has the old hook path loaded in memory, one more hook error may appear until the app is fully restarted.
Working set (files/ids/commands):
- `README.md`
- `plugins/codex-loop/README.md`
- `plugins/codex-loop/skills/codex-loop/SKILL.md`
- `.codex/ledger/2026-04-30-MEMORY-global-install.md`
- `/Users/pedronauck/.codex/config.toml`
- `/Users/pedronauck/.codex/hooks.json`
- `/Users/pedronauck/.codex/codex-loop/`
- Commands: `make verify`, `go install ./cmd/codex-loop`, `codex-loop install`, `codex plugin marketplace add /Users/pedronauck/Dev/ai/codex-loop-plugin`
