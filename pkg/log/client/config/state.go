package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// State holds persistent application state.
type State struct {
	CurrentContext string `yaml:"current-context"`
}

func getStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultConfigDir, "state.yaml"), nil
}

// LoadState reads the state from the configuration file.
func LoadState() (*State, error) {
	path, err := getStatePath()
	if err != nil {
		return &State{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return &State{}, err
	}
	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return &State{}, fmt.Errorf("parsing state file %s: %w", path, err)
	}
	return &state, nil
}

// SaveState writes the state to the configuration file.
func SaveState(state *State) error {
	path, err := getStatePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	// Use 0600 permissions for state file as it might contain sensitive context choice?
	// Standard 0600 is probably fine as it's just a selection name.
	return os.WriteFile(path, data, 0600)
}
