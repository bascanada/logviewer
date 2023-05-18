package opensearch

import (
	"errors"
	"fmt"

	"github.com/berlingoqc/logviewer/pkg/http"
	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/elk"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

type OpenSearchTarget struct {
	Endpoint string `json:"endpoint"`
}

type openSearchClient struct {
	target OpenSearchTarget
	client http.JsonGetClient
}

func (kc openSearchClient) Get(search client.LogSearch) (client.LogSearchResult, error) {
	var searchResult SearchResult

	index := search.Options.GetString("Index")

	if index == "" {
		return nil, errors.New("index is not provided for opensearch log client")
	}

	request, err := GetSearchRequest(search)
	if err != nil {
		return nil, err
	}

	err = kc.client.Get(fmt.Sprintf("/%s/_search", index), ty.MS{}, &request, &searchResult)
	if err != nil {
		return nil, err
	}

	return elk.GetSearchResult(&kc, search, searchResult.Hits), nil
}

func GetClient(target OpenSearchTarget) (client.LogClient, error) {
	client := new(openSearchClient)
	client.target = target
	client.client = http.GetClient(target.Endpoint)
	return client, nil
}
