Goal (incl. success criteria):
- Implement accepted plan for goal-based codex-loop condition and custom argv-based confirmation commands.
- Success: `goal` activation runs configurable headless confirmation until completion, supports configurable model/reasoning/timeouts, logs goal checks to JSONL, coexists with argv-based `pre_loop_continue`, updates docs/tests, and passes `make verify`.
Constraints/Assumptions:
- No destructive git commands.
- Use `golang-pro` for Go implementation.
- Accepted plan must be persisted under `.codex/plans/`.
- Default confirmation model is `gpt-5.5`; default reasoning effort is `high`.
- Stop hook timeout default is 2700s; nested goal confirmation timeout default is 2400s.
Key decisions:
- `goal` is a third exclusive limiter; no combined min/rounds mode.
- `confirm_model` and `confirm_reasoning_effort` are separate fields.
- Goal confirmation failures continue with warning and JSONL evidence.
- JSONL logs verdict metadata only, not full prompts/messages.
- Replace `[goal].codex_command` with `[goal].confirm_command` argv arrays.
- Replace `pre_loop_continue.command + args` with `pre_loop_continue.command` argv arrays.
- Default goal confirmation command uses `codex exec --yolo`.
State:
- Implementation complete and verified.
Done:
- Read cross-agent ledgers.
- Read `golang-pro` skill.
- Confirmed current worktree had no visible `git status --short` output.
- Persisted accepted plan under `.codex/plans/2026-05-02-goal-loop.md`.
- Implemented `goal` as a third exclusive limiter with optional `goal`, `confirm_model`, and `confirm_reasoning_effort` activation fields.
- Added runtime config defaults and TOML parsing for `[hooks] stop_timeout_seconds` and `[goal] confirm_model`, `confirm_reasoning_effort`, `codex_command`, `timeout_seconds`, and `max_output_bytes`.
- Added headless Codex goal confirmation before `pre_loop_continue`, with structured JSON verdicts, timeout handling, and continuation-on-warning behavior.
- Added JSONL run observability under the codex-loop runtime directory.
- Updated installer-managed hook timeout generation and plugin hook metadata.
- Updated README, plugin README, and plugin skill docs.
- Added parser/config/hook/installer regression tests for goal mode, timeout configuration, model/reasoning separation, logging, and `pre_loop_continue` coexistence.
- Verification passed: `go test ./...`, `go vet ./...`, `make lint`, `make verify`, and `git diff --check`.
- Confirmed official Codex config docs define `model` separately from `model_reasoning_effort` (`minimal|low|medium|high|xhigh`), matching the implementation.
- Persisted accepted confirm-command plan under `.codex/plans/2026-05-02-confirm-command-argv.md`.
- Replaced `[goal].codex_command` with argv-based `[goal].confirm_command`.
- Replaced `pre_loop_continue.command + args` with argv-based `pre_loop_continue.command`.
- Added shared argv expansion with `$VAR`, `${VAR}`, `$MODEL_ARGV`, `$REASONING_ARGV`, temp prompt/input files, and prefixed environment variables.
- Updated the default goal confirmation command to `codex exec --yolo`.
- Updated docs and tests for custom goal runners, stdout verdict fallback, and argv-based pre-loop commands.
- Verification passed again: `go test ./...`, `go vet ./...`, `make lint`, `make verify`, and `git diff --check`.
Now:
- Prepare final handoff.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- `.codex/plans/2026-05-02-goal-loop.md`
- `.codex/plans/2026-05-02-confirm-command-argv.md`
- `.codex/ledger/2026-05-02-MEMORY-goal-loop.md`
- `internal/loop/*`
- `internal/installer/*`
- `plugins/codex-loop/*`
- `README.md`
- Commands: `go test ./...`, `go vet ./...`, `make lint`, `make verify`, `git diff --check`
- Docs: `https://developers.openai.com/codex/config-reference#configtoml`, `https://developers.openai.com/codex/cli/reference#codex-exec`, `https://developers.openai.com/codex/agent-approvals-security#common-sandbox-and-approval-combinations`
