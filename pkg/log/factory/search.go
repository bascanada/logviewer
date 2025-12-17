package factory

import (
	"context"
	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
)

type SearchFactory interface {
	GetSearchResult(ctx context.Context, contextId string, inherits []string, logSearch client.LogSearch, runtimeVars map[string]string) (client.LogSearchResult, error)
	GetSearchContext(ctx context.Context, contextId string, inherits []string, logSearch client.LogSearch, runtimeVars map[string]string) (*config.SearchContext, error)
	// GetFieldValues returns distinct values for the specified fields.
	// If fields is empty, returns values for all fields found in the logs.
	GetFieldValues(ctx context.Context, contextId string, inherits []string, logSearch client.LogSearch, fields []string, runtimeVars map[string]string) (map[string][]string, error)
}

type logSearchFactory struct {
	clientsFactory  LogClientFactory
	searchesContext config.Contexts

	config config.ContextConfig
}

func (sf *logSearchFactory) GetSearchContext(ctx context.Context, contextId string, inherits []string, logSearch client.LogSearch, runtimeVars map[string]string) (*config.SearchContext, error) {
	searchContext, err := sf.config.GetSearchContext(contextId, inherits, logSearch, runtimeVars)
	if err != nil {
		return nil, err
	}
	return &searchContext, nil
}

func (sf *logSearchFactory) GetSearchResult(ctx context.Context, contextId string, inherits []string, logSearch client.LogSearch, runtimeVars map[string]string) (client.LogSearchResult, error) {

	searchContext, err := sf.config.GetSearchContext(contextId, inherits, logSearch, runtimeVars)
	if err != nil {
		return nil, err
	}

	logClient, err := sf.clientsFactory.Get(searchContext.Client)
	if err != nil {
		return nil, err
	}

	sr, err := (*logClient).Get(ctx, &searchContext.Search)

	return sr, err
}

func (sf *logSearchFactory) GetFieldValues(ctx context.Context, contextId string, inherits []string, logSearch client.LogSearch, fields []string, runtimeVars map[string]string) (map[string][]string, error) {
	searchContext, err := sf.config.GetSearchContext(contextId, inherits, logSearch, runtimeVars)
	if err != nil {
		return nil, err
	}

	logClient, err := sf.clientsFactory.Get(searchContext.Client)
	if err != nil {
		return nil, err
	}

	return (*logClient).GetFieldValues(ctx, &searchContext.Search, fields)
}

func GetLogSearchFactory(
	f LogClientFactory,
	c config.ContextConfig,
) (SearchFactory, error) {

	factory := new(logSearchFactory)
	factory.searchesContext = make(config.Contexts)
	factory.clientsFactory = f
	factory.config = c

	return factory, nil
}
