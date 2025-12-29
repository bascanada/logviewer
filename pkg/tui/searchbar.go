// SPDX-License-Identifier: GPL-3.0-only
package tui

import (
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/operator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchBarStyles defines the styles for the search bar
type SearchBarStyles struct {
	Container        lipgloss.Style
	Prompt           lipgloss.Style
	ChipField        lipgloss.Style
	ChipVariable     lipgloss.Style
	ChipFreeText     lipgloss.Style
	ChipTimeRange    lipgloss.Style
	ChipVarAssign    lipgloss.Style
	ChipNativeQuery  lipgloss.Style
	ChipFilterGroup  lipgloss.Style
	ChipSelected     lipgloss.Style
	InputActive      lipgloss.Style
	InputInactive    lipgloss.Style
	Autocomplete     lipgloss.Style
	SuggestionItem   lipgloss.Style
	SuggestionActive lipgloss.Style
}

// DefaultSearchBarStyles returns the default styles for the search bar
func DefaultSearchBarStyles() SearchBarStyles {
	return SearchBarStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1),
		Prompt: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),
		ChipField: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1),
		ChipVariable: lipgloss.NewStyle().
			Background(ColorSuccess).
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1),
		ChipFreeText: lipgloss.NewStyle().
			Background(ColorMuted).
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1),
		ChipTimeRange: lipgloss.NewStyle().
			Background(lipgloss.Color("208")). // Orange
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1),
		ChipVarAssign: lipgloss.NewStyle().
			Background(lipgloss.Color("141")). // Purple
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1),
		ChipNativeQuery: lipgloss.NewStyle().
			Background(lipgloss.Color("166")). // Orange-brown
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1),
		ChipFilterGroup: lipgloss.NewStyle().
			Background(lipgloss.Color("99")). // Light purple
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1).
			Italic(true), // Indicates grouped/complex
		ChipSelected: lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorBg).
			Padding(0, 1).
			MarginRight(1).
			Bold(true),
		InputActive: lipgloss.NewStyle().
			Foreground(ColorText),
		InputInactive: lipgloss.NewStyle().
			Foreground(ColorMuted),
		Autocomplete: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			MarginTop(1),
		SuggestionItem: lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1),
		SuggestionActive: lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorBg).
			Padding(0, 1),
	}
}

// SearchBar is the chip-based search input component
type SearchBar struct {
	State     ChipSearchState
	TextInput textinput.Model
	Styles    SearchBarStyles
	Width     int
	Focused   bool

	// Data sources for autocomplete
	AvailableFields    []string            // Fields discovered from loaded entries
	AvailableVariables []string            // Variables from config
	VariableMetadata   map[string]string   // Variable name -> description
	FieldValues        map[string][]string // Field -> possible values (cached)
}

// NewSearchBar creates a new search bar with default settings
func NewSearchBar() SearchBar {
	ti := textinput.New()
	ti.Placeholder = "type to search, Tab for autocomplete..."
	ti.CharLimit = 256

	return SearchBar{
		State:              NewChipSearchState(),
		TextInput:          ti,
		Styles:             DefaultSearchBarStyles(),
		Width:              80,
		Focused:            false,
		AvailableFields:    make([]string, 0),
		AvailableVariables: make([]string, 0),
		VariableMetadata:   make(map[string]string),
		FieldValues:        make(map[string][]string),
	}
}

// Focus activates the search bar
func (s *SearchBar) Focus() tea.Cmd {
	s.Focused = true
	s.TextInput.Focus()
	return textinput.Blink
}

// Blur deactivates the search bar
func (s *SearchBar) Blur() {
	s.Focused = false
	s.TextInput.Blur()
	s.State.AutocompleteOpen = false
}

// Update handles messages for the search bar
func (s SearchBar) Update(msg tea.Msg) (SearchBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return s.handleKey(msg)
	}

	// Pass through to text input
	var cmd tea.Cmd
	s.TextInput, cmd = s.TextInput.Update(msg)
	s.State.CurrentInput = s.TextInput.Value()
	return s, cmd
}

// handleKey processes keyboard input
func (s SearchBar) handleKey(msg tea.KeyMsg) (SearchBar, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Toggle/cycle autocomplete
		if !s.State.AutocompleteOpen {
			s.State.AutocompleteOpen = true
			s.State.AutocompleteSuggestions = s.generateSuggestions()
			s.State.AutocompleteIndex = 0
		} else if len(s.State.AutocompleteSuggestions) > 0 {
			// Cycle to next suggestion
			s.State.AutocompleteIndex = (s.State.AutocompleteIndex + 1) % len(s.State.AutocompleteSuggestions)
		}
		return s, nil

	case tea.KeyShiftTab:
		// Cycle backwards through autocomplete
		if s.State.AutocompleteOpen && len(s.State.AutocompleteSuggestions) > 0 {
			s.State.AutocompleteIndex = (s.State.AutocompleteIndex - 1 + len(s.State.AutocompleteSuggestions)) % len(s.State.AutocompleteSuggestions)
		}
		return s, nil

	case tea.KeyUp:
		if s.State.AutocompleteOpen && len(s.State.AutocompleteSuggestions) > 0 {
			s.State.AutocompleteIndex = (s.State.AutocompleteIndex - 1 + len(s.State.AutocompleteSuggestions)) % len(s.State.AutocompleteSuggestions)
			return s, nil
		}

	case tea.KeyDown:
		if s.State.AutocompleteOpen && len(s.State.AutocompleteSuggestions) > 0 {
			s.State.AutocompleteIndex = (s.State.AutocompleteIndex + 1) % len(s.State.AutocompleteSuggestions)
			return s, nil
		}

	case tea.KeyEnter:
		if s.State.AutocompleteOpen && len(s.State.AutocompleteSuggestions) > 0 {
			// Accept suggestion
			suggestion := s.State.AutocompleteSuggestions[s.State.AutocompleteIndex]
			s.acceptSuggestion(suggestion)
			s.State.AutocompleteOpen = false
			return s, nil
		}

		// Commit current input as chip
		if s.State.CurrentInput != "" {
			s.commitCurrentInput()
		}
		return s, nil

	case tea.KeyBackspace:
		if s.State.CurrentInput == "" && len(s.State.Chips) > 0 {
			// Remove last chip when input is empty
			s.State.RemoveLastChip()
			return s, nil
		}

	case tea.KeyEscape:
		if s.State.AutocompleteOpen {
			s.State.AutocompleteOpen = false
			return s, nil
		}
		// Let parent handle escape to exit search mode

	case tea.KeyLeft:
		if s.State.CurrentInput == "" && len(s.State.Chips) > 0 {
			// Navigate to chips
			if s.State.SelectedChip == -1 {
				s.State.SelectedChip = len(s.State.Chips) - 1
			} else if s.State.SelectedChip > 0 {
				s.State.SelectedChip--
			}
			return s, nil
		}

	case tea.KeyRight:
		if s.State.SelectedChip >= 0 {
			if s.State.SelectedChip < len(s.State.Chips)-1 {
				s.State.SelectedChip++
			} else {
				s.State.SelectedChip = -1 // Back to input
			}
			return s, nil
		}

	case tea.KeyDelete:
		if s.State.SelectedChip >= 0 {
			s.State.RemoveChip(s.State.SelectedChip)
			if s.State.SelectedChip >= len(s.State.Chips) {
				s.State.SelectedChip = len(s.State.Chips) - 1
			}
			return s, nil
		}
	}

	// Default: update text input
	var cmd tea.Cmd
	s.TextInput, cmd = s.TextInput.Update(msg)
	s.State.CurrentInput = s.TextInput.Value()

	// Close autocomplete when typing
	if msg.Type == tea.KeyRunes {
		s.State.AutocompleteOpen = false
	}

	return s, cmd
}

// View renders the search bar
func (s SearchBar) View() string {
	var parts []string

	// Prompt
	parts = append(parts, s.Styles.Prompt.Render("/"))

	// Render chips
	for i, chip := range s.State.Chips {
		style := s.getChipStyle(chip.Type)
		if i == s.State.SelectedChip {
			style = s.Styles.ChipSelected
		}
		parts = append(parts, style.Render(chip.Display))
	}

	// Input field
	if s.Focused {
		parts = append(parts, s.Styles.InputActive.Render(s.TextInput.View()))
	} else if s.State.CurrentInput != "" {
		parts = append(parts, s.Styles.InputInactive.Render(s.State.CurrentInput))
	} else if len(s.State.Chips) == 0 {
		parts = append(parts, s.Styles.InputInactive.Render("Press / to search..."))
	}

	searchLine := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

	// Add autocomplete dropdown if open
	if s.State.AutocompleteOpen && len(s.State.AutocompleteSuggestions) > 0 {
		autocompleteView := s.renderAutocomplete()
		return lipgloss.JoinVertical(lipgloss.Left, searchLine, autocompleteView)
	}

	return searchLine
}

// renderAutocomplete renders the autocomplete dropdown
func (s SearchBar) renderAutocomplete() string {
	var items []string

	maxItems := 8 // Limit visible suggestions
	count := len(s.State.AutocompleteSuggestions)
	if count > maxItems {
		count = maxItems
	}

	for i := 0; i < count; i++ {
		suggestion := s.State.AutocompleteSuggestions[i]
		style := s.Styles.SuggestionItem
		if i == s.State.AutocompleteIndex {
			style = s.Styles.SuggestionActive
		}

		text := suggestion.Text
		if suggestion.Description != "" {
			text += " - " + suggestion.Description
		}
		items = append(items, style.Render(text))
	}

	if len(s.State.AutocompleteSuggestions) > maxItems {
		items = append(items, s.Styles.SuggestionItem.Foreground(ColorMuted).Render(
			"... and more"))
	}

	return s.Styles.Autocomplete.Render(
		lipgloss.JoinVertical(lipgloss.Left, items...),
	)
}

// getChipStyle returns the appropriate style for a chip type
func (s SearchBar) getChipStyle(chipType ChipType) lipgloss.Style {
	switch chipType {
	case ChipTypeVariable:
		return s.Styles.ChipVariable
	case ChipTypeFreeText:
		return s.Styles.ChipFreeText
	case ChipTypeTimeRange:
		return s.Styles.ChipTimeRange
	case ChipTypeVarAssign:
		return s.Styles.ChipVarAssign
	case ChipTypeNativeQuery:
		return s.Styles.ChipNativeQuery
	case ChipTypeFilterGroup:
		return s.Styles.ChipFilterGroup
	default:
		return s.Styles.ChipField
	}
}

// generateSuggestions creates autocomplete suggestions based on current input
func (s *SearchBar) generateSuggestions() []Suggestion {
	input := strings.TrimSpace(s.State.CurrentInput)

	// Native query: no suggestions once typing the query
	if strings.HasPrefix(input, "query:") {
		return nil // Let user type their native query freely
	}

	// Time range suggestions when prefix is typed
	if strings.HasPrefix(input, "last:") || strings.HasPrefix(input, "from:") || strings.HasPrefix(input, "to:") {
		return s.suggestTimeValues(input)
	}

	// Suggest native query prefix when typing 'q'
	if input == "q" || input == "qu" || input == "que" || input == "quer" || input == "query" {
		return []Suggestion{
			{Text: "query:", Description: "native query (SPL, Lucene, etc.)", Context: AutocompleteContextField},
		}
	}

	// Suggest time range prefixes when typing 'l', 'f', 't'
	if input == "l" || input == "la" || input == "las" || input == "last" {
		return []Suggestion{
			{Text: "last:", Description: "relative time (e.g., 1h, 24h, 7d)", Context: AutocompleteContextField},
		}
	}
	if input == "f" || input == "fr" || input == "fro" || input == "from" {
		return []Suggestion{
			{Text: "from:", Description: "start time (absolute or relative)", Context: AutocompleteContextField},
		}
	}
	if input == "to" {
		return []Suggestion{
			{Text: "to:", Description: "end time (absolute or relative)", Context: AutocompleteContextField},
		}
	}

	// Variable assignment: $varName= suggests variable names for assignment
	if strings.HasPrefix(input, "$") && strings.Contains(input, "=") {
		// Already has assignment, no suggestions needed
		return nil
	}

	// Variable suggestions if starting with $
	if strings.HasPrefix(input, "$") {
		return s.suggestVariables(strings.TrimPrefix(input, "$"))
	}

	// Check for operator - suggest values
	if idx := strings.IndexAny(input, "=!~<>"); idx != -1 {
		field := strings.TrimSpace(input[:idx])
		return s.suggestValues(field)
	}

	// Check if input contains a partial field name
	if input != "" {
		// Suggest matching fields
		return s.suggestFields(input)
	}

	// Default: show time range options, native query, and fields
	suggestions := []Suggestion{
		{Text: "last:", Description: "relative time (e.g., 1h, 24h)", Context: AutocompleteContextField},
		{Text: "from:", Description: "start time", Context: AutocompleteContextField},
		{Text: "to:", Description: "end time", Context: AutocompleteContextField},
		{Text: "query:", Description: "native query (SPL, Lucene)", Context: AutocompleteContextField},
	}
	fieldSuggestions := s.suggestFields("")
	if len(fieldSuggestions) > 2 {
		fieldSuggestions = fieldSuggestions[:2]
	}
	suggestions = append(suggestions, fieldSuggestions...)
	return suggestions
}

// suggestFields suggests field names matching the prefix
func (s *SearchBar) suggestFields(prefix string) []Suggestion {
	var suggestions []Suggestion
	prefix = strings.ToLower(prefix)

	for _, field := range s.AvailableFields {
		if prefix == "" || strings.Contains(strings.ToLower(field), prefix) {
			suggestions = append(suggestions, Suggestion{
				Text:        field,
				Description: "field",
				Context:     AutocompleteContextField,
			})
		}
	}

	// Sort by relevance (prefix match first)
	sort.Slice(suggestions, func(i, j int) bool {
		iMatch := strings.HasPrefix(strings.ToLower(suggestions[i].Text), prefix)
		jMatch := strings.HasPrefix(strings.ToLower(suggestions[j].Text), prefix)
		if iMatch != jMatch {
			return iMatch
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

// suggestOperators suggests available operators
func (s *SearchBar) suggestOperators() []Suggestion {
	return []Suggestion{
		{Text: "=", Description: "equals", Context: AutocompleteContextOperator},
		{Text: "!=", Description: "not equals", Context: AutocompleteContextOperator},
		{Text: "~=", Description: "regex match", Context: AutocompleteContextOperator},
		{Text: ">", Description: "greater than", Context: AutocompleteContextOperator},
		{Text: ">=", Description: "greater or equal", Context: AutocompleteContextOperator},
		{Text: "<", Description: "less than", Context: AutocompleteContextOperator},
		{Text: "<=", Description: "less or equal", Context: AutocompleteContextOperator},
	}
}

// suggestValues suggests values for a field
func (s *SearchBar) suggestValues(field string) []Suggestion {
	var suggestions []Suggestion

	if values, ok := s.FieldValues[field]; ok {
		for _, val := range values {
			suggestions = append(suggestions, Suggestion{
				Text:        val,
				Description: "",
				Context:     AutocompleteContextValue,
			})
		}
	}

	return suggestions
}

// suggestVariables suggests variables matching the prefix
func (s *SearchBar) suggestVariables(prefix string) []Suggestion {
	var suggestions []Suggestion
	prefix = strings.ToLower(prefix)

	for _, varName := range s.AvailableVariables {
		if prefix == "" || strings.Contains(strings.ToLower(varName), prefix) {
			desc := s.VariableMetadata[varName]
			suggestions = append(suggestions, Suggestion{
				Text:        "$" + varName,
				Description: desc,
				Context:     AutocompleteContextVariable,
			})
		}
	}

	return suggestions
}

// suggestTimeValues suggests time presets for time range chips
func (s *SearchBar) suggestTimeValues(input string) []Suggestion {
	var suggestions []Suggestion

	// Determine the prefix (last:, from:, to:)
	var prefix string
	var currentValue string
	if strings.HasPrefix(input, "last:") {
		prefix = "last:"
		currentValue = strings.TrimPrefix(input, "last:")
	} else if strings.HasPrefix(input, "from:") {
		prefix = "from:"
		currentValue = strings.TrimPrefix(input, "from:")
	} else if strings.HasPrefix(input, "to:") {
		prefix = "to:"
		currentValue = strings.TrimPrefix(input, "to:")
	}

	// Time presets
	var presets []string
	if prefix == "last:" {
		presets = []string{"15m", "30m", "1h", "6h", "12h", "24h", "7d", "30d"}
	} else {
		presets = []string{"now", "now-1h", "now-6h", "now-24h", "now-7d", "now-30d"}
	}

	// Filter presets by current value
	currentValue = strings.ToLower(currentValue)
	for _, preset := range presets {
		if currentValue == "" || strings.Contains(strings.ToLower(preset), currentValue) {
			suggestions = append(suggestions, Suggestion{
				Text:        prefix + preset,
				Description: "",
				Context:     AutocompleteContextValue,
			})
		}
	}

	return suggestions
}

// acceptSuggestion applies a selected suggestion to the input
func (s *SearchBar) acceptSuggestion(suggestion Suggestion) {
	switch suggestion.Context {
	case AutocompleteContextField:
		// Check if it's a time range prefix (last:, from:, to:)
		if strings.HasSuffix(suggestion.Text, ":") {
			s.TextInput.SetValue(suggestion.Text)
			s.State.CurrentInput = s.TextInput.Value()
		} else {
			// Insert field name and wait for operator
			s.TextInput.SetValue(suggestion.Text + "=")
			s.State.CurrentInput = s.TextInput.Value()
		}

	case AutocompleteContextOperator:
		// Append operator to current input
		current := s.State.CurrentInput
		s.TextInput.SetValue(current + suggestion.Text)
		s.State.CurrentInput = s.TextInput.Value()

	case AutocompleteContextValue:
		// Check if it's a complete time range (last:1h, from:now, etc.)
		if strings.HasPrefix(suggestion.Text, "last:") ||
			strings.HasPrefix(suggestion.Text, "from:") ||
			strings.HasPrefix(suggestion.Text, "to:") {
			// Time range: set full value and commit
			s.TextInput.SetValue(suggestion.Text)
			s.State.CurrentInput = s.TextInput.Value()
			s.commitCurrentInput()
		} else {
			// Regular value: append to current input and commit
			s.TextInput.SetValue(s.State.CurrentInput + suggestion.Text)
			s.State.CurrentInput = s.TextInput.Value()
			s.commitCurrentInput()
		}

	case AutocompleteContextVariable:
		// Create variable chip
		chip := Chip{
			Type:    ChipTypeVariable,
			Value:   strings.TrimPrefix(suggestion.Text, "$"),
			Display: suggestion.Text,
		}
		s.State.AddChip(chip)
		s.TextInput.SetValue("")
	}
}

// commitCurrentInput parses the current input and creates a chip
func (s *SearchBar) commitCurrentInput() {
	input := strings.TrimSpace(s.State.CurrentInput)
	if input == "" {
		return
	}

	chip := s.parseInput(input)
	s.State.AddChip(chip)
	s.TextInput.SetValue("")
	s.State.CurrentInput = ""
}

// parseInput parses input text into a Chip
func (s *SearchBar) parseInput(input string) Chip {
	// Native query: query:index=main sourcetype=json
	if strings.HasPrefix(input, "query:") {
		value := strings.TrimPrefix(input, "query:")
		return Chip{
			Type:     ChipTypeNativeQuery,
			Value:    value,
			Display:  input,
			Editable: true,
		}
	}

	// Time range: last:1h, from:2024-01-01, to:now
	if strings.HasPrefix(input, "last:") {
		return Chip{
			Type:    ChipTypeTimeRange,
			Field:   "last",
			Value:   strings.TrimPrefix(input, "last:"),
			Display: input,
		}
	}
	if strings.HasPrefix(input, "from:") {
		return Chip{
			Type:    ChipTypeTimeRange,
			Field:   "from",
			Value:   strings.TrimPrefix(input, "from:"),
			Display: input,
		}
	}
	if strings.HasPrefix(input, "to:") {
		return Chip{
			Type:    ChipTypeTimeRange,
			Field:   "to",
			Value:   strings.TrimPrefix(input, "to:"),
			Display: input,
		}
	}

	// Variable assignment: $varName=value
	if strings.HasPrefix(input, "$") && strings.Contains(input, "=") {
		parts := strings.SplitN(strings.TrimPrefix(input, "$"), "=", 2)
		if len(parts) == 2 {
			return Chip{
				Type:    ChipTypeVarAssign,
				Field:   parts[0],
				Value:   parts[1],
				Display: input,
			}
		}
	}

	// Variable reference: $varName
	if strings.HasPrefix(input, "$") {
		return Chip{
			Type:    ChipTypeVariable,
			Value:   strings.TrimPrefix(input, "$"),
			Display: input,
		}
	}

	// Field with operator: field=value, field!=value, field~=value, etc.
	// Pattern: field{op}value where op is =, !=, ~=, >, >=, <, <=
	opPattern := regexp.MustCompile(`^([a-zA-Z0-9_.-]+)(!=|~=|>=|<=|=|>|<)(.*)$`)
	if matches := opPattern.FindStringSubmatch(input); len(matches) == 4 {
		field := matches[1]
		op := matches[2]
		value := matches[3]

		return Chip{
			Type:     ChipTypeField,
			Field:    field,
			Operator: op,
			Value:    value,
			Display:  input,
		}
	}

	// Free text search
	return Chip{
		Type:    ChipTypeFreeText,
		Text:    input,
		Display: input,
	}
}

// BuildFilter converts chips to a client.Filter
func (s *SearchBar) BuildFilter() *client.Filter {
	if len(s.State.Chips) == 0 {
		return nil
	}

	var filters []client.Filter

	for _, chip := range s.State.Chips {
		switch chip.Type {
		case ChipTypeField:
			op, negate := mapOperatorToClient(chip.Operator)
			filters = append(filters, client.Filter{
				Field:  chip.Field,
				Op:     op,
				Value:  chip.Value,
				Negate: negate,
			})

		case ChipTypeVariable:
			// Variables are resolved at search time
			// For now, treat as a field reference
			filters = append(filters, client.Filter{
				Field: chip.Value,
				Op:    operator.Exists,
			})

		case ChipTypeFreeText:
			// Free text searches the message field
			filters = append(filters, client.Filter{
				Field: "_",
				Op:    operator.Match,
				Value: chip.Text,
			})

		case ChipTypeFilterGroup:
			// Preserve the original filter structure for complex groups
			if chip.GroupFilter != nil {
				filters = append(filters, *chip.GroupFilter)
			}
		}
	}

	if len(filters) == 0 {
		return nil
	}

	if len(filters) == 1 {
		return &filters[0]
	}

	return &client.Filter{
		Logic:   client.LogicAnd,
		Filters: filters,
	}
}

// mapOperatorToClient converts UI operator to client operator
func mapOperatorToClient(op string) (string, bool) {
	switch op {
	case "=", "==":
		return operator.Equals, false
	case "!=":
		return operator.Equals, true // negate
	case "~=", "=~":
		return operator.Regex, false
	case "!~=", "!~":
		return operator.Regex, true // negate
	case ">":
		return operator.Gt, false
	case ">=":
		return operator.Gte, false
	case "<":
		return operator.Lt, false
	case "<=":
		return operator.Lte, false
	default:
		return operator.Match, false
	}
}

// UpdateAvailableFields refreshes field suggestions from entries
func (s *SearchBar) UpdateAvailableFields(entries []client.LogEntry) {
	fieldSet := make(map[string]bool)

	for _, entry := range entries {
		for field := range entry.Fields {
			fieldSet[field] = true
		}
		// Also include standard fields
		if entry.Level != "" {
			fieldSet["level"] = true
		}
		if entry.ContextID != "" {
			fieldSet["context"] = true
		}
	}

	s.AvailableFields = make([]string, 0, len(fieldSet))
	for field := range fieldSet {
		s.AvailableFields = append(s.AvailableFields, field)
	}
	sort.Strings(s.AvailableFields)
}

// UpdateAvailableVariables refreshes variable suggestions from config
func (s *SearchBar) UpdateAvailableVariables(vars map[string]client.VariableDefinition) {
	s.AvailableVariables = make([]string, 0, len(vars))
	s.VariableMetadata = make(map[string]string, len(vars))

	for name, def := range vars {
		s.AvailableVariables = append(s.AvailableVariables, name)
		s.VariableMetadata[name] = def.Description
	}
	sort.Strings(s.AvailableVariables)
}

// Clear clears all chips and input
func (s *SearchBar) Clear() {
	s.State.Clear()
	s.TextInput.SetValue("")
}

// HasFilter returns true if there are any active filters
func (s *SearchBar) HasFilter() bool {
	return len(s.State.Chips) > 0 || s.State.CurrentInput != ""
}

// GetFreeTextSearch returns the current free text to search (for simple filtering)
func (s *SearchBar) GetFreeTextSearch() string {
	// Collect all free text from chips and current input
	var texts []string
	for _, chip := range s.State.Chips {
		if chip.Type == ChipTypeFreeText {
			texts = append(texts, chip.Text)
		}
	}
	if s.State.CurrentInput != "" {
		texts = append(texts, s.State.CurrentInput)
	}
	return strings.Join(texts, " ")
}

// BuildSearchModifiers extracts time range and variable assignments from chips
func (s *SearchBar) BuildSearchModifiers() (timeRange *client.SearchRange, vars map[string]string) {
	vars = make(map[string]string)
	timeRange = &client.SearchRange{}

	for _, chip := range s.State.Chips {
		switch chip.Type {
		case ChipTypeTimeRange:
			switch chip.Field {
			case "last":
				timeRange.Last.S(chip.Value)
			case "from":
				timeRange.Gte.S(chip.Value)
			case "to":
				timeRange.Lte.S(chip.Value)
			}
		case ChipTypeVarAssign:
			vars[chip.Field] = chip.Value
		}
	}

	// Return nil if no time range set
	if !timeRange.Last.Set && !timeRange.Gte.Set && !timeRange.Lte.Set {
		timeRange = nil
	}

	return timeRange, vars
}

// mapClientOperatorToUI converts a client operator to a UI operator string
func mapClientOperatorToUI(op string, negate bool) string {
	switch op {
	case operator.Equals, "":
		if negate {
			return "!="
		}
		return "="
	case operator.Regex:
		if negate {
			return "!~="
		}
		return "~="
	case operator.Match:
		if negate {
			return "!~="
		}
		return "~="
	case operator.Gt:
		return ">"
	case operator.Gte:
		return ">="
	case operator.Lt:
		return "<"
	case operator.Lte:
		return "<="
	case operator.Exists:
		return " exists"
	case operator.Wildcard:
		if negate {
			return "!*="
		}
		return "*="
	default:
		if negate {
			return "!="
		}
		return "="
	}
}

// filterToChips recursively converts a Filter AST to chips
func filterToChips(filter *client.Filter) []Chip {
	return filterToChipsWithDepth(filter, 0)
}

const maxFilterDepth = 3

// filterToChipsWithDepth converts a Filter AST to chips with depth tracking
func filterToChipsWithDepth(filter *client.Filter, depth int) []Chip {
	if filter == nil {
		return nil
	}

	// Prevent infinite recursion and collapse deeply nested filters
	if depth > maxFilterDepth {
		return []Chip{{
			Type:        ChipTypeFilterGroup,
			Display:     "[Complex filter]",
			GroupFilter: filter,
			Editable:    false,
		}}
	}

	// Case 1: Leaf node (simple condition with Field set)
	if filter.Field != "" {
		return []Chip{leafFilterToChip(filter)}
	}

	// Case 2: Branch node (group with Logic set)
	if filter.Logic != "" {
		return groupFilterToChips(filter, depth)
	}

	return nil
}

// leafFilterToChip converts a simple field condition to a chip
func leafFilterToChip(filter *client.Filter) Chip {
	op := mapClientOperatorToUI(filter.Op, filter.Negate)
	display := filter.Field + op + filter.Value

	return Chip{
		Type:     ChipTypeField,
		Field:    filter.Field,
		Operator: op,
		Value:    filter.Value,
		Display:  display,
		Editable: true,
	}
}

// groupFilterToChips converts an AND/OR/NOT group to chips
func groupFilterToChips(filter *client.Filter, depth int) []Chip {
	switch filter.Logic {
	case client.LogicAnd:
		// AND at root level: flatten to separate chips (implicit AND between chips)
		var chips []Chip
		for i := range filter.Filters {
			chips = append(chips, filterToChipsWithDepth(&filter.Filters[i], depth+1)...)
		}
		return chips

	case client.LogicOr:
		// OR group: create a single grouped chip
		return []Chip{createGroupChip(filter)}

	case client.LogicNot:
		// NOT group: create a negated chip or group
		if len(filter.Filters) == 1 && filter.Filters[0].Field != "" {
			// Single condition negation: use negated operator
			negatedFilter := filter.Filters[0]
			negatedFilter.Negate = !negatedFilter.Negate
			return []Chip{leafFilterToChip(&negatedFilter)}
		}
		// Complex NOT: create group chip
		return []Chip{createGroupChip(filter)}
	}

	return nil
}

// createGroupChip creates a ChipTypeFilterGroup for OR/complex groups
func createGroupChip(filter *client.Filter) Chip {
	display := formatFilterForDisplay(filter)

	return Chip{
		Type:        ChipTypeFilterGroup,
		Display:     display,
		GroupLogic:  string(filter.Logic),
		GroupFilter: filter,
		Editable:    false, // Complex groups are read-only in chip form
	}
}

// formatFilterForDisplay creates a human-readable string for complex filters
func formatFilterForDisplay(filter *client.Filter) string {
	if filter == nil {
		return ""
	}

	// Leaf node
	if filter.Field != "" {
		op := mapClientOperatorToUI(filter.Op, filter.Negate)
		return filter.Field + op + filter.Value
	}

	// Branch node
	if filter.Logic == "" {
		return ""
	}

	var parts []string
	for i := range filter.Filters {
		part := formatFilterForDisplay(&filter.Filters[i])
		if part != "" {
			parts = append(parts, part)
		}
	}

	switch filter.Logic {
	case client.LogicOr:
		return "(" + strings.Join(parts, " OR ") + ")"
	case client.LogicAnd:
		return "(" + strings.Join(parts, " AND ") + ")"
	case client.LogicNot:
		if len(parts) == 1 {
			return "NOT " + parts[0]
		}
		return "NOT(" + strings.Join(parts, " AND ") + ")"
	}

	return strings.Join(parts, " ")
}

// truncateForDisplay truncates a string for chip display
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// PopulateFromSearch converts a LogSearch into chips for the search bar
// This is used to auto-fill the search bar from CLI arguments or context config
func (s *SearchBar) PopulateFromSearch(search *client.LogSearch) {
	if search == nil {
		return
	}

	// Add native query chip (if present)
	if search.NativeQuery.Set && search.NativeQuery.Value != "" {
		displayValue := search.NativeQuery.Value
		s.State.Chips = append(s.State.Chips, Chip{
			Type:     ChipTypeNativeQuery,
			Value:    search.NativeQuery.Value,
			Display:  "query:" + truncateForDisplay(displayValue, 40),
			Editable: true,
		})
	}

	// Add time range chips
	if search.Range.Last.Set && search.Range.Last.Value != "" {
		s.State.Chips = append(s.State.Chips, Chip{
			Type:    ChipTypeTimeRange,
			Field:   "last",
			Value:   search.Range.Last.Value,
			Display: "last:" + search.Range.Last.Value,
		})
	}
	if search.Range.Gte.Set && search.Range.Gte.Value != "" {
		s.State.Chips = append(s.State.Chips, Chip{
			Type:    ChipTypeTimeRange,
			Field:   "from",
			Value:   search.Range.Gte.Value,
			Display: "from:" + search.Range.Gte.Value,
		})
	}
	if search.Range.Lte.Set && search.Range.Lte.Value != "" {
		s.State.Chips = append(s.State.Chips, Chip{
			Type:    ChipTypeTimeRange,
			Field:   "to",
			Value:   search.Range.Lte.Value,
			Display: "to:" + search.Range.Lte.Value,
		})
	}

	// Add field filter chips (legacy Fields map)
	for field, value := range search.Fields {
		op := "="
		if condition, ok := search.FieldsCondition[field]; ok && condition != "" {
			op = mapClientOperatorToUI(condition, false)
		}
		s.State.Chips = append(s.State.Chips, Chip{
			Type:     ChipTypeField,
			Field:    field,
			Operator: op,
			Value:    value,
			Display:  field + op + value,
			Editable: true,
		})
	}

	// Add Filter AST chips
	if search.Filter != nil {
		filterChips := filterToChips(search.Filter)
		s.State.Chips = append(s.State.Chips, filterChips...)
	}
}

// mapUIOperatorToClient converts a UI operator to a client operator and negate flag
func mapUIOperatorToClient(uiOp string) (string, bool) {
	switch uiOp {
	case "=":
		return operator.Equals, false
	case "!=":
		return operator.Equals, true
	case "~=":
		return operator.Match, false
	case "!~=":
		return operator.Match, true
	case "*=":
		return operator.Wildcard, false
	case "!*=":
		return operator.Wildcard, true
	case ">":
		return operator.Gt, false
	case ">=":
		return operator.Gte, false
	case "<":
		return operator.Lt, false
	case "<=":
		return operator.Lte, false
	case " exists":
		return operator.Exists, false
	default:
		return operator.Equals, false
	}
}

// BuildSearchFromChips creates a LogSearch from the current chips
// This replaces the search fields/time range entirely based on chips
func (s *SearchBar) BuildSearchFromChips() *client.LogSearch {
	search := &client.LogSearch{
		Fields:          make(map[string]string),
		FieldsCondition: make(map[string]string),
	}

	log.Printf("[DEBUG] BuildSearchFromChips: chips count=%d", len(s.State.Chips))
	for i, c := range s.State.Chips {
		log.Printf("[DEBUG] BuildSearchFromChips: chip[%d] type=%d field=%s value=%s", i, c.Type, c.Field, c.Value)
	}

	var filterChips []client.Filter

	for _, chip := range s.State.Chips {
		switch chip.Type {
		case ChipTypeTimeRange:
			switch chip.Field {
			case "last":
				search.Range.Last.S(chip.Value)
			case "from":
				search.Range.Gte.S(chip.Value)
			case "to":
				search.Range.Lte.S(chip.Value)
			}

		case ChipTypeNativeQuery:
			search.NativeQuery.S(chip.Value)

		case ChipTypeField:
			// Convert to Filter node instead of legacy Fields map
			op, negate := mapUIOperatorToClient(chip.Operator)
			filterChips = append(filterChips, client.Filter{
				Field:  chip.Field,
				Op:     op,
				Value:  chip.Value,
				Negate: negate,
			})

		case ChipTypeFilterGroup:
			// Preserve the original filter structure
			if chip.GroupFilter != nil {
				filterChips = append(filterChips, *chip.GroupFilter)
			}
		}
	}

	// Build filter from chips
	if len(filterChips) > 0 {
		if len(filterChips) == 1 {
			search.Filter = &filterChips[0]
		} else {
			search.Filter = &client.Filter{
				Logic:   client.LogicAnd,
				Filters: filterChips,
			}
		}
	}

	return search
}
