package factory

import (
	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/client/config"
)

type logSearchFactory struct {
	clientsFactory  *logClientFactory
	searchesContext config.Contexts

	config config.ContextConfig
}

func (sf *logSearchFactory) GetSearchResult(contextId string, inherits []string, logSearch client.LogSearch) (client.LogSearchResult, error) {

	searchContext, err := sf.config.GetSearchContext(contextId, inherits, logSearch)
	if err != nil {
		return nil, err
	}

	logClient, err := sf.clientsFactory.clients.Get(searchContext.Client)
	if err != nil {
		return nil, err
	}

	sr, err := (*logClient).Get(&searchContext.Search)

	return sr, err
}

func GetLogSearchFactory(
	f *logClientFactory,
	c config.ContextConfig,
) (*logSearchFactory, error) {

	factory := new(logSearchFactory)
	factory.searchesContext = make(config.Contexts)
	factory.clientsFactory = f
	factory.config = c

	return factory, nil
}
