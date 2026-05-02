package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/compozy/codex-loop/internal/installer"
	"github.com/compozy/codex-loop/internal/loop"
	"github.com/compozy/codex-loop/internal/version"
)

type rootOptions struct {
	codexHome string
}

func Execute(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := NewRootCommand(ctx)
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	return cmd.Execute()
}

func NewRootCommand(ctx context.Context) *cobra.Command {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:           "codex-loop",
		Short:         "Codex lifecycle loop hooks for long-running agent work",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetContext(ctx)
	cmd.PersistentFlags().StringVar(&opts.codexHome, "codex-home", "", "override Codex home directory (defaults to CODEX_HOME or ~/.codex)")

	cmd.AddCommand(newInstallCommand(opts))
	cmd.AddCommand(newUninstallCommand(opts))
	cmd.AddCommand(newStatusCommand(opts))
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newHookCommand(opts))
	return cmd
}

func newInstallCommand(opts *rootOptions) *cobra.Command {
	var sourceBinary string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install or refresh the local codex-loop runtime",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			messages, err := installer.Install(paths, installer.Options{SourceBinary: sourceBinary})
			if err != nil {
				return err
			}
			for _, message := range messages {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), message); err != nil {
					return fmt.Errorf("write install output: %w", err)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sourceBinary, "source-binary", "", "source binary to install into the Codex runtime")
	_ = cmd.Flags().MarkHidden("source-binary")
	return cmd
}

func newUninstallCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the managed codex-loop runtime",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			messages, err := installer.Uninstall(paths)
			if err != nil {
				return err
			}
			for _, message := range messages {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), message); err != nil {
					return fmt.Errorf("write uninstall output: %w", err)
				}
			}
			return nil
		},
	}
}

func newStatusCommand(opts *rootOptions) *cobra.Command {
	var filter loop.StatusFilter
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Print codex-loop state as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			records, err := loop.ListStatusRecords(paths, filter)
			if err != nil {
				return err
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(records); err != nil {
				return fmt.Errorf("encode status JSON: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&filter.All, "all", false, "include non-active loop records")
	cmd.Flags().StringVar(&filter.SessionID, "session-id", "", "only print loop records for one session")
	cmd.Flags().StringVar(&filter.WorkspaceRoot, "workspace-root", "", "only print loop records for one workspace root")
	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print codex-loop version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version.String())
			return err
		},
	}
}

func newHookCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook",
		Short:  "Run Codex lifecycle hook handlers",
		Hidden: true,
	}
	cmd.AddCommand(newUserPromptSubmitHookCommand(opts))
	cmd.AddCommand(newStopHookCommand(opts))
	return cmd
}

func newUserPromptSubmitHookCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:    "user-prompt-submit",
		Short:  "Handle Codex UserPromptSubmit events",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var payload loop.UserPromptPayload
			if err := json.NewDecoder(cmd.InOrStdin()).Decode(&payload); err != nil {
				return fmt.Errorf("invalid JSON input: %w", err)
			}
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			result, err := loop.HandleUserPromptSubmit(paths, payload, zeroTime())
			if err != nil {
				return err
			}
			return writeHookResult(cmd.OutOrStdout(), result)
		},
	}
}

func newStopHookCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:    "stop",
		Short:  "Handle Codex Stop events",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var payload loop.StopPayload
			if err := json.NewDecoder(cmd.InOrStdin()).Decode(&payload); err != nil {
				return writeHookResult(cmd.OutOrStdout(), loop.StopWarning(fmt.Sprintf("Codex loop stop hook received invalid JSON: %v", err)))
			}
			paths, err := pathsFromOptions(opts)
			if err != nil {
				return err
			}
			result, err := loop.HandleStop(paths, payload, zeroTime())
			if err != nil {
				return err
			}
			return writeHookResult(cmd.OutOrStdout(), result)
		},
	}
}

func writeHookResult(out io.Writer, result loop.HookResult) error {
	if result == nil {
		return nil
	}
	encoder := json.NewEncoder(out)
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode hook result: %w", err)
	}
	return nil
}

func pathsFromOptions(opts *rootOptions) (loop.Paths, error) {
	if opts.codexHome != "" {
		return loop.NewPaths(opts.codexHome)
	}
	return loop.DefaultPaths()
}
