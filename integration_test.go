// Integration tests for GitHub Project Import Extension
// These tests simulate real-world scenarios without making actual API calls
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestEndToEndImportWorkflow(t *testing.T) {
	// Create temporary files
	tmpDir := t.TempDir()

	// Create test JSON file
	jsonFile := filepath.Join(tmpDir, "test_import.json")
	jsonContent := `[
		{
			"title": "Draft Issue Example",
			"notes": "This should become a draft issue",
			"Status": "Todo",
			"Estimate": 3,
      "Theme": "PTO"
		},
		{
			"title": "Existing Issue Example",
			"url": "https://github.com/cli/cli/issues/2",
			"Status": "In Progress",
			"Priority": "High",
			"Due Date": "2024-12-31"
		},
		{
			"title": "Pull Request Example", 
			"url": "https://github.com/cli/cli/pull/1",
			"Status": "Done",
			"Notes": "Ready for review",
      "M": "M123"
		}
	]`

	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	// Create test CSV file
	csvFile := filepath.Join(tmpDir, "test_import.csv")
	csvContent := `Title,URL,Status,Priority,Estimate,Notes,M,Theme
Simple Draft Task,,Todo,Low,1,Basic task without URL,M123,Ops
Bug Fix,https://github.com/cli/cli/issues/4,In Progress,High,2,Critical bug,,
Feature Request,https://github.com/cli/cli/pull/3,Done,Medium,5,New feature implementation,,`

	err = os.WriteFile(csvFile, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV file: %v", err)
	}

	tests := []struct {
		name        string
		sourceFile  string
		expectItems int
		expectError bool
	}{
		{
			name:        "JSON import workflow",
			sourceFile:  jsonFile,
			expectItems: 3,
			expectError: false,
		},
		{
			name:        "CSV import workflow",
			sourceFile:  csvFile,
			expectItems: 3,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the file
			var items []ImportItem
			var err error

			if filepath.Ext(tt.sourceFile) == ".json" {
				items, err = ParseJSONFile(tt.sourceFile)
			} else {
				items, err = ParseCSVFile(tt.sourceFile)
			}

			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			if len(items) != tt.expectItems {
				t.Errorf("Expected %d items, got %d", tt.expectItems, len(items))
			}

			// Validate items
			err = ValidateImportItems(items)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}

			// Test with snapshot client
			client, err := NewSnapshotGitHubClient("EndToEndWorkflow")
			if err != nil {
				t.Fatalf("Failed to create snapshot client: %v", err)
			}
			defer client.Close()

			// 1. Find project by name
			project, err := client.FindProject(fmt.Sprintf("%s/%s", testUsername, testProjectTitle))
			if err != nil {
				t.Fatalf("Failed to find project: %v", err)
			}
			t.Logf("Found project by name: %s", project.Title)

			fields, err := client.GetProjectFields(project.ID)
			if err != nil {
				t.Fatalf("Failed to get project fields: %v", err)
			}

			fieldMap := make(map[string]ProjectField)
			for _, field := range fields {
				fieldMap[field.Name] = field
			}

			// Test field validation
			warnings := validateItemFields(items, fieldMap, Config{Verbose: true})
			if len(warnings) > 0 {
				t.Logf("Field validation warnings: %v", warnings)
			}

			// Simulate the import process (without actually calling GitHub API)
			for i, item := range items {
				itemType := GetItemType(item)
				t.Logf("Item %d: %s (%s)", i+1, item.Title, itemType)

				// Test field conversion
				for fieldName, fieldValue := range item.Fields {
					if field, exists := fieldMap[fieldName]; exists {
						convertedValue, err := convertFieldValue(fieldValue, field)
						if err != nil {
							t.Logf("Field conversion warning for %s: %v", fieldName, err)
						} else {
							t.Logf("Field %s converted successfully: %v", fieldName, convertedValue)
						}
					}
				}
			}

			err = importItems(client, project, items, fieldMap, Config{})
			if err != nil {
				t.Fatalf("Failed integrated importItems: %v", err)
			}
		})
	}
}
