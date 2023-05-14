package kibana

import (
	"errors"

	"github.com/berlingoqc/logexplorer/pkg/http"
	"github.com/berlingoqc/logexplorer/pkg/log/client"
	"github.com/berlingoqc/logexplorer/pkg/log/elk"
	"github.com/berlingoqc/logexplorer/pkg/ty"
)

type KibanaTarget struct {
	Endpoint string `json:"endpoint"`
}

type kibanaClient struct {
	target KibanaTarget
	client http.JsonPostClient
}

func (kc kibanaClient) Get(search client.LogSearch) (client.LogSearchResult, error) {
	var searchResponse SearchResponse

	request, err := getSearchRequest(search)
	if err != nil {
		return nil, err
	}

	err = kc.client.Post("/internal/search/es", ty.MS{
		"kbn-version": search.Options.GetOr("version", "7.10.2").(string),
	}, &request, &searchResponse)
	if err != nil {
		return nil, err
	}

	return elk.GetSearchResult(&kc, search, searchResponse.RawResponse.Hits), nil
}

func getSearchRequest(search client.LogSearch) (SearchRequest, error) {
	request := SearchRequest{}

	index := search.Options.GetString("Index")

	if index == "" {
		return request, errors.New("index is not provided for kibana log client")
	}

	gte, lte, err := elk.GetDateRange(search)
	if err != nil {
		return SearchRequest{}, err
	}

	request.Params.Index = index
	request.Params.Body.Size = search.Size.Value
	request.Params.Body.Sort = []ty.MI{
		{
			"@timestamp": ty.MI{
				"order":         "desc",
				"unmapped_type": "boolean",
			},
		},
	}
	request.Params.Body.StoredFields = []string{"*"}
	request.Params.Body.DocValueFields = []ty.MI{
		{
			"field":  "@timestamp",
			"format": "date_time",
		},
	}

	request.Params.Body.Source = ty.MI{
		"excludes": []interface{}{},
	}

	conditions := make([]ty.MI, len(search.Fields)+2)
	conditions[0] = ty.MI{
		"match_all": ty.MI{},
	}

	i := 1

	for k, v := range search.Fields {
		op, b := search.FieldsCondition[k]
		if !b || op == "" {
			op = "match_phrase"
		}
		conditions[i] = ty.MI{
			op: ty.MI{
				k: v,
			},
		}

		i += 1
	}

	conditions[len(conditions)-1] = elk.GetDateRangeConditon(gte, lte)

	request.Params.Body.Query = ty.MI{
		"bool": ty.MI{
			"filter": conditions,
		},
	}

	return request, nil
}

func GetClient(target KibanaTarget) (client.LogClient, error) {
	client := new(kibanaClient)
	client.target = target
	client.client = http.PostClient(target.Endpoint)
	return client, nil
}
