package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"gopkg.in/yaml.v3"
)

// ErrContextNotFound is a sentinel error allowing callers to detect missing contexts via errors.Is.
var ErrContextNotFound = errors.New("context not found")

func Load(configPath string) (*ContextConfig, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at path: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config ContextConfig
	switch filepath.Ext(configPath) {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("error unmarshalling json: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("error unmarshalling yaml: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", configPath)
	}

	if len(config.Contexts) == 0 {
		return nil, errors.New("no contexts found in config file")
	}

	if len(config.Clients) == 0 {
		return nil, errors.New("no clients found in config file")
	}

	return &config, nil
}

type Client struct {
	Type    string `json:"type" yaml:"type"`
	Options ty.MI  `json:"options" yaml:"options"`
}

type SearchContext struct {
	Client        string           `json:"client" yaml:"client"`
	SearchInherit []string         `json:"searchInherit" yaml:"searchInherit"`
	Search        client.LogSearch `json:"search" yaml:"search"`
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
