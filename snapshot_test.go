// Tests for Snapshot tooling
package main

import (
	"os"
	"testing"
)

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

// TestSnapshotDeleteProjectItem tests project item deletion with snapshots
func TestSnapshotDeleteProjectItem(t *testing.T) {
	client, err := NewSnapshotGitHubClient("DeleteProjectItem")
	if err != nil {
		t.Fatalf("Failed to create snapshot client: %v", err)
	}
	defer client.Close()

	projectID := "PVT_test123"
	itemID := "ITEM_test456"

	err = client.DeleteProjectItem(projectID, itemID)
	if err != nil {
		t.Fatalf("Failed to delete project item: %v", err)
	}

	t.Log("Successfully deleted project item")
}
