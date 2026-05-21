package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	commandspkg "github.com/channyeintun/nami/internal/commands"
	configpkg "github.com/channyeintun/nami/internal/config"
)

func newMCPCommand() *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server configuration",
	}

	mcpCmd.AddCommand(newMCPAddCommand())
	mcpCmd.AddCommand(newMCPAddJSONCommand())
	mcpCmd.AddCommand(newMCPListCommand())
	mcpCmd.AddCommand(newMCPGetCommand())
	mcpCmd.AddCommand(newMCPRemoveCommand())

	return mcpCmd
}

func newMCPAddCommand() *cobra.Command {
	options := commandspkg.MCPAddOptions{}
	cmd := &cobra.Command{
		Use:   "add <name> <command-or-url> [args...]",
		Short: "Add an MCP server",
		Long: "Add an MCP server to Nami's MCP configuration.\n\n" +
			"Examples:\n" +
			"  nami mcp add my-server -- npx my-mcp-server\n" +
			"  nami mcp add --transport http sentry https://mcp.sentry.dev/mcp\n" +
			"  nami mcp add --scope user --transport sse relay https://example.com/sse --header 'Authorization: Bearer token'",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := commandspkg.RunMCPAdd(cwd, args, options)
			if err != nil {
				return err
			}
			return renderMCPResult(cmd, result)
		},
	}
	cmd.Flags().StringVarP(&options.Scope, "scope", "s", string(configpkg.MCPScopeProject), "Configuration scope (project or user)")
	cmd.Flags().StringVarP(&options.Transport, "transport", "t", "", "Transport type (stdio, http, sse, or ws). Defaults to stdio")
	cmd.Flags().StringArrayVarP(&options.Env, "env", "e", nil, "Set an environment variable for stdio transport (KEY=value)")
	cmd.Flags().StringArrayVarP(&options.Headers, "header", "H", nil, "Set a request header for http/sse/ws transports (Key: Value)")
	cmd.Flags().BoolVar(&options.Trust, "trust", false, "Mark the server as trusted so configured tool permissions apply")
	cmd.Flags().BoolVar(&options.Disabled, "disabled", false, "Add the server in a disabled state")
	cmd.Flags().IntVar(&options.StartupMS, "startup-timeout-ms", 0, "Override stdio startup timeout in milliseconds")
	return cmd
}

func newMCPAddJSONCommand() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "add-json <name> <json>",
		Short: "Add an MCP server from raw JSON",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := commandspkg.RunMCPAddJSON(cwd, scope, args[0], args[1])
			if err != nil {
				return err
			}
			return renderMCPResult(cmd, result)
		},
	}
	cmd.Flags().StringVarP(&scope, "scope", "s", string(configpkg.MCPScopeProject), "Configuration scope (project or user)")
	return cmd
}

func newMCPListCommand() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured MCP servers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := commandspkg.RunMCPList(cwd, scope)
			if err != nil {
				return err
			}
			return renderMCPResult(cmd, result)
		},
	}
	cmd.Flags().StringVarP(&scope, "scope", "s", "", "Optional scope filter (project or user)")
	return cmd
}

func newMCPGetCommand() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Show MCP server details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := commandspkg.RunMCPGet(cwd, args[0], scope)
			if err != nil {
				if renderErr := renderMCPResult(cmd, result); renderErr != nil {
					return renderErr
				}
				return err
			}
			return renderMCPResult(cmd, result)
		},
	}
	cmd.Flags().StringVarP(&scope, "scope", "s", "", "Optional scope filter (project or user)")
	return cmd
}

func newMCPRemoveCommand() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := commandspkg.RunMCPRemove(cwd, args[0], scope)
			if err != nil {
				return err
			}
			return renderMCPResult(cmd, result)
		},
	}
	cmd.Flags().StringVarP(&scope, "scope", "s", "", "Optional scope filter (project or user)")
	return cmd
}

func renderMCPResult(cmd *cobra.Command, result commandspkg.MCPCommandResult) error {
	for _, warning := range result.WarningLines {
		fmt.Fprintln(cmd.ErrOrStderr(), warning)
	}
	for _, line := range result.OutputLines {
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}
	return nil
}
