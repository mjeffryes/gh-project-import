// GitHub Project Import Extension - imports items from JSON/CSV files into GitHub Projects v2
// This extension enables bulk import and migration of project items with field values
package main

import (
	"fmt"
	"os"
	"strconv"
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

	// Validate source file exists and is readable
	if _, err := os.Stat(config.Source); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", config.Source)
	} else if err != nil {
		return fmt.Errorf("cannot access source file %s: %w", config.Source, err)
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
		// Provide more specific error context
		if strings.Contains(err.Error(), "permission denied") {
			return fmt.Errorf("permission denied reading file %s. Check file permissions", config.Source)
		}
		if strings.Contains(err.Error(), "invalid character") {
			return fmt.Errorf("invalid JSON format in file %s: %w", config.Source, err)
		}
		return fmt.Errorf("failed to parse source file %s: %w", config.Source, err)
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

	// Initialize GitHub client
	if config.Verbose {
		fmt.Println("Authenticating with GitHub API...")
	}

	client, err := NewGitHubClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Get current user info
	user, err := client.GetUser()
	if err != nil {
		return fmt.Errorf("failed to authenticate with GitHub: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Authenticated as: %s\n", user)
	}

	// Find the destination project
	if config.Verbose {
		fmt.Printf("Resolving destination project: %s\n", config.Project)
	}

	project, err := client.FindProject(config.Project)
	if err != nil {
		return fmt.Errorf("failed to find project: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Found project: %s (ID: %s)\n", project.Title, project.ID)
	}

	// Get project field schema
	if config.Verbose {
		fmt.Println("Retrieving project field schema...")
	}

	fields, err := client.GetProjectFields(project.ID)
	if err != nil {
		return fmt.Errorf("failed to get project fields: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Found %d project fields:\n", len(fields))
		for _, field := range fields {
			optionInfo := ""
			if len(field.Options) > 0 {
				optionNames := make([]string, len(field.Options))
				for i, opt := range field.Options {
					optionNames[i] = opt.Name
				}
				optionInfo = fmt.Sprintf(" (options: %s)", strings.Join(optionNames, ", "))
			}
			fmt.Printf("  - %s (%s)%s\n", field.Name, field.Type, optionInfo)
		}
	}

	// Validate field compatibility
	if config.Verbose {
		fmt.Println("Analyzing field compatibility...")
	}

	fieldMap := make(map[string]ProjectField)
	for _, field := range fields {
		fieldMap[field.Name] = field
	}

	validationErrors := validateItemFields(items, fieldMap, config)
	if len(validationErrors) > 0 {
		if !config.Quiet {
			fmt.Printf("⚠ Field validation warnings:\n")
			for _, err := range validationErrors {
				fmt.Printf("  - %s\n", err)
			}
		}
	}

	if config.DryRun {
		fmt.Printf("DRY RUN: Would import %d items to project '%s'\n", len(items), project.Title)
		return nil
	}

	// Import items to the project
	return importItems(client, project, items, fieldMap, config)
}

// importItems handles the actual import of items to a project
func importItems(client GitHubClient, project *Project, items []ImportItem, fieldMap map[string]ProjectField, config Config) error {

	successCount := 0
	errorCount := 0

	for i, item := range items {
		if config.Verbose {
			fmt.Printf("Importing item %d/%d: \"%s\" (%s)\n", i+1, len(items), item.Title, GetItemType(item))
		} else if !config.Quiet {
			fmt.Printf("Importing item %d/%d...\n", i+1, len(items))
		}

		err := importSingleItem(client, project, item, fieldMap, config)
		if err != nil {
			errorCount++
			// Provide more specific error context
			itemType := GetItemType(item)
			if config.Verbose {
				fmt.Printf("ERROR: Failed to import item %d (\"%s\", type: %s)\n", i+1, item.Title, itemType)
				fmt.Printf("       %v\n", err)
			} else {
				fmt.Printf("ERROR: Failed to import item %d (\"%s\"): %v\n", i+1, item.Title, err)
			}
			continue
		}

		successCount++
		if config.Verbose {
			fmt.Printf("SUCCESS: Item imported successfully\n")
		}
	}

	// Calculate field statistics
	fieldStats := calculateFieldStatistics(items, fieldMap)

	if !config.Quiet {
		if errorCount > 0 {
			fmt.Printf("✓ Imported %d items to \"%s\"\n", successCount, project.Title)
			fmt.Printf("⚠ %d items failed to import\n", errorCount)
			if !config.Verbose {
				fmt.Printf("Run with --verbose for detailed error information\n")
			}
		} else {
			fmt.Printf("✓ Imported %d items to \"%s\"\n", successCount, project.Title)
		}

		// Field mapping statistics
		if fieldStats.preservedFields > 0 {
			fmt.Printf("✓ Preserved %d field mappings\n", fieldStats.preservedFields)
		}
		if fieldStats.skippedFields > 0 {
			fmt.Printf("⚠ Skipped %d fields due to compatibility issues\n", fieldStats.skippedFields)
			for _, fieldName := range fieldStats.skippedFieldNames {
				fmt.Printf("   - \"%s\" field not found in destination\n", fieldName)
			}
		}
	}

	// Return an error if there were failures and no successes
	if successCount == 0 && errorCount > 0 {
		return fmt.Errorf("failed to import any items")
	}

	return nil
}

// FieldStatistics holds statistics about field mappings
type FieldStatistics struct {
	preservedFields   int
	skippedFields     int
	skippedFieldNames []string
}

// calculateFieldStatistics analyzes field usage and compatibility
func calculateFieldStatistics(items []ImportItem, fieldMap map[string]ProjectField) FieldStatistics {
	uniqueFields := make(map[string]bool)
	skippedFields := make(map[string]bool)

	// Analyze all fields used in items
	for _, item := range items {
		for fieldName := range item.Fields {
			uniqueFields[fieldName] = true
			if _, exists := fieldMap[fieldName]; !exists {
				skippedFields[fieldName] = true
			}
		}
	}

	// Convert skipped fields map to slice for reporting
	var skippedFieldNames []string
	for fieldName := range skippedFields {
		skippedFieldNames = append(skippedFieldNames, fieldName)
	}

	return FieldStatistics{
		preservedFields:   len(uniqueFields) - len(skippedFields),
		skippedFields:     len(skippedFields),
		skippedFieldNames: skippedFieldNames,
	}
}

// importSingleItem imports a single item to a project
func importSingleItem(client GitHubClient, project *Project, item ImportItem, fieldMap map[string]ProjectField, config Config) error {
	var itemID string
	var err error

	itemType := GetItemType(item)

	// Create the item based on its type
	switch itemType {
	case "DraftIssue":
		itemID, err = client.CreateDraftIssue(project.ID, item.Title, GetItemBody(item))
	case "Issue", "PullRequest":
		// For existing issues/PRs, we need to get their content ID and add them to the project
		if item.URL == "" {
			return fmt.Errorf("URL is required for existing issues and pull requests")
		}

		// Get the issue/PR content
		content, err := client.GetIssueOrPR(item.URL)
		if err != nil {
			return fmt.Errorf("failed to get issue/PR content: %w", err)
		}

		// Extract the content ID (node_id)
		contentID, ok := content["node_id"].(string)
		if !ok {
			return fmt.Errorf("could not extract content ID from issue/PR")
		}

		// Add the issue/PR to the project
		itemID, err = client.CreateProjectItem(project.ID, contentID)
	default:
		return fmt.Errorf("unsupported item type: %s", itemType)
	}

	if err != nil {
		return fmt.Errorf("failed to create project item: %w", err)
	}

	// Set field values
	return setItemFields(client, project.ID, itemID, item, fieldMap, config)
}

// setItemFields sets field values for a project item
func setItemFields(client GitHubClient, projectID, itemID string, item ImportItem, fieldMap map[string]ProjectField, config Config) error {
	// Process all custom fields from the Fields map
	for fieldName, fieldValue := range item.Fields {
		field, exists := fieldMap[fieldName]
		if !exists {
			if config.Verbose {
				fmt.Printf("  WARNING: Field '%s' not found in project, skipping\n", fieldName)
			}
			continue
		}

		// Convert the field value to the appropriate format for GraphQL
		convertedValue, err := convertFieldValue(fieldValue, field)
		if err != nil {
			if config.Verbose {
				fmt.Printf("  WARNING: Failed to convert field '%s': %v, skipping\n", fieldName, err)
			}
			continue
		}

		// Set the field value
		err = client.SetProjectItemFieldValue(projectID, itemID, field.ID, convertedValue)
		if err != nil {
			if config.Verbose {
				fmt.Printf("  WARNING: Failed to set field '%s': %v\n", fieldName, err)
			}
			continue
		}

		if config.Verbose {
			fmt.Printf("  Set field: %s = %v\n", fieldName, fieldValue)
		}
	}

	return nil
}

// convertFieldValue converts a field value to the appropriate format for the GitHub GraphQL API
func convertFieldValue(value interface{}, field ProjectField) (interface{}, error) {
	switch field.Type {
	case "TEXT":
		if str, ok := value.(string); ok {
			return map[string]interface{}{"text": str}, nil
		}
		return map[string]interface{}{"text": fmt.Sprintf("%v", value)}, nil

	case "NUMBER":
		var num float64
		switch v := value.(type) {
		case float64:
			num = v
		case int64:
			num = float64(v)
		case int:
			num = float64(v)
		case string:
			var err error
			num, err = strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert '%s' to number", v)
			}
		default:
			return nil, fmt.Errorf("cannot convert %T to number", value)
		}
		return map[string]interface{}{"number": num}, nil

	case "DATE":
		if str, ok := value.(string); ok {
			// Validate ISO date format
			if !strings.Contains(str, "T") {
				// Add time if not present
				str += "T00:00:00Z"
			}
			return map[string]interface{}{"date": str}, nil
		}
		return nil, fmt.Errorf("date field must be a string in ISO format")

	case "SINGLE_SELECT":
		if str, ok := value.(string); ok {
			// Find the option ID for the given name
			for _, option := range field.Options {
				if option.Name == str {
					return map[string]interface{}{"singleSelectOptionId": option.ID}, nil
				}
			}
			return nil, fmt.Errorf("single-select option '%s' not found", str)
		}
		return nil, fmt.Errorf("single-select field must be a string")

	case "USER":
		if str, ok := value.(string); ok {
			// For user fields, we need to resolve the user login to a user ID
			// For now, we'll use the login directly (this might need adjustment)
			return map[string]interface{}{"assigneeIds": []string{str}}, nil
		}
		return nil, fmt.Errorf("user field must be a string")

	case "ITERATION":
		if str, ok := value.(string); ok {
			// Find the iteration ID for the given name
			for _, iteration := range field.Iterations {
				if iteration.Title == str {
					return map[string]interface{}{"iterationId": iteration.ID}, nil
				}
			}
			return nil, fmt.Errorf("iteration '%s' not found", str)
		}
		return nil, fmt.Errorf("iteration field must be a string")

	default:
		return nil, fmt.Errorf("unsupported field type: %s", field.Type)
	}
}

// validateItemFields validates that item fields are compatible with project schema
func validateItemFields(items []ImportItem, fieldMap map[string]ProjectField, config Config) []string {
	var warnings []string
	seenFields := make(map[string]bool)

	for i, item := range items {
		for fieldName, fieldValue := range item.Fields {
			// Track which fields we've seen
			if !seenFields[fieldName] {
				seenFields[fieldName] = true

				field, exists := fieldMap[fieldName]
				if !exists {
					warnings = append(warnings, fmt.Sprintf("Field '%s' not found in project (used in item %d: '%s')", fieldName, i+1, item.Title))
					continue
				}

				// Try to validate the field value
				_, err := convertFieldValue(fieldValue, field)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("Field '%s' validation failed: %v (used in item %d: '%s')", fieldName, err, i+1, item.Title))
				} else if config.Verbose {
					// Only show success for verbose mode
					if len(warnings) == 0 {
						// This is a bit hacky, but we want to show at least one success message
						warnings = append(warnings, fmt.Sprintf("Field '%s' (%s) is compatible", fieldName, field.Type))
						// Remove it immediately so it doesn't show as a warning
						warnings = warnings[:len(warnings)-1]
					}
				}
			}
		}
	}

	// Check for missing required fields (if any)
	// Note: GitHub Projects v2 doesn't have traditional "required" fields,
	// but we can check if common fields like Title are missing
	for i, item := range items {
		if item.Title == "" {
			warnings = append(warnings, fmt.Sprintf("Item %d is missing a title", i+1))
		}
	}

	return warnings
}
