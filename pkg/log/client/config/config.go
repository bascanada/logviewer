package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
)

// ErrContextNotFound is a sentinel error allowing callers to detect missing contexts via errors.Is.
var ErrContextNotFound = errors.New("context not found")

// Sentinel errors returned by LoadContextConfig so callers can detect exact
// failure modes using errors.Is().
var (
	ErrConfigParse = errors.New("invalid config content")
	ErrNoContexts  = errors.New("no contexts found in config file")
	ErrNoClients   = errors.New("no clients found in config file")
)

const (
	// EnvConfigPath is the environment variable used to override the config path
	EnvConfigPath = "LOGVIEWER_CONFIG"

	// DefaultConfigDir is the directory under the user's home where the config
	// file is expected when no explicit path or env var is provided.
	DefaultConfigDir = ".logviewer"

	// DefaultConfigFile is the config filename to look for in the default dir.
	DefaultConfigFile = "config.yaml"
)

func LoadContextConfig(configPath string) (*ContextConfig, error) {
	// If no path provided, first check LOGVIEWER_CONFIG env var, then default $HOME/.logviewer/config.yaml
	if strings.TrimSpace(configPath) == "" {
		if envPath := strings.TrimSpace(os.Getenv(EnvConfigPath)); envPath != "" {
			configPath = envPath
		} else if home, err := os.UserHomeDir(); err == nil {
			defaultPath := filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
			if _, err := os.Stat(defaultPath); err == nil {
				configPath = defaultPath
			}
		}
	}

	if strings.TrimSpace(configPath) == "" {
		return nil, fmt.Errorf("config file not found at path: %s", configPath)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at path: %s", configPath)
	}

	// Read file contents and support JSON or YAML formats
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config ContextConfig
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("%w: parsing JSON %s: %v", ErrConfigParse, configPath, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("%w: parsing YAML %s: %v", ErrConfigParse, configPath, err)
		}
	default:
		// Try JSON then YAML as a fallback
		if err := json.Unmarshal(data, &config); err == nil {
			break
		}
		if err := yaml.Unmarshal(data, &config); err == nil {
			break
		}
		return nil, fmt.Errorf("%w: unsupported or invalid config format for file: %s", ErrConfigParse, configPath)
	}

	if len(config.Contexts) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoContexts, configPath)
	}

	if len(config.Clients) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoClients, configPath)
	}

	if err := validateClients(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateClients performs lightweight validation of configured clients and
// returns a combined error describing any missing required options. This is
// intended to help users detect common config typos (e.g. using "option"
// instead of "options") and missing fields such as Url/Endpoint/Addr.
func validateClients(cc *ContextConfig) error {
	problems := []string{}

	for name, c := range cc.Clients {
		switch strings.ToLower(c.Type) {
		case "splunk":
			if c.Options.GetString("Url") == "" {
				problems = append(problems, fmt.Sprintf("client '%s' (splunk) missing required option 'Url'", name))
			}
		case "opensearch", "kibana":
			if c.Options.GetString("Endpoint") == "" {
				problems = append(problems, fmt.Sprintf("client '%s' (%s) missing required option 'Endpoint'", name, c.Type))
			}
		case "ssh":
			if c.Options.GetString("Addr") == "" {
				problems = append(problems, fmt.Sprintf("client '%s' (ssh) missing required option 'Addr'", name))
			}
		case "docker":
			// docker Host can be empty (falls back to unix socket), so just warn
			// but do not fail.
			// no-op
		default:
			// Unknown types are not validated here.
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid client configuration:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

type Client struct {
	Type    string `json:"type"`
	Options ty.MI  `json:"options"`
}

type SearchContext struct {
	Client        string           `json:"client"`
	SearchInherit []string         `json:"searchInherit"`
	Search        client.LogSearch `json:"search"`
}

type Clients map[string]Client

type Searches map[string]client.LogSearch

type Contexts map[string]SearchContext

type ContextConfig struct {
	Clients
	Searches
	Contexts
}

func (cc ContextConfig) GetSearchContext(contextId string, inherits []string, logSearch client.LogSearch) (SearchContext, error) {
	if contextId == "" {
		return SearchContext{}, errors.New("contextId is empty , required when using config")
	}
	if searchContext, b := cc.Contexts[contextId]; b {
		inherits := append(searchContext.SearchInherit, inherits...)
		if len(inherits) > 0 {
			for _, inherit := range inherits {
				if inheritSearch, b := cc.Searches[inherit]; b {
					searchContext.Search.MergeInto(&inheritSearch)
				} else {
					return SearchContext{}, errors.New("failed to find a search context for " + inherit)
				}
			}
		}

		searchContext.Search.MergeInto(&logSearch)

		// Resolve env vars inside search options (MI)
		searchContext.Search.Options = searchContext.Search.Options.ResolveVariables()

		return searchContext, nil
	} else {
		return SearchContext{}, fmt.Errorf("%w: %s", ErrContextNotFound, contextId)
	}
}
