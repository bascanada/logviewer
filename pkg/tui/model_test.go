// SPDX-License-Identifier: GPL-3.0-only
package tui

import (
	"context"
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	tea "github.com/charmbracelet/bubbletea"
)

// MockSearchResult implements client.LogSearchResult
type MockSearchResult struct {
	Search *client.LogSearch
}

func (m *MockSearchResult) GetSearch() *client.LogSearch {
	return m.Search
}

func (m *MockSearchResult) GetEntries(context context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {
	return []client.LogEntry{}, nil, nil
}

func (m *MockSearchResult) GetFields(context context.Context) (ty.UniSet[string], chan ty.UniSet[string], error) {
	return nil, nil, nil
}

func (m *MockSearchResult) GetPaginationInfo() *client.PaginationInfo {
	return nil
}

func (m *MockSearchResult) Err() <-chan error {
	return nil
}

func TestModelUpdate_LogEntryMsg_SanitizesFields(t *testing.T) {
	// Setup
	model := New(nil, nil, nil)
	
	// Create a tab
	contextID := "test-context"
	tab := &Tab{
		ID:          "tab-1",
		ContextID:   contextID,
		SearchState: NewChipSearchState(),
		Loading:     true,
	}
	model.Tabs = append(model.Tabs, tab)
	model.ActiveTab = 0

	// Create search with "json" field (problematic config)
	search := &client.LogSearch{
		Fields: ty.MS{
			"json":   "true",
			"format": "template",
			"level":  "ERROR",
		},
		FieldsCondition: ty.MS{},
	}

	// Create message
	msg := LogEntryMsg{
		TabID:    tab.ID,
		Entries:  []client.LogEntry{},
		Result:   &MockSearchResult{Search: search},
		Fields:   make(ty.UniSet[string]),
	}

	// Act
	model.Update(msg)

	// Assert
	updatedTab := model.Tabs[0]
	chips := updatedTab.SearchState.Chips

	// Check that "level" is present
	foundLevel := false
	for _, chip := range chips {
		if chip.Type == ChipTypeField && chip.Field == "level" && chip.Value == "ERROR" {
			foundLevel = true
			break
		}
	}
	if !foundLevel {
		t.Errorf("Expected 'level' chip to be present, but it was missing")
	}

	// Check that "json" is ABSENT
	foundJson := false
	for _, chip := range chips {
		if chip.Type == ChipTypeField && chip.Field == "json" {
			foundJson = true
			break
		}
	}
	if foundJson {
		t.Errorf("Expected 'json' chip to be filtered out, but it was present")
	}

	// Check that "format" is ABSENT
	foundFormat := false
	for _, chip := range chips {
		if chip.Type == ChipTypeField && chip.Field == "format" {
			foundFormat = true
			break
		}
	}
	if foundFormat {
		t.Errorf("Expected 'format' chip to be filtered out, but it was present")
	}
}

func TestModelUpdate_LogEntryMsg_PreservesOtherFields(t *testing.T) {
	// Setup
	model := New(nil, nil, nil)
	
	// Create a tab
	tab := &Tab{
		ID:          "tab-1",
		ContextID:   "test-context",
		SearchState: NewChipSearchState(),
		Loading:     true,
	}
	model.Tabs = append(model.Tabs, tab)
	model.ActiveTab = 0

	// Create search with normal fields
	search := &client.LogSearch{
		Fields: ty.MS{
			"service": "api",
			"env":     "prod",
		},
		FieldsCondition: ty.MS{},
	}

	// Create message
	msg := LogEntryMsg{
		TabID:    tab.ID,
		Entries:  []client.LogEntry{},
		Result:   &MockSearchResult{Search: search},
		Fields:   make(ty.UniSet[string]),
	}

	// Act
	model.Update(msg)

	// Assert
	updatedTab := model.Tabs[0]
	chips := updatedTab.SearchState.Chips

	if len(chips) != 2 {
		t.Errorf("Expected 2 chips, got %d", len(chips))
	}
}

// Ensure Tea.Msg interface is satisfied (implied, but good practice)
var _ tea.Msg = LogEntryMsg{}
