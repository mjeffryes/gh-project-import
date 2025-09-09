// Snapshot testing framework for GitHub API interactions
// Records and replays GitHub API calls for deterministic testing
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SnapshotMode defines the operating mode for snapshot tests
type SnapshotMode int

const (
	SnapshotModeReplay SnapshotMode = iota // Default: replay from snapshots
	SnapshotModeRecord                     // Record new snapshots from real API calls
	SnapshotModeBypass                     // Make real API calls without recording
)

// APICall represents a single API call in a snapshot
type APICall struct {
	Method      string    `json:"method"`
	URL         string    `json:"url"`
	RequestBody string    `json:"request_body,omitempty"`
	StatusCode  int       `json:"status_code"`
	Response    string    `json:"response"`
	Timestamp   time.Time `json:"timestamp"`
}

// Snapshot represents a complete test scenario with multiple API calls
type Snapshot struct {
	TestName string    `json:"test_name"`
	Calls    []APICall `json:"calls"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

// SnapshotGitHubClient wraps GitHubClient to provide snapshot functionality
type SnapshotGitHubClient struct {
	realClient  GitHubClient
	mode        SnapshotMode
	snapshotDir string
	testName    string
	snapshot    *Snapshot
	callIndex   int
}

// NewSnapshotGitHubClient creates a new snapshot-enabled GitHub client
func NewSnapshotGitHubClient(testName string) (*SnapshotGitHubClient, error) {
	mode := getSnapshotMode()
	snapshotDir := getSnapshotDir()

	// Create snapshot directory if it doesn't exist
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	client := &SnapshotGitHubClient{
		mode:        mode,
		snapshotDir: snapshotDir,
		testName:    testName,
		callIndex:   0,
	}

	// For record and bypass modes, create a real GitHub client
	if mode == SnapshotModeRecord || mode == SnapshotModeBypass {
		realClient, err := NewGitHubClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create real GitHub client: %w", err)
		}
		client.realClient = realClient
	}

	// Load or create snapshot
	if err := client.loadOrCreateSnapshot(); err != nil {
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}

	return client, nil
}

// Close saves the snapshot if in record mode
func (sgc *SnapshotGitHubClient) Close() error {
	if sgc.mode == SnapshotModeRecord {
		return sgc.saveSnapshot()
	}
	return nil
}

// loadOrCreateSnapshot loads an existing snapshot or creates a new one
func (sgc *SnapshotGitHubClient) loadOrCreateSnapshot() error {
	snapshotPath := sgc.getSnapshotPath()

	if sgc.mode == SnapshotModeRecord {
		// In record mode, create a new snapshot
		sgc.snapshot = &Snapshot{
			TestName: sgc.testName,
			Calls:    []APICall{},
			Created:  time.Now(),
			Updated:  time.Now(),
		}
		return nil
	}

	// In replay mode, load existing snapshot
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot file not found: %s (try running with SNAPSHOT_MODE=record to create it)", snapshotPath)
	}

	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to read snapshot file: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to parse snapshot file: %w", err)
	}

	sgc.snapshot = &snapshot
	return nil
}

// saveSnapshot saves the current snapshot to disk
func (sgc *SnapshotGitHubClient) saveSnapshot() error {
	sgc.snapshot.Updated = time.Now()

	data, err := json.MarshalIndent(sgc.snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	snapshotPath := sgc.getSnapshotPath()
	if err := os.WriteFile(snapshotPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return nil
}

// getSnapshotPath returns the file path for the snapshot
func (sgc *SnapshotGitHubClient) getSnapshotPath() string {
	// Create safe filename from test name
	safeTestName := strings.ReplaceAll(sgc.testName, " ", "_")
	safeTestName = regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(safeTestName, "_")
	return filepath.Join(sgc.snapshotDir, safeTestName+".json")
}

// recordCall records an API call in record mode
func (sgc *SnapshotGitHubClient) recordCall(method, url, requestBody string, statusCode int, response string) {
	if sgc.mode != SnapshotModeRecord {
		return
	}

	call := APICall{
		Method:      method,
		URL:         url,
		RequestBody: requestBody,
		StatusCode:  statusCode,
		Response:    response,
		Timestamp:   time.Now(),
	}

	sgc.snapshot.Calls = append(sgc.snapshot.Calls, call)
}

// getNextCall returns the next expected call from the snapshot
func (sgc *SnapshotGitHubClient) getNextCall() (*APICall, error) {
	if sgc.callIndex >= len(sgc.snapshot.Calls) {
		return nil, fmt.Errorf("no more recorded calls available (call %d)", sgc.callIndex+1)
	}

	call := &sgc.snapshot.Calls[sgc.callIndex]
	sgc.callIndex++
	return call, nil
}

// executeWithSnapshot executes a function with snapshot recording/replay
func (sgc *SnapshotGitHubClient) executeWithSnapshot(
	operation string,
	realFunc func() (interface{}, error),
	parseResponse func(string) (interface{}, error),
) (interface{}, error) {
	switch sgc.mode {
	case SnapshotModeBypass:
		// Direct call without recording
		return realFunc()

	case SnapshotModeRecord:
		// Execute real call and record the result
		result, err := realFunc()
		if err != nil {
			// Record the error
			sgc.recordCall("API", operation, "", 500, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
			return nil, err
		}

		// Record successful response
		responseData, _ := json.Marshal(result)
		sgc.recordCall("API", operation, "", 200, string(responseData))
		return result, nil

	case SnapshotModeReplay:
		// Replay from snapshot
		call, err := sgc.getNextCall()
		if err != nil {
			return nil, err
		}

		if call.StatusCode != 200 {
			var errorResp struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal([]byte(call.Response), &errorResp); err == nil {
				return nil, fmt.Errorf(errorResp.Error)
			}
			return nil, fmt.Errorf("API error (status %d)", call.StatusCode)
		}

		return parseResponse(call.Response)

	default:
		return nil, fmt.Errorf("unknown snapshot mode: %v", sgc.mode)
	}
}

// GetUser implements GitHubClient interface
func (sgc *SnapshotGitHubClient) GetUser() (string, error) {
	result, err := sgc.executeWithSnapshot(
		"GetUser",
		func() (interface{}, error) {
			return sgc.realClient.GetUser()
		},
		func(response string) (interface{}, error) {
			return response, nil
		},
	)

	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// FindProject implements GitHubClient interface
func (sgc *SnapshotGitHubClient) FindProject(identifier string) (*Project, error) {
	result, err := sgc.executeWithSnapshot(
		"FindProject",
		func() (interface{}, error) {
			return sgc.realClient.FindProject(identifier)
		},
		func(response string) (interface{}, error) {
			var project Project
			if err := json.Unmarshal([]byte(response), &project); err != nil {
				return nil, err
			}
			return &project, nil
		},
	)

	if err != nil {
		return nil, err
	}
	return result.(*Project), nil
}

// GetProjectFields implements GitHubClient interface
func (sgc *SnapshotGitHubClient) GetProjectFields(projectID string) ([]ProjectField, error) {
	result, err := sgc.executeWithSnapshot(
		"GetProjectFields",
		func() (interface{}, error) {
			return sgc.realClient.GetProjectFields(projectID)
		},
		func(response string) (interface{}, error) {
			var fields []ProjectField
			if err := json.Unmarshal([]byte(response), &fields); err != nil {
				return nil, err
			}
			return fields, nil
		},
	)

	if err != nil {
		return nil, err
	}
	return result.([]ProjectField), nil
}

// CreateDraftIssue implements GitHubClient interface
func (sgc *SnapshotGitHubClient) CreateDraftIssue(projectID, title, body string) (string, error) {
	result, err := sgc.executeWithSnapshot(
		"CreateDraftIssue",
		func() (interface{}, error) {
			return sgc.realClient.CreateDraftIssue(projectID, title, body)
		},
		func(response string) (interface{}, error) {
			return response, nil
		},
	)

	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// CreateProjectItem implements GitHubClient interface
func (sgc *SnapshotGitHubClient) CreateProjectItem(projectID, contentID string) (string, error) {
	result, err := sgc.executeWithSnapshot(
		"CreateProjectItem",
		func() (interface{}, error) {
			return sgc.realClient.CreateProjectItem(projectID, contentID)
		},
		func(response string) (interface{}, error) {
			return response, nil
		},
	)

	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// GetIssueOrPR implements GitHubClient interface
func (sgc *SnapshotGitHubClient) GetIssueOrPR(url string) (map[string]interface{}, error) {
	result, err := sgc.executeWithSnapshot(
		"GetIssueOrPR",
		func() (interface{}, error) {
			return sgc.realClient.GetIssueOrPR(url)
		},
		func(response string) (interface{}, error) {
			var content map[string]interface{}
			if err := json.Unmarshal([]byte(response), &content); err != nil {
				return nil, err
			}
			return content, nil
		},
	)

	if err != nil {
		return nil, err
	}
	return result.(map[string]interface{}), nil
}

// SetProjectItemFieldValue implements GitHubClient interface
func (sgc *SnapshotGitHubClient) SetProjectItemFieldValue(projectID, itemID, fieldID string, value interface{}) error {
	_, err := sgc.executeWithSnapshot(
		"SetProjectItemFieldValue",
		func() (interface{}, error) {
			err := sgc.realClient.SetProjectItemFieldValue(projectID, itemID, fieldID, value)
			return "success", err
		},
		func(response string) (interface{}, error) {
			return "success", nil
		},
	)

	return err
}

// DeleteProjectItem implements GitHubClient interface
func (sgc *SnapshotGitHubClient) DeleteProjectItem(projectID, itemID string) error {
	_, err := sgc.executeWithSnapshot(
		"DeleteProjectItem",
		func() (interface{}, error) {
			err := sgc.realClient.DeleteProjectItem(projectID, itemID)
			return "success", err
		},
		func(response string) (interface{}, error) {
			return "success", nil
		},
	)

	return err
}

// Helper functions

// getSnapshotMode returns the current snapshot mode from environment
func getSnapshotMode() SnapshotMode {
	mode := strings.ToLower(os.Getenv("SNAPSHOT_MODE"))
	switch mode {
	case "record":
		return SnapshotModeRecord
	case "bypass":
		return SnapshotModeBypass
	default:
		return SnapshotModeReplay
	}
}

// getSnapshotDir returns the snapshot directory from environment or default
func getSnapshotDir() string {
	if dir := os.Getenv("SNAPSHOT_DIR"); dir != "" {
		return dir
	}
	return "testdata/snapshots"
}
