# Support Project-local `codex-loop.toml` for `pre_loop_continue`

## Summary

Add project-local configuration support through a `codex-loop.toml` file discovered from the hook CWD up to the resolved workspace root. The effective runtime config precedence is:

`defaults` -> `~/.codex/codex-loop/config.toml` -> nearest project `codex-loop.toml`

The local file overlays by field: values defined in the project override global values, and absent values inherit from global/defaults. This primarily enables per-project `[pre_loop_continue]` behavior without forcing a single global command for every repository.

## Key Changes

- Keep `LoadRuntimeConfig(paths)` as the global-only loader for install/uninstall and hook installation behavior.
- Add an effective runtime config loader for active runtime decisions, taking `paths`, CWD, and workspace root.
- Discover `codex-loop.toml` from CWD upward to workspace root inclusive, never above the workspace; nearest file wins.
- Apply the project file as a per-field overlay over global config/defaults.
- Preserve explicit local `command = ""` as a real override that disables a global `pre_loop_continue.command`.
- Use effective config in the `Stop` path for goal confirmation settings, `pre_loop_continue`, and `pre_loop_continue_active` goal logs.
- Document that `[hooks].stop_timeout_seconds` remains a global install-time setting; local project config affects runtime decisions, not installed hook registrations.

## Test Plan

- Config tests:
  - no local config preserves current global-only behavior;
  - root-level local config overrides global fields;
  - partial local config inherits global fields;
  - local `command = ""` disables global pre-loop command;
  - nearest local file wins when nested configs exist;
  - discovery does not read above workspace root.
- Stop/hook tests:
  - local `pre_loop_continue` output wins over global when both exist;
  - `cwd = "workspace_root"` still resolves to the loop workspace root;
  - goal continuation logs `pre_loop_continue_active: true` when command comes from project config;
  - CLI `hook stop` respects local config using payload CWD.
- Verification:
  - iterate with `go test ./internal/loop ./internal/cli`;
  - final gate: `make fmt`, `go vet ./...`, `make verify`.

## Assumptions

- The project-local filename is exactly `codex-loop.toml`.
- Discovery uses CWD -> ancestors -> workspace root and does not search above workspace root.
- Overlay is per field.
- Global config compatibility is preserved.
- Local config affects runtime behavior only; it does not rewrite installed hooks.
