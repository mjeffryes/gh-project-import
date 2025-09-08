// Snapshot tests for GitHub Project Import Extension
// Tests GitHub API interactions using recorded snapshots
package main

import (
	"os"
	"testing"
)

// TestSnapshotGetUser tests the GetUser API call with snapshots
func TestSnapshotGetUser(t *testing.T) {
	client, err := NewSnapshotGitHubClient("GetUser")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()
	
	user, err := client.GetUser()
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	if user == "" {
		t.Error("Expected non-empty user login")
	}
	
	t.Logf("User: %s", user)
}

// TestSnapshotFindProject tests project discovery with snapshots
func TestSnapshotFindProject(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expectError bool
	}{
		{
			name:       "valid project identifier",
			identifier: "test/project",
			expectError: false,
		},
		{
			name:       "numeric project identifier", 
			identifier: "123",
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testName := "FindProject_" + tt.name
			client, err := NewSnapshotGitHubClient(testName)
			if err != nil {
				t.Fatalf("Failed to create snapshot client: %v", err)
			}
			defer client.Close()
			
			project, err := client.FindProject(tt.identifier)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if project == nil {
					t.Error("Expected project but got nil")
				} else {
					t.Logf("Project: %s (ID: %s)", project.Title, project.ID)
				}
			}
		})
	}
}

// TestSnapshotGetProjectFields tests field schema retrieval with snapshots
func TestSnapshotGetProjectFields(t *testing.T) {
	client, err := NewSnapshotGitHubClient("GetProjectFields")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()
	
	projectID := "PVT_test123"
	fields, err := client.GetProjectFields(projectID)
	if err != nil {
		t.Fatalf("Failed to get project fields: %v", err)
	}
	
	if len(fields) == 0 {
		t.Error("Expected at least one field")
	}
	
	// Verify field structure
	for i, field := range fields {
		t.Logf("Field %d: %s (%s)", i+1, field.Name, field.Type)
		
		if field.ID == "" {
			t.Errorf("Field %d missing ID", i+1)
		}
		if field.Name == "" {
			t.Errorf("Field %d missing name", i+1)
		}
		if field.Type == "" {
			t.Errorf("Field %d missing type", i+1)
		}
		
		// Check single-select fields have options
		if field.Type == "SINGLE_SELECT" && len(field.Options) == 0 {
			t.Errorf("Single-select field %s has no options", field.Name)
		}
	}
}

// TestSnapshotCreateDraftIssue tests draft issue creation with snapshots
func TestSnapshotCreateDraftIssue(t *testing.T) {
	client, err := NewSnapshotGitHubClient("CreateDraftIssue")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()
	
	projectID := "PVT_test123"
	title := "Test Draft Issue"
	body := "This is a test draft issue created by snapshot tests"
	
	itemID, err := client.CreateDraftIssue(projectID, title, body)
	if err != nil {
		t.Fatalf("Failed to create draft issue: %v", err)
	}
	
	if itemID == "" {
		t.Error("Expected non-empty item ID")
	}
	
	t.Logf("Created draft issue with ID: %s", itemID)
}

// TestSnapshotCreateProjectItem tests adding existing issues/PRs to projects
func TestSnapshotCreateProjectItem(t *testing.T) {
	client, err := NewSnapshotGitHubClient("CreateProjectItem")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()
	
	projectID := "PVT_test123"
	contentID := "ISSUE_test789"
	
	itemID, err := client.CreateProjectItem(projectID, contentID)
	if err != nil {
		t.Fatalf("Failed to create project item: %v", err)
	}
	
	if itemID == "" {
		t.Error("Expected non-empty item ID")
	}
	
	t.Logf("Created project item with ID: %s", itemID)
}

// TestSnapshotGetIssueOrPR tests issue/PR retrieval with snapshots
func TestSnapshotGetIssueOrPR(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "github issue",
			url:  "https://github.com/test/repo/issues/123",
		},
		{
			name: "github pull request",
			url:  "https://github.com/test/repo/pull/456",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testName := "GetIssueOrPR_" + tt.name
			client, err := NewSnapshotGitHubClient(testName)
			if err != nil {
				t.Fatalf("Failed to create snapshot client: %v", err)
			}
			defer client.Close()
			
			content, err := client.GetIssueOrPR(tt.url)
			if err != nil {
				t.Fatalf("Failed to get issue/PR: %v", err)
			}
			
			if content == nil {
				t.Error("Expected content but got nil")
			}
			
			// Check required fields
			if nodeID, ok := content["node_id"].(string); !ok || nodeID == "" {
				t.Error("Expected node_id in response")
			}
			
			if title, ok := content["title"].(string); !ok || title == "" {
				t.Error("Expected title in response")
			}
			
			t.Logf("Retrieved %s: %v", tt.name, content["title"])
		})
	}
}

// TestSnapshotSetProjectItemFieldValue tests field value setting with snapshots
func TestSnapshotSetProjectItemFieldValue(t *testing.T) {
	client, err := NewSnapshotGitHubClient("SetProjectItemFieldValue")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()
	
	projectID := "PVT_test123"
	itemID := "ITEM_test456"
	fieldID := "field1"
	value := map[string]interface{}{"singleSelectOptionId": "opt2"}
	
	err = client.SetProjectItemFieldValue(projectID, itemID, fieldID, value)
	if err != nil {
		t.Fatalf("Failed to set field value: %v", err)
	}
	
	t.Log("Successfully set field value")
}

// TestSnapshotEndToEndWorkflow tests a complete workflow with snapshots
func TestSnapshotEndToEndWorkflow(t *testing.T) {
	client, err := NewSnapshotGitHubClient("EndToEndWorkflow")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()
	
	// 1. Get authenticated user
	user, err := client.GetUser()
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	t.Logf("Authenticated as: %s", user)
	
	// 2. Find project
	project, err := client.FindProject("test/project")
	if err != nil {
		t.Fatalf("Failed to find project: %v", err)
	}
	t.Logf("Found project: %s", project.Title)
	
	// 3. Get project fields
	fields, err := client.GetProjectFields(project.ID)
	if err != nil {
		t.Fatalf("Failed to get project fields: %v", err)
	}
	t.Logf("Retrieved %d fields", len(fields))
	
	// 4. Create a draft issue
	itemID, err := client.CreateDraftIssue(project.ID, "Test Item", "Test description")
	if err != nil {
		t.Fatalf("Failed to create draft issue: %v", err)
	}
	t.Logf("Created draft issue: %s", itemID)
	
	// 5. Set field values
	if len(fields) > 0 {
		field := fields[0]
		var value interface{}
		
		switch field.Type {
		case "SINGLE_SELECT":
			if len(field.Options) > 0 {
				value = map[string]interface{}{"singleSelectOptionId": field.Options[0].ID}
			}
		case "TEXT":
			value = map[string]interface{}{"text": "Test value"}
		case "NUMBER":
			value = map[string]interface{}{"number": 42}
		}
		
		if value != nil {
			err = client.SetProjectItemFieldValue(project.ID, itemID, field.ID, value)
			if err != nil {
				t.Fatalf("Failed to set field value: %v", err)
			}
			t.Logf("Set field %s to %v", field.Name, value)
		}
	}
}

// TestSnapshotModes tests different snapshot modes
func TestSnapshotModes(t *testing.T) {
	// Test that snapshot mode can be controlled via environment variables
	originalMode := os.Getenv("SNAPSHOT_MODE")
	defer os.Setenv("SNAPSHOT_MODE", originalMode) // Restore
	
	// Test replay mode (default)
	os.Setenv("SNAPSHOT_MODE", "replay")
	mode := getSnapshotMode()
	if mode != SnapshotModeReplay {
		t.Errorf("Expected replay mode, got %v", mode)
	}
	
	// Test record mode
	os.Setenv("SNAPSHOT_MODE", "record")
	mode = getSnapshotMode()
	if mode != SnapshotModeRecord {
		t.Errorf("Expected record mode, got %v", mode)
	}
	
	// Test bypass mode
	os.Setenv("SNAPSHOT_MODE", "bypass")
	mode = getSnapshotMode()
	if mode != SnapshotModeBypass {
		t.Errorf("Expected bypass mode, got %v", mode)
	}
}

// TestSnapshotDirectory tests snapshot directory configuration
func TestSnapshotDirectory(t *testing.T) {
	originalDir := os.Getenv("SNAPSHOT_DIR")
	defer os.Setenv("SNAPSHOT_DIR", originalDir) // Restore
	
	// Test default directory
	os.Unsetenv("SNAPSHOT_DIR")
	dir := getSnapshotDir()
	expectedDefault := "testdata/snapshots"
	if dir != expectedDefault {
		t.Errorf("Expected default directory %s, got %s", expectedDefault, dir)
	}
	
	// Test custom directory
	customDir := "/tmp/test-snapshots"
	os.Setenv("SNAPSHOT_DIR", customDir)
	dir = getSnapshotDir()
	if dir != customDir {
		t.Errorf("Expected custom directory %s, got %s", customDir, dir)
	}
}