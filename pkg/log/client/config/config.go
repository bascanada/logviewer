package config

import (
	"errors"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

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
		return SearchContext{}, errors.New("cant find context : " + contextId)
	}
}
