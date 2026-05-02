# Unified Argv Commands for Goal Confirmation and Pre Loop Continue

## Summary

Replace ambiguous command configuration with argv arrays. Goal confirmation uses a configurable `confirm_command`; `pre_loop_continue` uses the same `command = []` shape. The default goal confirmation command uses `codex exec --yolo`, with model and reasoning still configured separately.

## Key Changes

- Remove `[goal].codex_command`.
- Add `[goal].confirm_command` as `[]string`.
- Change `[pre_loop_continue]` to `command = []` and remove `args = []`.
- Keep `confirm_model = "gpt-5.5"` and `confirm_reasoning_effort = "high"` defaults.
- Do not keep compatibility aliases for old keys.

Default goal command:

```toml
[goal]
confirm_model = "gpt-5.5"
confirm_reasoning_effort = "high"
confirm_command = [
  "codex", "exec",
  "--cd", "$WORKSPACE_ROOT",
  "--ephemeral",
  "--yolo",
  "--output-schema", "$SCHEMA_PATH",
  "--output-last-message", "$OUTPUT_PATH",
  "$MODEL_ARGV",
  "$REASONING_ARGV",
  "--skip-git-repo-check",
  "-"
]
timeout_seconds = 2400
max_output_bytes = 12000
```

## Command Contract

- Commands run via `exec.CommandContext`, without `/bin/sh -c`.
- Each token expands `$VAR` and `${VAR}` placeholders.
- `$MODEL_ARGV` expands to `["--model", "$MODEL"]` when model is non-empty, otherwise no argv.
- `$REASONING_ARGV` expands to `["--config", "model_reasoning_effort=\"<effort>\""]` when reasoning is non-empty, otherwise no argv.
- Goal confirmation receives the prompt on stdin, plus `$PROMPT` and `$PROMPT_FILE`.
- `pre_loop_continue` receives JSON on stdin, plus `$INPUT_JSON` and `$INPUT_FILE`.
- Goal verdict is read from `$OUTPUT_PATH` first and stdout second.

## Tests

- Config defaults and custom argv parsing.
- Goal default command includes `--yolo` and expands model/reasoning/schema/output variables.
- Custom goal command works with `$PROMPT`, `$PROMPT_FILE`, and stdout verdicts.
- `pre_loop_continue` command arrays work with stdin and `$INPUT_FILE`.
- Timeout, command failure, and invalid JSON continue with warnings.
- Final verification: `go test ./...`, `go vet ./...`, `make lint`, `make verify`, `git diff --check`.

## Assumptions

- `--yolo` is intentional for the default Codex confirmation command despite its documented elevated risk.
- Arrays are the only supported command shape for this v1 feature.
- Custom goal runners must return JSON matching the existing goal verdict schema.
