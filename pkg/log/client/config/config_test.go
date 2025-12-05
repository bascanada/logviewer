package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
)

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	}
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}

const sampleJSON = `{
  "clients": {
    "c1": { "type": "local", "options": {} }
  },
  "searches": {},
  "contexts": {
    "ctx1": { "client": "c1", "searchInherit": [], "search": {} }
  }
}`

const sampleYAML = `clients:
  c1:
    type: local
    options: {}
searches: {}
contexts:
  ctx1:
    client: c1
    searchInherit: []
    search: {}
`

func TestLoadContextConfig_JSON(t *testing.T) {
	path := writeTemp(t, "", "cfg.json", sampleJSON)
	cfg, err := LoadContextConfig(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Contexts) != 1 {
		t.Fatalf("unexpected config contents: contexts=%d", len(cfg.Contexts))
	}
	if _, ok := cfg.Clients["c1"]; !ok {
		t.Fatalf("expected client 'c1' present")
	}
}

func TestLoadContextConfig_YAML(t *testing.T) {
	path := writeTemp(t, "", "cfg.yaml", sampleYAML)
	cfg, err := LoadContextConfig(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Contexts) != 1 {
		t.Fatalf("unexpected config contents: contexts=%d", len(cfg.Contexts))
	}
	if _, ok := cfg.Clients["c1"]; !ok {
		t.Fatalf("expected client 'c1' present")
	}
}

func TestLoadContextConfig_EnvVarPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "envcfg.yaml", sampleYAML)

	// set env var and call with empty configPath
	if err := os.Setenv(EnvConfigPath, path); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer os.Unsetenv(EnvConfigPath)

	cfg, err := LoadContextConfig("")
	if err != nil {
		t.Fatalf("expected no error loading via env var, got %v", err)
	}
	if len(cfg.Contexts) != 1 {
		t.Fatalf("unexpected config contents from env var: contexts=%d", len(cfg.Contexts))
	}
	if _, ok := cfg.Clients["c1"]; !ok {
		t.Fatalf("expected client 'c1' present from env var config")
	}

	// ensure env var path takes precedence: create a default file that would be different
	defaultDir := filepath.Join(dir, DefaultConfigDir)
	defaultPath := writeTemp(t, defaultDir, DefaultConfigFile, sampleJSON)
	// override env var to point to a different file and verify it's used
	if err := os.Setenv(EnvConfigPath, defaultPath); err != nil {
		t.Fatalf("failed to set env var to default path: %v", err)
	}
	cfg2, err := LoadContextConfig("")
	if err != nil {
		t.Fatalf("expected no error loading via env var (default override), got %v", err)
	}
	if len(cfg2.Contexts) != 1 {
		t.Fatalf("unexpected config contents from overridden env var: contexts=%d", len(cfg2.Contexts))
	}
	if _, ok := cfg2.Clients["c1"]; !ok {
		t.Fatalf("expected client 'c1' present from overridden env var config")
	}

	// cleanup env var
	os.Unsetenv(EnvConfigPath)
}

func ExampleLoadContextConfig() {
	// Quick example demonstrating passing explicit path.
	// (Not executed as part of tests, just documentation.)
	fmt.Println("use LoadContextConfig(path) to load a config file")
	// Output: use LoadContextConfig(path) to load a config file
}

func TestLoadContextConfig_InvalidContent(t *testing.T) {
	path := writeTemp(t, "", "bad.json", "{ invalid json }")
	_, err := LoadContextConfig(path)
	if err == nil {
		t.Fatalf("expected parse error, got nil")
	}
	if !errors.Is(err, ErrConfigParse) {
		t.Fatalf("expected ErrConfigParse, got %v", err)
	}
}

func TestLoadContextConfig_MissingSections(t *testing.T) {
	// create a file with no clients: the loader should add a default 'local' client
	noClients := `{"searches":{}, "contexts": {"a": {"client":"c","searchInherit":[],"search":{}}}}`
	path := writeTemp(t, "", "noclients.json", noClients)
	cfg, err := LoadContextConfig(path)
	if err != nil {
		t.Fatalf("expected config to load and default local client added, got %v", err)
	}
	if _, ok := cfg.Clients["local"]; !ok {
		t.Fatalf("expected default 'local' client to be added")
	}

	// create a file with no contexts
	noContexts := `{"clients": {"c1": {"type":"local","options":{}}}, "searches":{}}`
	path2 := writeTemp(t, "", "nocontexts.json", noContexts)
	_, err2 := LoadContextConfig(path2)
	if err2 == nil || !errors.Is(err2, ErrNoContexts) {
		t.Fatalf("expected ErrNoContexts, got %v", err2)
	}
}

func TestGetSearchContext_VariableDefaults(t *testing.T) {
	// Test that default values from variable definitions are used when runtime vars are not provided
	configContent := `{
		"clients": {
			"c1": { "type": "local", "options": {} }
		},
		"searches": {},
		"contexts": {
			"test-ctx": {
				"client": "c1",
				"searchInherit": [],
				"search": {
					"fields": {
						"level": "${log_level}",
						"service": "${service_name}"
					},
					"options": {
						"cmd": "cat /var/log/${log_file}"
					},
					"variables": {
						"log_level": {
							"description": "Log level filter",
							"type": "string",
							"default": "INFO",
							"required": false
						},
						"service_name": {
							"description": "Service name",
							"type": "string",
							"default": "api-service",
							"required": false
						},
						"log_file": {
							"description": "Log file name",
							"type": "string",
							"default": "app.log",
							"required": false
						}
					}
				}
			}
		}
	}`

	path := writeTemp(t, "", "vardefaults.json", configContent)
	cfg, err := LoadContextConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Test 1: No runtime vars provided - should use all defaults
	ctx1, err := cfg.GetSearchContext("test-ctx", nil, client.LogSearch{}, nil)
	if err != nil {
		t.Fatalf("failed to get search context: %v", err)
	}
	if ctx1.Search.Fields["level"] != "INFO" {
		t.Errorf("expected level=INFO (default), got %s", ctx1.Search.Fields["level"])
	}
	if ctx1.Search.Fields["service"] != "api-service" {
		t.Errorf("expected service=api-service (default), got %s", ctx1.Search.Fields["service"])
	}
	cmdOpt := ctx1.Search.Options.GetString("cmd")
	if cmdOpt != "cat /var/log/app.log" {
		t.Errorf("expected cmd with app.log (default), got %s", cmdOpt)
	}

	// Test 2: Some runtime vars provided - should override defaults
	runtimeVars := map[string]string{
		"log_level": "ERROR",
		// service_name not provided, should use default
		"log_file": "error.log",
	}
	ctx2, err := cfg.GetSearchContext("test-ctx", nil, client.LogSearch{}, runtimeVars)
	if err != nil {
		t.Fatalf("failed to get search context with runtime vars: %v", err)
	}
	if ctx2.Search.Fields["level"] != "ERROR" {
		t.Errorf("expected level=ERROR (runtime), got %s", ctx2.Search.Fields["level"])
	}
	if ctx2.Search.Fields["service"] != "api-service" {
		t.Errorf("expected service=api-service (default), got %s", ctx2.Search.Fields["service"])
	}
	cmdOpt2 := ctx2.Search.Options.GetString("cmd")
	if cmdOpt2 != "cat /var/log/error.log" {
		t.Errorf("expected cmd with error.log (runtime), got %s", cmdOpt2)
	}
}
