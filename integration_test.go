// Integration tests for GitHub Project Import Extension
// These tests simulate real-world scenarios without making actual API calls
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockGitHubClient simulates GitHub API responses for testing
type MockGitHubClient struct {
	shouldError bool
	projects    map[string]*Project
	fields      map[string][]ProjectField
}

func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		projects: make(map[string]*Project),
		fields:   make(map[string][]ProjectField),
	}
}

func (m *MockGitHubClient) GetUser() (string, error) {
	if m.shouldError {
		return "", &mockError{"mock user error"}
	}
	return "test-user", nil
}

func (m *MockGitHubClient) FindProject(identifier string) (*Project, error) {
	if m.shouldError {
		return nil, &mockError{"mock project not found"}
	}
	
	if project, exists := m.projects[identifier]; exists {
		return project, nil
	}
	
	// Create a default project for testing
	project := &Project{
		ID:     "PVT_test123",
		Number: 1,
		Title:  "Test Project",
		URL:    "https://github.com/test/project/projects/1",
	}
	m.projects[identifier] = project
	return project, nil
}

func (m *MockGitHubClient) GetProjectFields(projectID string) ([]ProjectField, error) {
	if m.shouldError {
		return nil, &mockError{"mock fields error"}
	}
	
	if fields, exists := m.fields[projectID]; exists {
		return fields, nil
	}
	
	// Return default fields for testing
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
	m.fields[projectID] = fields
	return fields, nil
}

func (m *MockGitHubClient) CreateDraftIssue(projectID, title, body string) (string, error) {
	if m.shouldError {
		return "", &mockError{"mock create draft issue error"}
	}
	return "ITEM_test123", nil
}

func (m *MockGitHubClient) CreateProjectItem(projectID, contentID string) (string, error) {
	if m.shouldError {
		return "", &mockError{"mock create project item error"}
	}
	return "ITEM_test456", nil
}

func (m *MockGitHubClient) GetIssueOrPR(url string) (map[string]interface{}, error) {
	if m.shouldError {
		return nil, &mockError{"mock get issue/PR error"}
	}
	
	return map[string]interface{}{
		"node_id": "ISSUE_test789",
		"title":   "Test Issue",
		"body":    "Test issue body",
	}, nil
}

func (m *MockGitHubClient) SetProjectItemFieldValue(projectID, itemID, fieldID string, value interface{}) error {
	if m.shouldError {
		return &mockError{"mock set field value error"}
	}
	return nil
}

// mockError implements the error interface for testing
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

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
			"Estimate": 3
		},
		{
			"title": "Existing Issue Example",
			"url": "https://github.com/test/repo/issues/123",
			"Status": "In Progress",
			"Priority": "High",
			"Due Date": "2024-12-31"
		},
		{
			"title": "Pull Request Example", 
			"url": "https://github.com/test/repo/pull/456",
			"Status": "Done",
			"Notes": "Ready for review"
		}
	]`
	
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}
	
	// Create test CSV file
	csvFile := filepath.Join(tmpDir, "test_import.csv")
	csvContent := `Title,URL,Status,Priority,Estimate,Notes
Simple Draft Task,,Todo,Low,1,Basic task without URL
Bug Fix,https://github.com/test/repo/issues/789,In Progress,High,2,Critical bug
Feature Request,https://github.com/test/repo/pull/101,Done,Medium,5,New feature implementation`
	
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
			
			// Test with mock client
			mockClient := NewMockGitHubClient()
			
			project := &Project{
				ID:     "PVT_test123",
				Number: 1,
				Title:  "Test Project",
				URL:    "https://github.com/test/project/projects/1",
			}
			
			fields, err := mockClient.GetProjectFields(project.ID)
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
		})
	}
}

func TestErrorHandlingWorkflows(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockGitHubClient)
		expectError bool
		errorType   string
	}{
		{
			name: "API error handling",
			setupMock: func(m *MockGitHubClient) {
				m.shouldError = true
			},
			expectError: true,
			errorType:   "API",
		},
		{
			name: "successful workflow",
			setupMock: func(m *MockGitHubClient) {
				m.shouldError = false
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockGitHubClient()
			tt.setupMock(mockClient)
			
			// Test user authentication
			user, err := mockClient.GetUser()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if user != "test-user" {
					t.Errorf("Expected user 'test-user', got '%s'", user)
				}
			}
			
			// Test project discovery
			project, err := mockClient.FindProject("test/project")
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if project.Title != "Test Project" {
					t.Errorf("Expected project title 'Test Project', got '%s'", project.Title)
				}
			}
		})
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
				"Status":     "InvalidStatus", // Invalid single-select option
				"Priority":   "ValidPriority", // This should be valid  
				"Estimate":   "not-a-number",  // Invalid number
				"Due Date":   "invalid-date",  // Invalid date format
				"NonExistent": "value",        // Field doesn't exist
			},
		},
	}
	
	mockClient := NewMockGitHubClient()
	fields, _ := mockClient.GetProjectFields("test-project")
	
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