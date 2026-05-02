Goal (incl. success criteria):
- Replace the currently installed codex-loop plugin/runtime under ~/.codex with the published GitHub release v0.1.0 from https://github.com/compozy/codex-loop/releases/tag/v0.1.0.
- Success: old installed plugin/runtime removed or superseded safely, v0.1.0 release installed into ~/.codex, install verified without touching unrelated user state.
Constraints/Assumptions:
- Follow AGENTS.md: no destructive git commands; preserve unrelated ~/.codex configuration; do not manually edit global hook files as primary integration path.
- Current date: 2026-05-02.
- Preserve existing ~/.codex/codex-loop/loops state unless the released installer requires migration.
Key decisions:
- Used codex-loop skill guidance for runtime install/update workflow.
- Installed from GitHub release v0.1.0 darwin_arm64 asset and verified checksum.
- Re-added Codex marketplace from https://github.com/compozy/codex-loop.git ref v0.1.0.
- Updated active Go 1.26.2 PATH binary with the same verified release binary so manual codex-loop resolves to v0.1.0.
State:
- Completed.
Done:
- Read codex-loop skill guidance.
- Scanned existing ledgers.
- Created this session ledger.
- Inspected ~/.codex/config.toml, ~/.codex/hooks.json, plugin cache, runtime config, and release API.
- Confirmed release v0.1.0 was published 2026-05-02T16:40:36Z and includes darwin_arm64 asset.
- Confirmed current runtime had one active loop record and preserved it.
- Downloaded checksums.txt and codex-loop_0.1.0_darwin_arm64.tar.gz; shasum verification passed.
- Moved stale ~/.codex/plugins/cache/codex-loop-plugins to backup outside ~/.codex.
- Ran released binary install; installed ~/.codex/codex-loop/bin/codex-loop and updated managed ~/.codex/hooks.json.
- Ran codex plugin marketplace add compozy/codex-loop --ref v0.1.0 and upgrade codex-loop-plugins.
- Verified ~/.codex/config.toml has [marketplaces.codex-loop-plugins] source=https://github.com/compozy/codex-loop.git ref=v0.1.0 and [plugins."codex-loop@codex-loop-plugins"] enabled=true.
- Verified new plugin cache metadata points to github.com/compozy/codex-loop and includes goal-mode skill docs.
- Verified ~/.codex/codex-loop/bin/codex-loop version is 0.1.0 (commit=770a9ae date=2026-05-02T16:39:39Z).
- Verified PATH codex-loop resolves to /Users/pedronauck/.local/share/mise/installs/go/1.26.2/bin/codex-loop and matches managed runtime SHA-256.
- Verified status --all still shows the existing improve-docs active loop.
- Verified no current ~/.codex config/hooks/plugin/runtime hits for github.com/pedronauck/codex-loop.
Now:
- Final handoff.
Next:
- Restart Codex so the running app reloads plugin marketplace and hooks.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- .codex/ledger/2026-05-02-MEMORY-install-release.md
- ~/.codex/config.toml
- ~/.codex/hooks.json
- ~/.codex/plugins/cache/codex-loop-plugins/
- ~/.codex/.tmp/marketplaces/codex-loop-plugins/
- ~/.codex/codex-loop/
- /Users/pedronauck/.local/share/mise/installs/go/1.26.2/bin/codex-loop
- Backups: /tmp/codex-loop-upgrade-20260502T134649
- Release temp: /tmp/codex-loop-release.zJ9y1w
