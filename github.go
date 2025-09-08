// GitHub API wrapper functions for project import operations
// Provides functions to interact with GitHub Projects v2 API
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Project represents a GitHub Projects v2 project
type Project struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// ProjectField represents a field in a GitHub project
type ProjectField struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"dataType"`
	Options []ProjectFieldOption `json:"options,omitempty"`
}

// ProjectFieldOption represents an option for single-select fields
type ProjectFieldOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProjectItem represents an item in a GitHub project
type ProjectItem struct {
	ID      string                 `json:"id"`
	Content map[string]interface{} `json:"content"`
	Fields  map[string]interface{} `json:"fieldValues"`
}

// GitHubClient wraps the GitHub API client
type GitHubClient struct {
	client api.RESTClient
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient() (*GitHubClient, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	
	return &GitHubClient{client: *client}, nil
}

// GetUser returns the authenticated user information
func (gc *GitHubClient) GetUser() (string, error) {
	response := struct {
		Login string `json:"login"`
	}{}
	
	err := gc.client.Get("user", &response)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	
	return response.Login, nil
}

// FindProject finds a project by identifier (owner/project-name or project-number)
func (gc *GitHubClient) FindProject(identifier string) (*Project, error) {
	// Check if identifier is a number (project number)
	if num, err := strconv.Atoi(identifier); err == nil {
		return gc.findProjectByNumber(num)
	}
	
	// Parse owner/project-name format
	parts := strings.Split(identifier, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid project identifier format: %s (expected owner/project-name or project-number)", identifier)
	}
	
	owner := parts[0]
	projectName := strings.Join(parts[1:], "/")
	
	return gc.findProjectByName(owner, projectName)
}

// findProjectByNumber finds a project by its number
func (gc *GitHubClient) findProjectByNumber(number int) (*Project, error) {
	query := fmt.Sprintf(`
		query {
			node(id: "PVT_kwDO%d") {
				... on ProjectV2 {
					id
					number
					title
					url
				}
			}
		}
	`, number)
	
	return gc.executeGraphQLQuery(query, nil, func(data map[string]interface{}) (*Project, error) {
		nodeData, ok := data["node"].(map[string]interface{})
		if !ok || nodeData == nil {
			return nil, fmt.Errorf("project with number %d not found", number)
		}
		
		return &Project{
			ID:          getString(nodeData, "id"),
			Number:      getInt(nodeData, "number"),
			Title:       getString(nodeData, "title"),
			URL:         getString(nodeData, "url"),
		}, nil
	})
}

// findProjectByName finds a project by owner and name
func (gc *GitHubClient) findProjectByName(owner, name string) (*Project, error) {
	// First, determine if owner is an organization or user
	isOrg, err := gc.isOrganization(owner)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if %s is organization: %w", owner, err)
	}
	
	var query string
	if isOrg {
		query = fmt.Sprintf(`
			query {
				organization(login: "%s") {
					projectsV2(first: 100, query: "%s") {
						nodes {
							id
							number
							title
							url
						}
					}
				}
			}
		`, owner, name)
	} else {
		query = fmt.Sprintf(`
			query {
				user(login: "%s") {
					projectsV2(first: 100, query: "%s") {
						nodes {
							id
							number
							title
							url
						}
					}
				}
			}
		`, owner, name)
	}
	
	return gc.executeGraphQLQuery(query, nil, func(data map[string]interface{}) (*Project, error) {
		var projects []Project
		
		if isOrg {
			if orgData, ok := data["organization"].(map[string]interface{}); ok {
				if projectsData, ok := orgData["projectsV2"].(map[string]interface{}); ok {
					if nodes, ok := projectsData["nodes"].([]interface{}); ok {
						for _, node := range nodes {
							if nodeMap, ok := node.(map[string]interface{}); ok {
								projects = append(projects, Project{
									ID:          getString(nodeMap, "id"),
									Number:      getInt(nodeMap, "number"),
									Title:       getString(nodeMap, "title"),
									URL:         getString(nodeMap, "url"),
								})
							}
						}
					}
				}
			}
		} else {
			if userData, ok := data["user"].(map[string]interface{}); ok {
				if projectsData, ok := userData["projectsV2"].(map[string]interface{}); ok {
					if nodes, ok := projectsData["nodes"].([]interface{}); ok {
						for _, node := range nodes {
							if nodeMap, ok := node.(map[string]interface{}); ok {
								projects = append(projects, Project{
									ID:          getString(nodeMap, "id"),
									Number:      getInt(nodeMap, "number"),
									Title:       getString(nodeMap, "title"),
									URL:         getString(nodeMap, "url"),
								})
							}
						}
					}
				}
			}
		}
		
		// Find exact match by title
		for _, project := range projects {
			if project.Title == name {
				return &project, nil
			}
		}
		
		return nil, fmt.Errorf("project %s/%s not found", owner, name)
	})
}

// isOrganization checks if the given login is an organization
func (gc *GitHubClient) isOrganization(login string) (bool, error) {
	response := struct {
		Type string `json:"type"`
	}{}
	
	err := gc.client.Get("users/"+login, &response)
	if err != nil {
		return false, err
	}
	
	return response.Type == "Organization", nil
}

// GetProjectFields retrieves the field schema for a project
func (gc *GitHubClient) GetProjectFields(projectID string) ([]ProjectField, error) {
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
							... on ProjectV2IterationField {
								id
								name
								dataType
							}
						}
					}
				}
			}
		}
	`, projectID)
	
	payload := map[string]interface{}{
		"query": query,
	}
	
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}
	
	var response struct {
		Data struct {
			Node struct {
				Fields struct {
					Nodes []json.RawMessage `json:"nodes"`
				} `json:"fields"`
			} `json:"node"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	
	err = gc.client.Post("graphql", bytes.NewReader(jsonBytes), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get project fields: %w", err)
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", response.Errors[0].Message)
	}
	
	var fields []ProjectField
	for _, node := range response.Data.Node.Fields.Nodes {
		var field ProjectField
		if err := json.Unmarshal(node, &field); err != nil {
			continue // Skip fields we can't parse
		}
		fields = append(fields, field)
	}
	
	return fields, nil
}

// CreateProjectItem creates a new item in the specified project
func (gc *GitHubClient) CreateProjectItem(projectID, contentID string) (string, error) {
	mutation := `
		mutation($projectId: ID!, $contentId: ID!) {
			addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
				item {
					id
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"projectId": projectID,
		"contentId": contentID,
	}
	
	data, err := gc.executeGraphQLMutation(mutation, variables)
	if err != nil {
		return "", fmt.Errorf("failed to create project item: %w", err)
	}
	
	if addData, ok := data["addProjectV2ItemById"].(map[string]interface{}); ok {
		if itemData, ok := addData["item"].(map[string]interface{}); ok {
			return getString(itemData, "id"), nil
		}
	}
	
	return "", fmt.Errorf("unexpected response format")
}

// CreateDraftIssue creates a draft issue and returns its ID
func (gc *GitHubClient) CreateDraftIssue(projectID, title, body string) (string, error) {
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
	
	data, err := gc.executeGraphQLMutation(mutation, variables)
	if err != nil {
		return "", fmt.Errorf("failed to create draft issue: %w", err)
	}
	
	if addData, ok := data["addProjectV2DraftIssue"].(map[string]interface{}); ok {
		if itemData, ok := addData["projectItem"].(map[string]interface{}); ok {
			return getString(itemData, "id"), nil
		}
	}
	
	return "", fmt.Errorf("unexpected response format")
}

// SetProjectItemFieldValue sets a field value for a project item
func (gc *GitHubClient) SetProjectItemFieldValue(projectID, itemID, fieldID string, value interface{}) error {
	mutation := `
		mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: ProjectV2FieldValue!) {
			updateProjectV2ItemFieldValue(input: {
				projectId: $projectId, 
				itemId: $itemId, 
				fieldId: $fieldId, 
				value: $value
			}) {
				projectV2Item {
					id
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"value":     value,
	}
	
	_, err := gc.executeGraphQLMutation(mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to set field value: %w", err)
	}
	
	return nil
}

// ParseRepositoryURL extracts owner and repository name from GitHub URL
func ParseRepositoryURL(url string) (string, string, error) {
	// Regular expression to match GitHub URLs
	re := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)`)
	matches := re.FindStringSubmatch(url)
	
	if len(matches) < 3 {
		return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
	}
	
	return matches[1], matches[2], nil
}

// GetIssueOrPR retrieves issue or PR information by URL
func (gc *GitHubClient) GetIssueOrPR(url string) (map[string]interface{}, error) {
	owner, repo, err := ParseRepositoryURL(url)
	if err != nil {
		return nil, err
	}
	
	// Extract issue/PR number from URL
	re := regexp.MustCompile(`/(?:issues|pull)/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract issue/PR number from URL: %s", url)
	}
	
	number := matches[1]
	
	// Try to get as issue first, then as PR
	var response map[string]interface{}
	
	// Check if it's an issue
	err = gc.client.Get(fmt.Sprintf("repos/%s/%s/issues/%s", owner, repo, number), &response)
	if err != nil {
		// Try as PR
		err = gc.client.Get(fmt.Sprintf("repos/%s/%s/pulls/%s", owner, repo, number), &response)
		if err != nil {
			return nil, fmt.Errorf("failed to get issue/PR %s: %w", url, err)
		}
	}
	
	return response, nil
}

// executeGraphQLQuery executes a GraphQL query and processes the response
func (gc *GitHubClient) executeGraphQLQuery(query string, variables map[string]interface{}, processor func(map[string]interface{}) (*Project, error)) (*Project, error) {
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}
	
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}
	
	var response struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	
	err = gc.client.Post("graphql", bytes.NewReader(jsonBytes), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", response.Errors[0].Message)
	}
	
	return processor(response.Data)
}

// executeGraphQLMutation executes a GraphQL mutation
func (gc *GitHubClient) executeGraphQLMutation(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"query":     mutation,
		"variables": variables,
	}
	
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mutation: %w", err)
	}
	
	var response struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	
	err = gc.client.Post("graphql", bytes.NewReader(jsonBytes), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL mutation: %w", err)
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", response.Errors[0].Message)
	}
	
	return response.Data, nil
}

// Helper functions to safely extract values from maps
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return int(num)
		}
	}
	return 0
}