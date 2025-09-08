// GitHub Project Import Extension - imports items from JSON/CSV files into GitHub Projects v2
// This extension enables bulk import and migration of project items with field values
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type Config struct {
	Source  string
	Project string
	DryRun  bool
	Verbose bool
	Quiet   bool
}

func main() {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "project-import",
		Short: "Import items from JSON/CSV files into GitHub Projects v2",
		Long: `Import multiple items to a GitHub Projects v2 board from a JSON or CSV file.
This tool helps automate bulk additions, synchronization, or migration between projects.

Examples:
  gh project-import --source items.json --project "owner/project-name"
  gh project-import --source items.csv --project "123" --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(config)
		},
	}

	rootCmd.Flags().StringVarP(&config.Source, "source", "s", "", "Source file with items to import (required)")
	rootCmd.Flags().StringVarP(&config.Project, "project", "p", "", "Destination project identifier (format: owner/project-name or project-number) (required)")
	rootCmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Preview what would be imported without making changes")
	rootCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().BoolVarP(&config.Quiet, "quiet", "q", false, "Suppress non-error output")

	rootCmd.MarkFlagRequired("source")
	rootCmd.MarkFlagRequired("project")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runImport(config Config) error {
	// Validate flags
	if config.Verbose && config.Quiet {
		return fmt.Errorf("cannot use both --verbose and --quiet flags")
	}

	if !config.Quiet {
		fmt.Printf("Starting import from %s to project %s\n", config.Source, config.Project)
		if config.DryRun {
			fmt.Println("Running in dry-run mode - no changes will be made")
		}
	}

	// Parse the source file
	var items []ImportItem
	var err error

	if strings.HasSuffix(strings.ToLower(config.Source), ".json") {
		items, err = ParseJSONFile(config.Source)
	} else if strings.HasSuffix(strings.ToLower(config.Source), ".csv") {
		items, err = ParseCSVFile(config.Source)
	} else {
		return fmt.Errorf("unsupported file format. Only .json and .csv files are supported")
	}

	if err != nil {
		return fmt.Errorf("failed to parse source file: %w", err)
	}

	// Validate items
	if err := ValidateImportItems(items); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Successfully parsed %d items from %s\n", len(items), config.Source)
		for i, item := range items {
			fmt.Printf("  %d. %s (%s)\n", i+1, item.Title, GetItemType(item))
		}
	} else if !config.Quiet {
		fmt.Printf("Parsed %d items from source file\n", len(items))
	}

	// TODO: Implement GitHub API integration
	fmt.Println("GitHub API integration not yet implemented")
	
	return nil
}
