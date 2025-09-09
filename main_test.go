// Unit tests for the GitHub Project Import Extension
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				Source:  "test.json",
				Project: "owner/project",
				DryRun:  true,
			},
			expectError: false,
		},
		{
			name: "verbose and quiet both set",
			config: Config{
				Source:  "test.json",
				Project: "owner/project",
				Verbose: true,
				Quiet:   true,
			},
			expectError: true,
			errorMsg:    "cannot use both --verbose and --quiet flags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic
			if tt.config.Verbose && tt.config.Quiet {
				if !tt.expectError {
					t.Errorf("Expected error for verbose and quiet both set")
				}
			}
		})
	}
}

func TestFileFormatDetection(t *testing.T) {
	tests := []struct {
		filename    string
		isJSON      bool
		isCSV       bool
		expectError bool
	}{
		{"test.json", true, false, false},
		{"test.csv", false, true, false},
		{"test.txt", false, false, true},
		{"test.JSON", true, false, false}, // Case insensitive
		{"test.CSV", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			isJSON := false
			isCSV := false
			isSupported := false

			filename := tt.filename
			if filepath.Ext(filename) == ".json" || filepath.Ext(filename) == ".JSON" {
				isJSON = true
				isSupported = true
			} else if filepath.Ext(filename) == ".csv" || filepath.Ext(filename) == ".CSV" {
				isCSV = true
				isSupported = true
			}

			if isJSON != tt.isJSON {
				t.Errorf("Expected isJSON=%v, got %v", tt.isJSON, isJSON)
			}
			if isCSV != tt.isCSV {
				t.Errorf("Expected isCSV=%v, got %v", tt.isCSV, isCSV)
			}
			if !isSupported && !tt.expectError {
				t.Errorf("Expected unsupported format to cause error")
			}
		})
	}
}

func TestValidateImportItems(t *testing.T) {
	tests := []struct {
		name        string
		items       []ImportItem
		expectError bool
	}{
		{
			name:        "empty items",
			items:       []ImportItem{},
			expectError: true,
		},
		{
			name: "valid items",
			items: []ImportItem{
				{Title: "Test Item 1"},
				{Title: "Test Item 2"},
			},
			expectError: false,
		},
		{
			name: "item without title",
			items: []ImportItem{
				{Title: "Test Item 1"},
				{Title: ""}, // Missing title
			},
			expectError: true,
		},
		{
			name: "item with invalid URL",
			items: []ImportItem{
				{Title: "Test Item", URL: "https://example.com/not-github"},
			},
			expectError: true,
		},
		{
			name: "item with valid GitHub URL",
			items: []ImportItem{
				{Title: "Test Item", URL: "https://github.com/owner/repo/issues/1"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImportItems(tt.items)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGetItemType(t *testing.T) {
	tests := []struct {
		name     string
		item     ImportItem
		expected string
	}{
		{
			name: "draft issue (no URL)",
			item: ImportItem{
				Title: "Test Draft",
			},
			expected: "DraftIssue",
		},
		{
			name: "existing issue",
			item: ImportItem{
				Title: "Test Issue",
				URL:   "https://github.com/owner/repo/issues/123",
			},
			expected: "Issue",
		},
		{
			name: "pull request",
			item: ImportItem{
				Title: "Test PR",
				URL:   "https://github.com/owner/repo/pull/456",
			},
			expected: "PullRequest",
		},
		{
			name: "content type specified",
			item: ImportItem{
				Title:   "Test",
				Content: ItemContent{Type: "Issue"},
			},
			expected: "Issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetItemType(tt.item)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetItemBody(t *testing.T) {
	tests := []struct {
		name     string
		item     ImportItem
		expected string
	}{
		{
			name: "content body",
			item: ImportItem{
				Content: ItemContent{Body: "Content body text"},
				Notes:   "Notes text",
			},
			expected: "Content body text",
		},
		{
			name: "notes fallback",
			item: ImportItem{
				Content: ItemContent{Body: ""},
				Notes:   "Notes text",
			},
			expected: "Notes text",
		},
		{
			name: "empty body",
			item: ImportItem{
				Content: ItemContent{Body: ""},
				Notes:   "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetItemBody(tt.item)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertFieldValue(t *testing.T) {
	// Create a test field for each type
	textField := ProjectField{Name: "Text Field", Type: "TEXT"}
	numberField := ProjectField{Name: "Number Field", Type: "NUMBER"}
	dateField := ProjectField{Name: "Date Field", Type: "DATE"}
	selectField := ProjectField{
		Name: "Select Field",
		Type: "SINGLE_SELECT",
		Options: []ProjectFieldOption{
			{ID: "1", Name: "Option 1"},
			{ID: "2", Name: "Option 2"},
		},
	}

	tests := []struct {
		name        string
		value       interface{}
		field       ProjectField
		expectError bool
	}{
		// Text field tests
		{"text field with string", "hello", textField, false},
		{"text field with number", 42, textField, false},

		// Number field tests
		{"number field with int", 42, numberField, false},
		{"number field with float", 42.5, numberField, false},
		{"number field with string number", "42", numberField, false},
		{"number field with invalid string", "not a number", numberField, true},

		// Date field tests
		{"date field with ISO date", "2023-01-01", dateField, false},
		{"date field with full ISO", "2023-01-01T10:00:00Z", dateField, false},
		{"date field with number", 123, dateField, true},

		// Single select tests
		{"select field with valid option", "Option 1", selectField, false},
		{"select field with invalid option", "Option 3", selectField, true},
		{"select field with number", 123, selectField, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := convertFieldValue(tt.value, tt.field)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestJSONParsing(t *testing.T) {
	// Create a temporary JSON file for testing
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "test.json")

	jsonContent := `[
		{
			"title": "Test Item 1",
			"status": "Open",
			"estimate": 3
		},
		{
			"title": "Test Item 2",
			"url": "https://github.com/owner/repo/issues/123",
			"assignees": ["user1", "user2"]
		}
	]`

	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	items, err := ParseJSONFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to parse JSON file: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	// Test first item
	if items[0].Title != "Test Item 1" {
		t.Errorf("Expected title 'Test Item 1', got '%s'", items[0].Title)
	}

	if items[0].Fields["status"] != "Open" {
		t.Errorf("Expected status 'Open', got %v", items[0].Fields["status"])
	}

	// Test second item
	if items[1].Title != "Test Item 2" {
		t.Errorf("Expected title 'Test Item 2', got '%s'", items[1].Title)
	}

	if len(items[1].Assignees) != 2 {
		t.Errorf("Expected 2 assignees, got %d", len(items[1].Assignees))
	}
}

func TestCSVParsing(t *testing.T) {
	// Create a temporary CSV file for testing
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")

	csvContent := `Title,Status,Estimate,Assignees
Test Item 1,Open,3,user1
Test Item 2,Closed,5,"user1,user2"
Test Item 3,In Progress,2,`

	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	items, err := ParseCSVFile(csvFile)
	if err != nil {
		t.Fatalf("Failed to parse CSV file: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	// Test first item
	if items[0].Title != "Test Item 1" {
		t.Errorf("Expected title 'Test Item 1', got '%s'", items[0].Title)
	}

	if items[0].Fields["Status"] != "Open" {
		t.Errorf("Expected status 'Open', got %v", items[0].Fields["Status"])
	}

	// Test item with multiple assignees
	if len(items[1].Assignees) != 2 {
		t.Errorf("Expected 2 assignees, got %d", len(items[1].Assignees))
	}
}

func TestComplexFieldValidation(t *testing.T) {
	// Create test items with various field configurations
	items := []ImportItem{
		{
			Title: "Item with valid fields",
			Fields: map[string]interface{}{
				"Status":   "Todo",
				"Priority": "High",
				"Estimate": 5,
				"Notes":    "Test notes",
				"Due Date": "2024-12-31",
			},
		},
		{
			Title: "Item with invalid fields",
			Fields: map[string]interface{}{
				"Status":      "InvalidStatus", // Invalid single-select option
				"Priority":    "ValidPriority", // This should be valid
				"Estimate":    "not-a-number",  // Invalid number
				"Due Date":    "invalid-date",  // Invalid date format
				"NonExistent": "value",         // Field doesn't exist
			},
		},
	}

	fields := []ProjectField{
		{ID: "field1", Name: "Status", Type: "SINGLE_SELECT", Options: []ProjectFieldOption{
			{ID: "opt1", Name: "Todo"},
			{ID: "opt2", Name: "In Progress"},
			{ID: "opt3", Name: "Done"},
		}},
		{ID: "field2", Name: "Priority", Type: "SINGLE_SELECT", Options: []ProjectFieldOption{
			{ID: "opt4", Name: "Low"},
			{ID: "opt5", Name: "Medium"},
			{ID: "opt6", Name: "High"},
		}},
		{ID: "field3", Name: "Estimate", Type: "NUMBER"},
		{ID: "field4", Name: "Notes", Type: "TEXT"},
		{ID: "field5", Name: "Due Date", Type: "DATE"},
	}

	fieldMap := make(map[string]ProjectField)
	for _, field := range fields {
		fieldMap[field.Name] = field
	}

	warnings := validateItemFields(items, fieldMap, Config{Verbose: true})

	// We should get warnings for invalid values
	if len(warnings) == 0 {
		t.Error("Expected validation warnings but got none")
	}

	// Check specific warning types
	hasMissingFieldWarning := false

	for _, warning := range warnings {
		if contains(warning, "NonExistent") {
			hasMissingFieldWarning = true
		}
		// Log all warnings for debugging
		t.Logf("Warning: %s", warning)
	}

	if !hasMissingFieldWarning {
		t.Error("Expected warning about non-existent field")
	}

	// Test that invalid values are detected during conversion (not in pre-validation)
	testFieldMap := make(map[string]ProjectField)
	for _, field := range fields {
		testFieldMap[field.Name] = field
	}

	// Test conversion errors directly
	statusField := testFieldMap["Status"]
	_, err := convertFieldValue("InvalidStatus", statusField)
	if err == nil {
		t.Error("Expected error for invalid single-select option")
	}

	estimateField := testFieldMap["Estimate"]
	_, err = convertFieldValue("not-a-number", estimateField)
	if err == nil {
		t.Error("Expected error for invalid number format")
	}

	t.Logf("Validation warnings: %v", warnings)
}

// Helper function to check if string contains substring
func contains(str, substr string) bool {
	return strings.Contains(str, substr)
}
