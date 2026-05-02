# Go 1.26.2 And Safe String Command Config

## Summary

- Upgrade the project toolchain target from `go 1.24` to `go 1.26.2`, including CI/release workflow defaults and local setup action defaults.
- Replace config-facing command arrays with shell-like command strings for `[goal].confirm_command` and `[pre_loop_continue].command`.
- Keep runtime execution via `exec.CommandContext(command, args...)`; do not introduce implicit `/bin/sh -c` execution.
- Add `mvdan.cc/sh/v3` as the command parsing/quoting dependency.

## Key Changes

- Change `go.mod` to `go 1.26.2` and add `mvdan.cc/sh/v3` using `go get`, then run `go mod tidy`.
- Update GitHub workflow/tooling Go versions from `1.24.x` to `1.26.2`.
- Change `GoalConfig.ConfirmCommand` and `PreLoopContinueConfig.Command` from `[]string` to `string`.
- Change default config and docs from arrays to strings.
- Preserve direct argv execution and require explicit `bash -lc` or similar when users need shell features.

## Command Parsing And Escaping

- Replace argv-array expansion with a string-based parser that returns `(command string, args []string, env []string, err error)`.
- Quote known scalar placeholders with `syntax.Quote(value, syntax.LangBash)` before tokenization so injected values remain literal argv items, including multiline prompts that POSIX quoting cannot represent.
- Expand known multi-arg placeholders to a sequence of quoted argv literals before tokenization, preserving `$MODEL_ARGV` and `$REASONING_ARGV`.
- Tokenize the expanded command line with `mvdan.cc/sh/v3/shell.Fields`.
- Continue exporting the same values as environment variables with `CODEX_LOOP_CONFIRM_` and `CODEX_LOOP_PRE_LOOP_` prefixes.
- Preserve relative executable resolution against the selected cwd/workspace root.

## Test Plan

- Add parser unit tests for quoted strings, scalar placeholder escaping, multi placeholder expansion, unmatched quotes, empty commands, and relative executable resolution.
- Update config tests for string command parsing and default command content.
- Update Stop handler tests so goal confirmation and pre-loop commands use string config while preserving current behavior.
- Run `make deps`, `make fmt`, `go vet ./...`, `make lint`, `go test -race ./...`, `make verify`, and release checks if tooling is affected.

## Assumptions

- The accepted toolchain target is exactly Go `1.26.2`.
- Config command strings improve DX, while runtime must still execute argv directly.
- Shell metacharacters in placeholder values must never gain control-flow semantics.
- `$MODEL_ARGV` and `$REASONING_ARGV` intentionally remain multi-argument placeholders; all other placeholders are scalar.
