// File parsing utilities for JSON and CSV import files
// Handles parsing and validation of input data for project import
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ImportItem represents a project item to be imported
type ImportItem struct {
	Title      string                 `json:"title"`
	URL        string                 `json:"url,omitempty"`
	Content    ItemContent            `json:"content,omitempty"`
	Assignees  []string               `json:"assignees,omitempty"`
	Repository string                 `json:"repository,omitempty"`
	Labels     []string               `json:"labels,omitempty"`
	Notes      string                 `json:"notes,omitempty"`
	Fields     map[string]interface{} `json:"-"` // All other fields
}

// ItemContent represents the content of a project item
type ItemContent struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Body       string `json:"body,omitempty"`
	Number     int    `json:"number,omitempty"`
	Repository string `json:"repository,omitempty"`
	URL        string `json:"url,omitempty"`
}

// ParseJSONFile parses a JSON file containing project items
func ParseJSONFile(filename string) ([]ImportItem, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Handle both array format and object with items array
	var items []ImportItem
	var rawItems []map[string]interface{}

	// First try to parse as array directly
	if err := json.Unmarshal(data, &rawItems); err != nil {
		// Try to parse as object with items array
		var wrapper struct {
			Items []map[string]interface{} `json:"items"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return nil, fmt.Errorf("failed to parse JSON file %s: %w", filename, err)
		}
		rawItems = wrapper.Items
	}

	// Convert raw items to ImportItem structs
	for i, rawItem := range rawItems {
		item, err := convertRawItemToImportItem(rawItem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse item %d: %w", i, err)
		}
		items = append(items, item)
	}

	return items, nil
}

// ParseCSVFile parses a CSV file containing project items
func ParseCSVFile(filename string) ([]ImportItem, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file %s: %w", filename, err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must have at least a header row and one data row")
	}

	headers := records[0]
	var items []ImportItem

	for i, record := range records[1:] {
		if len(record) != len(headers) {
			return nil, fmt.Errorf("row %d has %d fields, expected %d", i+2, len(record), len(headers))
		}

		item, err := convertCSVRecordToImportItem(headers, record)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV row %d: %w", i+2, err)
		}
		items = append(items, item)
	}

	return items, nil
}

// convertRawItemToImportItem converts a raw map to ImportItem
func convertRawItemToImportItem(rawItem map[string]interface{}) (ImportItem, error) {
	item := ImportItem{
		Fields: make(map[string]interface{}),
	}

	// Extract known fields
	if title, ok := rawItem["title"].(string); ok {
		item.Title = title
	}

	if url, ok := rawItem["url"].(string); ok {
		item.URL = url
	}

	if repo, ok := rawItem["repository"].(string); ok {
		item.Repository = repo
	}

	if notes, ok := rawItem["notes"].(string); ok {
		item.Notes = notes
	}

	// Handle assignees
	if assigneesRaw, ok := rawItem["assignees"]; ok {
		if assigneesList, ok := assigneesRaw.([]interface{}); ok {
			for _, assignee := range assigneesList {
				if assigneeStr, ok := assignee.(string); ok {
					item.Assignees = append(item.Assignees, assigneeStr)
				}
			}
		}
	}

	// Handle labels
	if labelsRaw, ok := rawItem["labels"]; ok {
		if labelsList, ok := labelsRaw.([]interface{}); ok {
			for _, label := range labelsList {
				if labelStr, ok := label.(string); ok {
					item.Labels = append(item.Labels, labelStr)
				}
			}
		}
	}

	// Handle content
	if contentRaw, ok := rawItem["content"].(map[string]interface{}); ok {
		item.Content = ItemContent{
			Type:       getString(contentRaw, "type"),
			Title:      getString(contentRaw, "title"),
			Body:       getString(contentRaw, "body"),
			Number:     getInt(contentRaw, "number"),
			Repository: getString(contentRaw, "repository"),
			URL:        getString(contentRaw, "url"),
		}
	}

	// Store all other fields in Fields map
	knownFields := map[string]bool{
		"title": true, "url": true, "repository": true, "assignees": true,
		"labels": true, "notes": true, "content": true, "id": true,
	}

	for key, value := range rawItem {
		if !knownFields[key] {
			item.Fields[key] = value
		}
	}

	// Validate required fields
	if item.Title == "" && item.Content.Title == "" {
		return item, fmt.Errorf("item must have either 'title' field or 'content.title'")
	}

	// Use content title if no top-level title
	if item.Title == "" {
		item.Title = item.Content.Title
	}

	return item, nil
}

// convertCSVRecordToImportItem converts a CSV record to ImportItem
func convertCSVRecordToImportItem(headers []string, record []string) (ImportItem, error) {
	item := ImportItem{
		Fields: make(map[string]interface{}),
	}

	for i, header := range headers {
		value := strings.TrimSpace(record[i])
		if value == "" {
			continue // Skip empty values
		}

		// Normalize header name
		normalizedHeader := strings.ToLower(strings.TrimSpace(header))

		switch normalizedHeader {
		case "title":
			item.Title = value
		case "url":
			item.URL = value
		case "repository":
			item.Repository = value
		case "notes":
			item.Notes = value
		case "assignees", "assignee":
			// Handle comma-separated assignees
			assignees := strings.Split(value, ",")
			for _, assignee := range assignees {
				if assignee = strings.TrimSpace(assignee); assignee != "" {
					item.Assignees = append(item.Assignees, assignee)
				}
			}
		case "labels", "label":
			// Handle comma-separated labels
			labels := strings.Split(value, ",")
			for _, label := range labels {
				if label = strings.TrimSpace(label); label != "" {
					item.Labels = append(item.Labels, label)
				}
			}
		default:
			// Try to parse as number if it looks like one
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				// Check if it's actually an integer
				if num == float64(int64(num)) {
					item.Fields[header] = int64(num)
				} else {
					item.Fields[header] = num
				}
			} else {
				item.Fields[header] = value
			}
		}
	}

	// Validate required fields
	if item.Title == "" {
		return item, fmt.Errorf("item must have a 'Title' field")
	}

	return item, nil
}

// ValidateImportItems performs basic validation on import items
func ValidateImportItems(items []ImportItem) error {
	if len(items) == 0 {
		return fmt.Errorf("no items found to import")
	}

	for i, item := range items {
		if err := ValidateImportItem(item); err != nil {
			return fmt.Errorf("validation failed for item %d: %w", i+1, err)
		}
	}

	return nil
}

// ValidateImportItem validates a single import item
func ValidateImportItem(item ImportItem) error {
	if item.Title == "" {
		return fmt.Errorf("item must have a title")
	}

	// If URL is provided, it should be a valid GitHub URL
	if item.URL != "" {
		if !strings.Contains(item.URL, "github.com") {
			return fmt.Errorf("URL must be a GitHub URL: %s", item.URL)
		}
	}

	return nil
}

// GetItemType determines the type of item (Issue, PullRequest, or DraftIssue)
func GetItemType(item ImportItem) string {
	if item.Content.Type != "" {
		return item.Content.Type
	}

	if item.URL != "" {
		if strings.Contains(item.URL, "/pull/") {
			return "PullRequest"
		} else if strings.Contains(item.URL, "/issues/") {
			return "Issue"
		}
	}

	// Default to DraftIssue if no URL
	return "DraftIssue"
}

// GetItemBody returns the body text for an item
func GetItemBody(item ImportItem) string {
	if item.Content.Body != "" {
		return item.Content.Body
	}
	if item.Notes != "" {
		return item.Notes
	}
	return ""
}