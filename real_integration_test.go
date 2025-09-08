// Real GitHub integration tests for project import extension
// These tests create actual projects on GitHub and perform real operations
//go:build integration
// +build integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRealIntegrationWorkflow tests the complete workflow with real GitHub projects
func TestRealIntegrationWorkflow(t *testing.T) {
	// Skip if not explicitly enabled
	if os.Getenv("REAL_INTEGRATION_TESTS") != "true" {
		t.Skip("Real integration tests not enabled - set REAL_INTEGRATION_TESTS=true to run")
	}
	
	client, err := NewGitHubClient()
	if err != nil {
		t.Fatalf("Failed to create GitHub client: %v", err)
	}
	
	// Get current user to determine project owner
	user, err := client.GetUser()
	if err != nil {
		t.Fatalf("Failed to get authenticated user: %v", err)
	}
	
	t.Logf("Running integration test as user: %s", user)
	
	// Test project names with timestamp to avoid conflicts
	timestamp := time.Now().Format("20060102-150405")
	sourceTitle := "Test Source Project " + timestamp
	destTitle := "Test Dest Project " + timestamp
	
	var sourceProject *Project
	var destProject *Project
	
	// Cleanup function to delete projects when done
	defer func() {
		if sourceProject != nil {
			t.Logf("Cleaning up source project: %s", sourceProject.ID)
			if err := client.DeleteProject(sourceProject.ID); err != nil {
				t.Logf("Warning: failed to delete source project: %v", err)
			}
		}
		if destProject != nil {
			t.Logf("Cleaning up destination project: %s", destProject.ID)
			if err := client.DeleteProject(destProject.ID); err != nil {
				t.Logf("Warning: failed to delete destination project: %v", err)
			}
		}
	}()
	
	// 1. Create source project
	t.Log("Creating source project...")
	sourceProject, err = client.CreateProject("user", user, sourceTitle, "Test source project for integration testing")
	if err != nil {
		t.Fatalf("Failed to create source project: %v", err)
	}
	t.Logf("Created source project: %s (ID: %s)", sourceProject.Title, sourceProject.ID)
	
	// 2. Create destination project
	t.Log("Creating destination project...")
	destProject, err = client.CreateProject("user", user, destTitle, "Test destination project for integration testing")
	if err != nil {
		t.Fatalf("Failed to create destination project: %v", err)
	}
	t.Logf("Created destination project: %s (ID: %s)", destProject.Title, destProject.ID)
	
	// 3. Get project fields
	t.Log("Retrieving project fields...")
	fields, err := client.GetProjectFields(destProject.ID)
	if err != nil {
		t.Fatalf("Failed to get project fields: %v", err)
	}
	t.Logf("Retrieved %d fields from destination project", len(fields))
	
	// 4. Create test data items in source project
	t.Log("Adding test items to source project...")
	
	// Create a draft issue
	draftItemID, err := client.CreateDraftIssue(sourceProject.ID, "Test Draft Item", "This is a test draft issue for integration testing")
	if err != nil {
		t.Fatalf("Failed to create draft issue: %v", err)
	}
	t.Logf("Created draft issue with ID: %s", draftItemID)
	
	// 5. Create test data file for import
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "integration_test.json")
	testContent := `[
		{
			"title": "Integration Test Draft Item",
			"notes": "This is a test draft item created during integration testing"
		}
	]`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test data file: %v", err)
	}
	
	// 6. Parse the test data
	items, err := ParseJSONFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse test data: %v", err)
	}
	
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	
	// 7. Validate import items
	if err := ValidateImportItems(items); err != nil {
		t.Fatalf("Failed to validate import items: %v", err)
	}
	
	// 8. Simulate import process (create items in destination project)
	t.Logf("Importing %d items to destination project...", len(items))
	
	fieldMap := make(map[string]ProjectField)
	for _, field := range fields {
		fieldMap[field.Name] = field
	}
	
	for i, item := range items {
		t.Logf("Processing item %d: %s", i+1, item.Title)
		
		itemType := GetItemType(item)
		var itemID string
		
		if itemType == "DraftIssue" {
			// Create draft issue
			body := GetItemBody(item)
			itemID, err = client.CreateDraftIssue(destProject.ID, item.Title, body)
			if err != nil {
				t.Fatalf("Failed to create draft issue: %v", err)
			}
			t.Logf("Created draft issue with ID: %s", itemID)
		} else if item.URL != "" {
			// Get the content ID from the URL
			content, err := client.GetIssueOrPR(item.URL)
			if err != nil {
				t.Fatalf("Failed to get issue/PR: %v", err)
			}
			
			contentID, ok := content["node_id"].(string)
			if !ok {
				t.Fatalf("Failed to get node_id from content")
			}
			
			// Create project item
			itemID, err = client.CreateProjectItem(destProject.ID, contentID)
			if err != nil {
				t.Fatalf("Failed to create project item: %v", err)
			}
			t.Logf("Created project item with ID: %s", itemID)
		}
		
		// Set field values if any
		for fieldName, fieldValue := range item.Fields {
			if field, exists := fieldMap[fieldName]; exists {
				convertedValue, err := convertFieldValue(fieldValue, field)
				if err != nil {
					t.Logf("Warning: failed to convert field %s: %v", fieldName, err)
					continue
				}
				
				err = client.SetProjectItemFieldValue(destProject.ID, itemID, field.ID, convertedValue)
				if err != nil {
					t.Logf("Warning: failed to set field %s: %v", fieldName, err)
					continue
				}
				
				t.Logf("Set field %s to %v", fieldName, convertedValue)
			} else {
				t.Logf("Warning: field %s not found in destination project", fieldName)
			}
		}
	}
	
	t.Log("Integration test completed successfully!")
}

// TestRealProjectCreationAndDeletion tests basic project lifecycle
func TestRealProjectCreationAndDeletion(t *testing.T) {
	// Skip if not explicitly enabled
	if os.Getenv("REAL_INTEGRATION_TESTS") != "true" {
		t.Skip("Real integration tests not enabled - set REAL_INTEGRATION_TESTS=true to run")
	}
	
	client, err := NewGitHubClient()
	if err != nil {
		t.Fatalf("Failed to create GitHub client: %v", err)
	}
	
	// Get current user
	user, err := client.GetUser()
	if err != nil {
		t.Fatalf("Failed to get authenticated user: %v", err)
	}
	
	timestamp := time.Now().Format("20060102-150405")
	projectTitle := "Test Project Lifecycle " + timestamp
	
	// Create project
	project, err := client.CreateProject("user", user, projectTitle, "Test project for lifecycle testing")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	
	t.Logf("Created project: %s (ID: %s)", project.Title, project.ID)
	
	// Validate project properties
	if project.Title != projectTitle {
		t.Errorf("Expected title %s, got %s", projectTitle, project.Title)
	}
	
	if project.ID == "" {
		t.Error("Project ID should not be empty")
	}
	
	if project.Number <= 0 {
		t.Error("Project number should be positive")
	}
	
	if !strings.Contains(project.URL, user) {
		t.Errorf("Project URL should contain user: %s", project.URL)
	}
	
	// Test finding the project
	foundProject, err := client.FindProject(project.ID)
	if err != nil {
		t.Fatalf("Failed to find project by ID: %v", err)
	}
	
	if foundProject.ID != project.ID {
		t.Errorf("Found project ID mismatch: expected %s, got %s", project.ID, foundProject.ID)
	}
	
	// Delete project
	err = client.DeleteProject(project.ID)
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}
	
	t.Log("Project lifecycle test completed successfully!")
}

// TestRealErrorHandling tests error scenarios with real API calls
func TestRealErrorHandling(t *testing.T) {
	// Skip if not explicitly enabled
	if os.Getenv("REAL_INTEGRATION_TESTS") != "true" {
		t.Skip("Real integration tests not enabled - set REAL_INTEGRATION_TESTS=true to run")
	}
	
	client, err := NewGitHubClient()
	if err != nil {
		t.Fatalf("Failed to create GitHub client: %v", err)
	}
	
	// Test with invalid project ID
	_, err = client.FindProject("INVALID_PROJECT_ID")
	if err == nil {
		t.Error("Expected error when finding invalid project")
	}
	t.Logf("Correctly handled invalid project ID: %v", err)
	
	// Test with invalid user ID for project creation
	_, err = client.CreateProject("user", "INVALID_USER_NAME_THAT_DOES_NOT_EXIST", "Test", "Test description")
	if err == nil {
		t.Error("Expected error when creating project for invalid user")
	}
	t.Logf("Correctly handled invalid user: %v", err)
	
	// Test deleting non-existent project
	err = client.DeleteProject("PVT_INVALID_PROJECT_ID")
	if err == nil {
		t.Error("Expected error when deleting non-existent project")
	}
	t.Logf("Correctly handled project deletion error: %v", err)
	
	t.Log("Error handling test completed successfully!")
}