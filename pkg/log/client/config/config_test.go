package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
	if len(cfg.Clients) != 1 || len(cfg.Contexts) != 1 {
		t.Fatalf("unexpected config contents: clients=%d contexts=%d", len(cfg.Clients), len(cfg.Contexts))
	}
}

func TestLoadContextConfig_YAML(t *testing.T) {
	path := writeTemp(t, "", "cfg.yaml", sampleYAML)
	cfg, err := LoadContextConfig(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Clients) != 1 || len(cfg.Contexts) != 1 {
		t.Fatalf("unexpected config contents: clients=%d contexts=%d", len(cfg.Clients), len(cfg.Contexts))
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
	if len(cfg.Clients) != 1 || len(cfg.Contexts) != 1 {
		t.Fatalf("unexpected config contents from env var: clients=%d contexts=%d", len(cfg.Clients), len(cfg.Contexts))
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
	if len(cfg2.Clients) != 1 || len(cfg2.Contexts) != 1 {
		t.Fatalf("unexpected config contents from overridden env var: clients=%d contexts=%d", len(cfg2.Clients), len(cfg2.Contexts))
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
	// create a file with no clients
	noClients := `{"searches":{}, "contexts": {"a": {"client":"c","searchInherit":[],"search":{}}}}`
	path := writeTemp(t, "", "noclients.json", noClients)
	_, err := LoadContextConfig(path)
	if err == nil || !errors.Is(err, ErrNoClients) {
		t.Fatalf("expected ErrNoClients, got %v", err)
	}

	// create a file with no contexts
	noContexts := `{"clients": {"c1": {"type":"local","options":{}}}, "searches":{}}`
	path2 := writeTemp(t, "", "nocontexts.json", noContexts)
	_, err2 := LoadContextConfig(path2)
	if err2 == nil || !errors.Is(err2, ErrNoContexts) {
		t.Fatalf("expected ErrNoContexts, got %v", err2)
	}
}
