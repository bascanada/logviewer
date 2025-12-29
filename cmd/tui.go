// SPDX-License-Identifier: GPL-3.0-only
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/factory"
	"github.com/bascanada/logviewer/pkg/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:     "tui",
	Aliases: []string{"live", "ui"},
	Short:   "Launch interactive TUI for log viewing",
	Long: `Launch an interactive Terminal User Interface for browsing and filtering logs.

The TUI provides:
  - Tab-based navigation between multiple contexts
  - Real-time log streaming
  - Vim-style navigation (j/k, gg, G)
  - Full-text search with /
  - Detailed JSON field inspection in sidebar

Examples:
  # Launch TUI with current context
  logviewer tui

  # Launch TUI with specific context(s)
  logviewer tui -i prod-logs
  logviewer tui -i prod-logs -i staging-logs

  # Launch TUI with filters
  logviewer tui -i prod-logs -f level=ERROR --last 1h

  # Launch TUI with query
  logviewer tui -i prod-logs -q "level=ERROR AND service=api"`,
	PreRun: onCommandStart,
	Run:    runTUI,
}

func init() {
	// TUI uses the same flags as query
	// These are already defined on queryCommand, so we just need to add this as a subcommand
}

func runTUI(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, _, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Tip: Run 'logviewer configure' to set up a configuration.")
		os.Exit(1)
	}

	// Create factories
	clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client factory: %v\n", err)
		os.Exit(1)
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating search factory: %v\n", err)
		os.Exit(1)
	}

	// Build search request from flags
	searchRequest := buildSearchRequest()

	// Get runtime variables
	runtimeVars := parseRuntimeVars()

	// Resolve context IDs
	resolvedContextIds := resolveContextIdsFromConfig(cfg)
	if len(resolvedContextIds) == 0 {
		// If no context specified, try to show available contexts
		if len(cfg.Contexts) > 0 {
			fmt.Fprintln(os.Stderr, "No context specified. Available contexts:")
			for id := range cfg.Contexts {
				fmt.Fprintf(os.Stderr, "  - %s\n", id)
			}
			fmt.Fprintln(os.Stderr, "\nUse: logviewer tui -i <context-id>")
			fmt.Fprintln(os.Stderr, "Or set a default: logviewer context use <context-id>")
		} else {
			fmt.Fprintln(os.Stderr, "No contexts defined in configuration.")
			fmt.Fprintln(os.Stderr, "Run 'logviewer configure' to set up contexts.")
		}
		os.Exit(1)
	}

	// Create TUI model
	model := tui.New(cfg, clientFactory, searchFactory)
	model.RuntimeVars = runtimeVars
	model.InitialContexts = resolvedContextIds
	model.InitialSearch = copySearchRequest(&searchRequest)

	// Create the bubbletea program
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Run the TUI
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Check if there was an error
	if m, ok := finalModel.(tui.Model); ok {
		for _, tab := range m.Tabs {
			if tab.Error != nil {
				fmt.Fprintf(os.Stderr, "Tab '%s' had error: %v\n", tab.Name, tab.Error)
			}
		}
	}
}

// copySearchRequest creates a deep copy of a LogSearch
func copySearchRequest(src *client.LogSearch) *client.LogSearch {
	dst := &client.LogSearch{
		Fields:          make(map[string]string),
		FieldsCondition: make(map[string]string),
		Options:         make(map[string]interface{}),
		Follow:          src.Follow,
	}

	// Copy simple fields
	dst.Size = src.Size
	dst.PageToken = src.PageToken
	dst.Range = src.Range
	dst.Refresh = src.Refresh
	dst.FieldExtraction = src.FieldExtraction
	dst.PrinterOptions = src.PrinterOptions
	dst.NativeQuery = src.NativeQuery

	// Copy filter if present
	if src.Filter != nil {
		filterCopy := *src.Filter
		dst.Filter = &filterCopy
	}

	// Copy maps
	for k, v := range src.Fields {
		dst.Fields[k] = v
	}
	for k, v := range src.FieldsCondition {
		dst.FieldsCondition[k] = v
	}
	for k, v := range src.Options {
		dst.Options[k] = v
	}

	// Copy variables if present
	if src.Variables != nil {
		dst.Variables = make(map[string]client.VariableDefinition)
		for k, v := range src.Variables {
			dst.Variables[k] = v
		}
	}

	return dst
}

// addTUIFlags adds TUI-specific flags
func addTUIFlags(cmd *cobra.Command) {
	// TUI currently shares all flags with query command
	// Future: add TUI-specific flags like --split-ratio, --theme, etc.
}

// formatContextList formats a list of contexts for display
func formatContextList(contexts map[string]interface{}) string {
	if len(contexts) == 0 {
		return "  (none)"
	}
	var lines []string
	for id := range contexts {
		lines = append(lines, fmt.Sprintf("  - %s", id))
	}
	return strings.Join(lines, "\n")
}
