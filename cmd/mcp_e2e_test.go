package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_ListContexts(t *testing.T) {
	cfg := &config.ContextConfig{Clients: config.Clients{}, Searches: config.Searches{}, Contexts: config.Contexts{}}
	cfg.Clients["dummy"] = config.Client{Type: "local", Options: ty.MI{}}
	cfg.Contexts["alpha"] = config.SearchContext{Client: "dummy", Search: client.LogSearch{}}

	bundle, err := BuildMCPServer(cfg)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	handler := bundle.ToolHandlers["list_contexts"]
	res, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("tool error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("no content")
	}
	textPayload := ""
	if tc, ok := res.Content[0].(mcp.TextContent); ok {
		textPayload = tc.Text
	} else {
		b, err := json.Marshal(res.Content[0])
		if err != nil {
			t.Fatalf("failed to marshal tool content: %v", err)
		}
		textPayload = string(b)
	}
	var list []string
	if err := json.Unmarshal([]byte(textPayload), &list); err != nil {
		t.Fatalf("failed to unmarshal context list: %v raw=%s", err, textPayload)
	}
	found := false
	for _, v := range list {
		if v == "alpha" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'alpha' in context list: %v", list)
	}
}

func TestMCP_GetContextDetails(t *testing.T) {
	cfg := &config.ContextConfig{
		Clients: config.Clients{"dummy": {Type: "local"}},
		Contexts: config.Contexts{
			"context-with-vars": {
				Client: "dummy",
				Search: client.LogSearch{
					Variables: map[string]client.VariableDefinition{
						"sessionId": {
							Description: "The session ID to filter on.",
							Required:    true,
						},
					},
				},
			},
		},
	}

	bundle, err := BuildMCPServer(cfg)
	require.NoError(t, err)

	handler := bundle.ToolHandlers["get_context_details"]
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_context_details",
			Arguments: map[string]interface{}{
				"contextId": "context-with-vars",
			},
		},
	}

	res, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError, "tool call should not return an error")

	var result client.LogSearch
	textContent := res.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(textContent), &result)
	require.NoError(t, err, "failed to unmarshal result from tool")

	assert.NotNil(t, result.Variables)
	assert.Contains(t, result.Variables, "sessionId")
	assert.True(t, result.Variables["sessionId"].Required)
	assert.Equal(t, "The session ID to filter on.", result.Variables["sessionId"].Description)
}

func TestMCP_QueryLogs_MissingRequiredVariable(t *testing.T) {
	cfg := &config.ContextConfig{
		Clients: config.Clients{"dummy": {Type: "local"}},
		Contexts: config.Contexts{
			"context-with-req-var": {
				Client: "dummy",
				Search: client.LogSearch{
					Variables: map[string]client.VariableDefinition{
						"sessionId": {
							Description: "The user session ID.",
							Required:    true,
						},
					},
				},
			},
		},
	}

	bundle, err := BuildMCPServer(cfg)
	require.NoError(t, err)

	handler := bundle.ToolHandlers["query_logs"]
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query_logs",
			Arguments: map[string]interface{}{
				"contextId": "context-with-req-var",
				// "variables" is intentionally omitted
			},
		},
	}

	res, err := handler(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, res.IsError, "tool call should have returned an error")

	textContent := res.Content[0].(mcp.TextContent).Text
	assert.Contains(t, textContent, "Missing required variable 'sessionId'")
	assert.Contains(t, textContent, "Please ask the user for 'The user session ID.'")
}

func TestMCP_QueryLogs_WithVariables(t *testing.T) {
	cfg := &config.ContextConfig{
		Clients: config.Clients{"dummy": {Type: "local", Options: ty.MI{"cmd": "echo \"hello\""}}},
		Contexts: config.Contexts{
			"context-to-query": {
				Client: "dummy",
				Search: client.LogSearch{
					Fields: ty.MS{"id": "${sessionId}"},
					Variables: map[string]client.VariableDefinition{
						"sessionId": {Required: true},
					},
				},
			},
		},
	}

	bundle, err := BuildMCPServer(cfg)
	require.NoError(t, err)

	handler := bundle.ToolHandlers["query_logs"]
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query_logs",
			Arguments: map[string]interface{}{
				"contextId": "context-to-query",
				"variables": map[string]interface{}{
					"sessionId": "session-123",
				},
			},
		},
	}

	res, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError, "tool call should not return an error, got: %s", res.Content[0].(mcp.TextContent).Text)

	var result map[string]interface{}
	textContent := res.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(textContent), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "entries")
	assert.Contains(t, result, "meta")
}
