package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

// ErrContextNotFound is a sentinel error allowing callers to detect missing contexts via errors.Is.
var ErrContextNotFound = errors.New("context not found")

func LoadContextConfig(configPath string) (*ContextConfig, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at path: %s", configPath)
	}

	var config ContextConfig
	if err := ty.ReadJsonFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
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

		return searchContext, nil
	} else {
		return SearchContext{}, fmt.Errorf("%w: %s", ErrContextNotFound, contextId)
	}
}
