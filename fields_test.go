// Tests for field conversion and validation logic
// Tests the various field type conversions used in the import process
package main

import (
	"testing"
)

func TestIterationFieldConversion(t *testing.T) {
	tests := []struct {
		name     string
		field    ProjectField
		value    interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "text field",
			field: ProjectField{
				Type: "TEXT",
			},
			value:    "test value",
			expected: map[string]interface{}{"text": "test value"},
			wantErr:  false,
		},
		{
			name: "number field with integer",
			field: ProjectField{
				Type: "NUMBER",
			},
			value:    42,
			expected: map[string]interface{}{"number": float64(42)},
			wantErr:  false,
		},
		{
			name: "single select field with valid option",
			field: ProjectField{
				Type: "SINGLE_SELECT",
				Options: []ProjectFieldOption{
					{ID: "opt1", Name: "Option 1"},
					{ID: "opt2", Name: "Option 2"},
				},
			},
			value:    "Option 1",
			expected: map[string]interface{}{"singleSelectOptionId": "opt1"},
			wantErr:  false,
		},
		{
			name: "single select field with invalid option",
			field: ProjectField{
				Type: "SINGLE_SELECT",
				Options: []ProjectFieldOption{
					{ID: "opt1", Name: "Option 1"},
				},
			},
			value:   "Invalid Option",
			wantErr: true,
		},
		{
			name: "iteration field with valid iteration",
			field: ProjectField{
				Type: "ITERATION",
				Iterations: []IterationOption{
					{ID: "iter1", Title: "Sprint 1"},
					{ID: "iter2", Title: "Sprint 2"},
				},
			},
			value:    "Sprint 1",
			expected: map[string]interface{}{"iterationId": "iter1"},
			wantErr:  false,
		},
		{
			name: "iteration field with invalid iteration",
			field: ProjectField{
				Type: "ITERATION",
				Iterations: []IterationOption{
					{ID: "iter1", Title: "Sprint 1"},
				},
			},
			value:   "Sprint 99",
			wantErr: true,
		},
		{
			name: "date field with ISO format",
			field: ProjectField{
				Type: "DATE",
			},
			value:    "2024-01-15",
			expected: map[string]interface{}{"date": "2024-01-15T00:00:00Z"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertFieldValue(tt.value, tt.field)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			// Compare the result with expected
			if !deepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// deepEqual is a simple deep equal comparison for our use case
func deepEqual(a, b interface{}) bool {
	mapA, okA := a.(map[string]interface{})
	mapB, okB := b.(map[string]interface{})
	
	if !okA || !okB {
		return a == b
	}
	
	if len(mapA) != len(mapB) {
		return false
	}
	
	for key, valueA := range mapA {
		valueB, exists := mapB[key]
		if !exists || valueA != valueB {
			return false
		}
	}
	
	return true
}