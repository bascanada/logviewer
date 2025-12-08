package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

func LoadState() (*State, error) {
	path, err := getStatePath()
		if err != nil {
			return nil, err
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

func SaveState(state *State) error {
	path, err := getStatePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	// Use 0600 permissions for state file as it might contain sensitive context choice?
	// Standard 0644 is probably fine as it's just a selection name.
	return os.WriteFile(path, data, 0644)
}
