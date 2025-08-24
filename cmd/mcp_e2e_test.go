package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/client/config"
	"github.com/berlingoqc/logviewer/pkg/ty"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMCP_ListContexts(t *testing.T) {
	cfg := &config.ContextConfig{Clients: config.Clients{}, Searches: config.Searches{}, Contexts: config.Contexts{}}
	cfg.Clients["dummy"] = config.Client{Type: "local", Options: ty.MI{}}
	cfg.Contexts["alpha"] = config.SearchContext{Client: "dummy", Search: client.LogSearch{}}

	bundle, err := BuildMCPServer(cfg)
	if err != nil { t.Fatalf("build error: %v", err) }
	handler := bundle.ToolHandlers["list_contexts"]
	res, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil { t.Fatalf("tool error: %v", err) }
	if len(res.Content) == 0 { t.Fatalf("no content") }
	raw := res.Content[0]
	b, _ := json.Marshal(raw)
	if !strings.Contains(string(b), "alpha") {
		t.Fatalf("expected alpha in payload: %s", string(b))
	}
}
