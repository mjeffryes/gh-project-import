// Tests GitHub API interactions using recorded snapshots
package main

import (
	"fmt"
	"testing"
)

// username for test projects
const testUsername = "mjeffryes"
const testProjectTitle = "Import Test Project"

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

// TestSnapshotFindProject tests the FindProject API call with snapshots
func TestSnapshotFindProject(t *testing.T) {
	client, err := NewSnapshotGitHubClient("GetUser")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()

	// Test finding project by name
	project, err := client.FindProject(fmt.Sprintf("%s/%s", testUsername, testProjectTitle))
	if err != nil {
		t.Fatalf("Failed to find project: %v", err)
	}
	t.Logf("Found project by name: %s", project.Title)

	/*
		 TODO: Find by number is not working correctly in the base github client
			// Test finding project by number
			project, err = client.FindProject(fmt.Sprintf("%s/%d", testUsername, project.Number))
			if err != nil {
				t.Fatalf("Failed to find project: %v", err)
			}
			t.Logf("Found project by number: %s", project.Title)
	*/
}

// TestSnapshotEndToEndWorkflow tests a complete workflow with snapshots
func TestSnapshotEndToEndWorkflow(t *testing.T) {
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

	// TODO: should clean up by deleting the created item
}
