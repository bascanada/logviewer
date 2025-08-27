package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_DefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".logviewer")
	err := os.MkdirAll(configDir, 0755)
	assert.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `
clients:
  my-client:
    type: ssh
    options:
      addr: 127.0.0.1:2222
contexts:
  my-context:
    client: my-client
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cmd := &cobra.Command{}
	addConfigFlag(cmd)

	cfg, err := loadConfig(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Clients, "my-client")
}

func TestLoadConfig_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	tmpfile := filepath.Join(dir, "test-config.json")

	configContent := `{
    "clients": {
      "my-client": {
        "type": "ssh",
        "options": {
          "addr": "127.0.0.1:2222"
        }
      }
    },
    "contexts": {
      "my-context": {
        "client": "my-client"
      }
    }
  }`

	err := os.WriteFile(tmpfile, []byte(configContent), 0644)
	assert.NoError(t, err)

	cmd := &cobra.Command{}
	addConfigFlag(cmd)
	cmd.Flags().Set("config", tmpfile)

	cfg, _, err := loadConfig(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Clients, "my-client")
}

func TestLoadConfig_YamlSupport(t *testing.T) {
	dir := t.TempDir()
	tmpfile := filepath.Join(dir, "test-config.yaml")

	configContent := `
clients:
  my-client:
    type: ssh
    options:
      addr: 127.0.0.1:2222
contexts:
  my-context:
    client: my-client
`
	err := os.WriteFile(tmpfile, []byte(configContent), 0644)
	assert.NoError(t, err)

	cmd := &cobra.Command{}
	addConfigFlag(cmd)
	cmd.Flags().Set("config", tmpfile)

	cfg, _, err := loadConfig(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Clients, "my-client")
}

func TestLoadConfig_JsonSupport(t *testing.T) {
	dir := t.TempDir()
	tmpfile := filepath.Join(dir, "test-config.json")

	configContent := `{
    "clients": {
      "my-client": {
        "type": "ssh",
        "options": {
          "addr": "127.0.0.1:2222"
        }
      }
    },
    "contexts": {
      "my-context": {
        "client": "my-client"
      }
    }
  }`

	err := os.WriteFile(tmpfile, []byte(configContent), 0644)
	assert.NoError(t, err)

	cmd := &cobra.Command{}
	addConfigFlag(cmd)
	cmd.Flags().Set("config", tmpfile)

	cfg, _, err := loadConfig(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Clients, "my-client")
}

func TestLoadConfig_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	tmpfile, err := os.Create(filepath.Join(dir, "test-config.txt"))
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	cmd := &cobra.Command{}
	addConfigFlag(cmd)
	cmd.Flags().Set("config", tmpfile.Name())

	_, _, err = loadConfig(cmd)
	assert.Error(t, err)
}

func TestLoadConfig_NoFileFound(t *testing.T) {
	cmd := &cobra.Command{}
	addConfigFlag(cmd)
	cmd.Flags().Set("config", "non-existent-file.json")

	_, _, err := loadConfig(cmd)
	assert.Error(t, err)
}
