# Accepted Plan: Text `confirm_command` With `gpt-5.4-mini` Interpreter

Accepted on 2026-05-02 after the user confirmed the researched plan.

## Summary

- Change goal confirmation so `[goal].confirm_command` is a plain-text integration point: custom runners return normal human-readable review text, not codex-loop verdict JSON.
- Add a private interpreter phase whose default model is `gpt-5.4-mini`; it converts the review text into the existing internal `GoalCheckVerdict` JSON.
- Keep the default interpreter implemented through `codex exec --output-schema`, not `openai-go`, because official docs and local CLI behavior show Codex supports structured output while reusing Codex login/configuration.
- Do not add `github.com/openai/openai-go/v3` in this change. It supports OpenAI API Structured Outputs, but does not document reuse of Codex CLI ChatGPT OAuth credentials.

## Key Changes

- Update `[goal]` config:
  - Keep `confirm_model` and `confirm_reasoning_effort` for the text-producing reviewer command.
  - Change default `confirm_command` to text output only, using `codex exec --output-last-message $CONFIRM_OUTPUT_PATH` and no `--output-schema`.
  - Add `interpret_model = "gpt-5.4-mini"`, `interpret_reasoning_effort = "low"`, `interpret_command`, and `interpret_timeout_seconds = 120`.
  - Default `interpret_command` uses `codex exec` with `--output-schema $INTERPRET_SCHEMA_PATH`, `--output-last-message $INTERPRET_OUTPUT_PATH`, `--sandbox read-only`, and `--disable shell_tool`.
- Rework goal confirmation flow:
  - Phase 1 runs `confirm_command`, captures stdout or `$CONFIRM_OUTPUT_PATH`, and treats it as plain text.
  - Phase 2 runs `interpret_command` with a prompt containing the original goal, task prompt, latest assistant message, and captured review text.
  - Decode `GoalCheckVerdict` only from interpreter output; no custom runner JSON shortcut.
  - If confirm fails, is empty, or times out, continue with a warning.
  - If interpretation fails, times out, refuses, or emits invalid JSON, continue with a warning and include the captured review text in the continuation prompt.
- Update variables and docs:
  - Public confirm variables include `$PROMPT`, `$PROMPT_FILE`, `$CONFIRM_OUTPUT_PATH`, `$MODEL_ARGV`, `$REASONING_ARGV`, `$MODEL`, `$REASONING_EFFORT`, `$WORKSPACE_ROOT`, `$CWD`, `$SESSION_ID`, `$LOOP_NAME`, `$LOOP_SLUG`, `$RUNS_LOG_PATH`, and `$CODEX_HOME`.
  - Interpreter-only variables include `$INTERPRET_SCHEMA_PATH`, `$INTERPRET_OUTPUT_PATH`, `$INTERPRET_MODEL_ARGV`, `$INTERPRET_REASONING_ARGV`, `$INTERPRET_MODEL`, and `$INTERPRET_REASONING_EFFORT`.
  - Remove public docs/examples that tell custom `confirm_command` integrations to emit `completed`, `confidence`, `reason`, `missing_work`, or `next_round_guidance` JSON.
  - README and plugin skill should state that Codex `Stop` hook output remains JSON because Codex requires it, but `confirm_command` output is plain text.

## Test Plan

- Unit and integration-style tests:
  - Default confirm command does not include `--output-schema`.
  - Default interpreter model is exactly `gpt-5.4-mini`.
  - Plain-text complete review is interpreted as completed and marks the loop completed.
  - Plain-text incomplete review is interpreted as incomplete and emits a continuation prompt with interpreted missing work plus the review text.
  - Confirm command failure, empty output, timeout, interpreter invalid JSON, interpreter timeout, and interpreter refusal all keep the loop active with warnings.
  - `runs.jsonl` remains compact JSON metadata and does not persist full raw review text.
- Verification commands:
  - `go test ./internal/loop ./internal/cli`
  - `go test -race ./internal/loop ./internal/cli`
  - `make fmt`
  - `go vet ./...`
  - `make verify`
- Manual isolated auth check:
  - With a temp `CODEX_HOME`, verify API-key login still works through `codex exec`.
  - In the real local environment, `codex login status` can confirm ChatGPT OAuth is available without reading credential files.

## Assumptions And References

- `interpret_reasoning_effort = "low"` is the default because the interpreter is a constrained classification/normalization step; users can raise it if evals show better decisions.
- `openai-go` is not the default path for this feature: its README documents Responses API, Structured Outputs, API-key default auth, and workload identity, but not reuse of Codex ChatGPT OAuth.
- `openai-go` remains a possible future optional backend only if the project intentionally adds an API-key or workload-identity interpreter mode.
- Official docs checked:
  - Structured Outputs: https://developers.openai.com/api/docs/guides/structured-outputs
  - Codex CLI `login`: https://developers.openai.com/codex/cli/reference#codex-login
  - Codex config/auth/provider reference: https://developers.openai.com/codex/config-reference#configtoml
  - Codex custom provider auth: https://developers.openai.com/codex/config-advanced#custom-model-providers
  - Codex structured review cookbook: https://developers.openai.com/cookbook/examples/codex/build_code_review_with_codex_sdk
  - `openai-go`: https://github.com/openai/openai-go
- Research also used Exa and Context7:
  - Exa found the Codex structured review cookbook, Codex TypeScript SDK structured output docs, and `openai-go` structured output example.
  - Context7 confirmed `/openai/codex` exposes per-turn `outputSchema` examples and `/openai/openai-go` exposes Structured Outputs examples.
