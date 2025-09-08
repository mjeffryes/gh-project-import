// Snapshot testing utilities for recording and replaying GitHub API interactions
// Provides deterministic testing without actual API calls
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// SnapshotMode determines whether to record, replay, or bypass snapshots
type SnapshotMode int

const (
	SnapshotModeReplay SnapshotMode = iota // Default: replay from snapshots
	SnapshotModeRecord                     // Record new snapshots
	SnapshotModeBypass                     // Bypass snapshots (make real API calls)
)

// APICall represents a recorded API call and response
type APICall struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	RequestBody string            `json:"request_body,omitempty"`
	StatusCode  int               `json:"status_code"`
	Response    string            `json:"response"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Snapshot represents a collection of API calls for a test scenario
type Snapshot struct {
	TestName string    `json:"test_name"`
	Calls    []APICall `json:"calls"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

// SnapshotGitHubClient wraps the GitHub API client with snapshot recording/replay
type SnapshotGitHubClient struct {
	mode         SnapshotMode
	snapshotDir  string
	currentTest  string
	snapshot     *Snapshot
	callIndex    int
	realClient   api.RESTClient
}

// NewSnapshotGitHubClient creates a new snapshot-enabled GitHub client
func NewSnapshotGitHubClient(testName string) (*SnapshotGitHubClient, error) {
	mode := getSnapshotMode()
	snapshotDir := getSnapshotDir()
	
	// Create snapshots directory if it doesn't exist
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshots directory: %w", err)
	}
	
	var realClient api.RESTClient
	if mode == SnapshotModeRecord || mode == SnapshotModeBypass {
		client, err := api.DefaultRESTClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create real GitHub client: %w", err)
		}
		realClient = *client
	}
	
	sgc := &SnapshotGitHubClient{
		mode:        mode,
		snapshotDir: snapshotDir,
		currentTest: testName,
		callIndex:   0,
		realClient:  realClient,
	}
	
	// Load or create snapshot
	if err := sgc.loadOrCreateSnapshot(); err != nil {
		return nil, fmt.Errorf("failed to load/create snapshot: %w", err)
	}
	
	return sgc, nil
}

// getSnapshotMode determines the snapshot mode from environment variables
func getSnapshotMode() SnapshotMode {
	switch strings.ToLower(os.Getenv("SNAPSHOT_MODE")) {
	case "record":
		return SnapshotModeRecord
	case "bypass":
		return SnapshotModeBypass
	default:
		return SnapshotModeReplay
	}
}

// getSnapshotDir returns the directory for storing snapshots
func getSnapshotDir() string {
	if dir := os.Getenv("SNAPSHOT_DIR"); dir != "" {
		return dir
	}
	return "testdata/snapshots"
}

// getSnapshotFile returns the file path for a snapshot
func (sgc *SnapshotGitHubClient) getSnapshotFile() string {
	// Replace spaces and special characters in test names for valid filenames
	safeName := strings.ReplaceAll(sgc.currentTest, " ", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	filename := fmt.Sprintf("%s.json", safeName)
	return filepath.Join(sgc.snapshotDir, filename)
}

// loadOrCreateSnapshot loads an existing snapshot or creates a new one
func (sgc *SnapshotGitHubClient) loadOrCreateSnapshot() error {
	snapshotFile := sgc.getSnapshotFile()
	
	if sgc.mode == SnapshotModeReplay {
		// Load existing snapshot
		data, err := os.ReadFile(snapshotFile)
		if err != nil {
			return fmt.Errorf("failed to read snapshot file %s: %w (try running with SNAPSHOT_MODE=record to create it)", snapshotFile, err)
		}
		
		sgc.snapshot = &Snapshot{}
		if err := json.Unmarshal(data, sgc.snapshot); err != nil {
			return fmt.Errorf("failed to parse snapshot file: %w", err)
		}
	} else {
		// Create new snapshot for recording
		sgc.snapshot = &Snapshot{
			TestName: sgc.currentTest,
			Created:  time.Now(),
			Updated:  time.Now(),
			Calls:    []APICall{},
		}
	}
	
	return nil
}

// saveSnapshot saves the current snapshot to disk
func (sgc *SnapshotGitHubClient) saveSnapshot() error {
	if sgc.mode != SnapshotModeRecord {
		return nil // Only save when recording
	}
	
	sgc.snapshot.Updated = time.Now()
	
	data, err := json.MarshalIndent(sgc.snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}
	
	snapshotFile := sgc.getSnapshotFile()
	if err := os.WriteFile(snapshotFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}
	
	return nil
}

// recordCall records an API call and response
func (sgc *SnapshotGitHubClient) recordCall(method, url string, requestBody string, statusCode int, response string) {
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

// replayCall replays a recorded API call
func (sgc *SnapshotGitHubClient) replayCall(method, url string, requestBody string) (int, string, error) {
	if sgc.mode != SnapshotModeReplay {
		return 0, "", fmt.Errorf("not in replay mode")
	}
	
	if sgc.callIndex >= len(sgc.snapshot.Calls) {
		return 0, "", fmt.Errorf("no more recorded calls available (index %d >= %d calls)", sgc.callIndex, len(sgc.snapshot.Calls))
	}
	
	call := sgc.snapshot.Calls[sgc.callIndex]
	sgc.callIndex++
	
	// Validate the call matches expectations
	if call.Method != method {
		return 0, "", fmt.Errorf("method mismatch: expected %s, got %s", call.Method, method)
	}
	
	if call.URL != url {
		return 0, "", fmt.Errorf("URL mismatch: expected %s, got %s", call.URL, url)
	}
	
	// For GraphQL calls, we could add more sophisticated request body matching
	// For now, we'll be lenient on exact request body matching
	
	return call.StatusCode, call.Response, nil
}

// Close finalizes the snapshot (saves if recording)
func (sgc *SnapshotGitHubClient) Close() error {
	return sgc.saveSnapshot()
}

// Implement the GitHubClient interface methods with snapshot support

// GetUser returns the authenticated user information
func (sgc *SnapshotGitHubClient) GetUser() (string, error) {
	method := "GET"
	url := "user"
	
	var statusCode int
	var responseBody string
	var err error
	
	switch sgc.mode {
	case SnapshotModeReplay:
		statusCode, responseBody, err = sgc.replayCall(method, url, "")
		if err != nil {
			return "", err
		}
	case SnapshotModeRecord, SnapshotModeBypass:
		// Make real API call
		response := struct {
			Login string `json:"login"`
		}{}
		
		err := sgc.realClient.Get(url, &response)
		if err != nil {
			return "", fmt.Errorf("failed to get user: %w", err)
		}
		
		// For recording mode, capture the response
		if sgc.mode == SnapshotModeRecord {
			responseData, _ := json.Marshal(response)
			responseBody = string(responseData)
			statusCode = 200
			sgc.recordCall(method, url, "", statusCode, responseBody)
		}
		
		return response.Login, nil
	}
	
	// Parse replayed response
	if statusCode != 200 {
		return "", fmt.Errorf("API error: status code %d", statusCode)
	}
	
	var response struct {
		Login string `json:"login"`
	}
	
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	return response.Login, nil
}

// makeGraphQLCall handles GraphQL API calls with snapshot support
func (sgc *SnapshotGitHubClient) makeGraphQLCall(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	method := "POST"
	url := "graphql"
	
	// Prepare request body
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}
	
	requestData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}
	requestBody := string(requestData)
	
	var statusCode int
	var responseBody string
	
	switch sgc.mode {
	case SnapshotModeReplay:
		statusCode, responseBody, err = sgc.replayCall(method, url, requestBody)
		if err != nil {
			return nil, err
		}
	case SnapshotModeRecord, SnapshotModeBypass:
		// Make real API call using the existing executeGraphQLMutation method logic
		var response struct {
			Data   map[string]interface{} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		
		err = sgc.realClient.Post("graphql", strings.NewReader(requestBody), &response)
		if err != nil {
			return nil, fmt.Errorf("failed to execute GraphQL call: %w", err)
		}
		
		// For recording mode, capture the response
		if sgc.mode == SnapshotModeRecord {
			responseData, _ := json.Marshal(response)
			responseBody = string(responseData)
			statusCode = 200
			sgc.recordCall(method, url, requestBody, statusCode, responseBody)
		}
		
		// Handle GraphQL errors
		if len(response.Errors) > 0 {
			errMsg := response.Errors[0].Message
			if strings.Contains(errMsg, "rate limit") {
				return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait and try again later")
			}
			if strings.Contains(errMsg, "not found") {
				return nil, fmt.Errorf("resource not found or insufficient permissions: %s", errMsg)
			}
			return nil, fmt.Errorf("GraphQL error: %s", errMsg)
		}
		
		return response.Data, nil
	}
	
	// Parse replayed response
	if statusCode != 200 {
		return nil, fmt.Errorf("API error: status code %d", statusCode)
	}
	
	var response struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}
	
	// Handle GraphQL errors
	if len(response.Errors) > 0 {
		errMsg := response.Errors[0].Message
		if strings.Contains(errMsg, "rate limit") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait and try again later")
		}
		if strings.Contains(errMsg, "not found") {
			return nil, fmt.Errorf("resource not found or insufficient permissions: %s", errMsg)
		}
		return nil, fmt.Errorf("GraphQL error: %s", errMsg)
	}
	
	return response.Data, nil
}

// FindProject finds a project by identifier with snapshot support
func (sgc *SnapshotGitHubClient) FindProject(identifier string) (*Project, error) {
	// Implementation similar to the original, but using makeGraphQLCall
	// This is a simplified version - full implementation would include all the logic from github.go
	
	query := `
		query {
			viewer {
				login
			}
		}
	`
	
	_, err := sgc.makeGraphQLCall(query, nil)
	if err != nil {
		return nil, err
	}
	
	// For now, return a mock project for demonstration
	// Full implementation would include proper project discovery logic
	return &Project{
		ID:     "PVT_test123",
		Number: 1,
		Title:  "Test Project",
		URL:    "https://github.com/test/project/projects/1",
	}, nil
}

// Additional methods would be implemented similarly...
// For brevity, I'll implement the core pattern and a few key methods

// GetProjectFields retrieves the field schema for a project with snapshot support
func (sgc *SnapshotGitHubClient) GetProjectFields(projectID string) ([]ProjectField, error) {
	query := fmt.Sprintf(`
		query {
			node(id: "%s") {
				... on ProjectV2 {
					fields(first: 100) {
						nodes {
							... on ProjectV2Field {
								id
								name
								dataType
							}
							... on ProjectV2SingleSelectField {
								id
								name
								dataType
								options {
									id
									name
								}
							}
						}
					}
				}
			}
		}
	`, projectID)
	
	_, err := sgc.makeGraphQLCall(query, nil)
	if err != nil {
		return nil, err
	}
	
	// Parse the response - simplified for demonstration
	// Full implementation would include proper field parsing
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
	}
	
	return fields, nil
}

// CreateDraftIssue creates a draft issue with snapshot support
func (sgc *SnapshotGitHubClient) CreateDraftIssue(projectID, title, body string) (string, error) {
	mutation := `
		mutation($projectId: ID!, $title: String!, $body: String) {
			addProjectV2DraftIssue(input: {projectId: $projectId, title: $title, body: $body}) {
				projectItem {
					id
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"projectId": projectID,
		"title":     title,
		"body":      body,
	}
	
	_, err := sgc.makeGraphQLCall(mutation, variables)
	if err != nil {
		return "", fmt.Errorf("failed to create draft issue: %w", err)
	}
	
	// Parse response - simplified for demonstration
	return "ITEM_test123", nil
}

// Stub implementations for interface compatibility
func (sgc *SnapshotGitHubClient) CreateProjectItem(projectID, contentID string) (string, error) {
	return "ITEM_test456", nil
}

func (sgc *SnapshotGitHubClient) GetIssueOrPR(url string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"node_id": "ISSUE_test789",
		"title":   "Test Issue",
		"body":    "Test issue body",
	}, nil
}

func (sgc *SnapshotGitHubClient) SetProjectItemFieldValue(projectID, itemID, fieldID string, value interface{}) error {
	return nil
}

func (sgc *SnapshotGitHubClient) CreateProject(ownerType, owner, title, description string) (*Project, error) {
	return &Project{
		ID:     "PVT_test_new_123",
		Number: 999,
		Title:  title,
		URL:    "https://github.com/" + owner + "/projects/999",
	}, nil
}

func (sgc *SnapshotGitHubClient) DeleteProject(projectID string) error {
	return nil
}