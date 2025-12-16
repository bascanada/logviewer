package kibana

import (
	"context"
	"errors"

	"github.com/bascanada/logviewer/pkg/http"
	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/operator"
	"github.com/bascanada/logviewer/pkg/log/impl/elk"
	"github.com/bascanada/logviewer/pkg/ty"
)

type KibanaTarget struct {
	Endpoint string `json:"endpoint"`
}

type kibanaClient struct {
	target KibanaTarget
	client http.HttpClient
}

func (kc kibanaClient) Get(ctx context.Context, search *client.LogSearch) (client.LogSearchResult, error) {
	var searchResponse SearchResponse

	request, err := getSearchRequest(search)
	if err != nil {
		return nil, err
	}

	err = kc.client.PostJson("/internal/search/es", ty.MS{
		"kbn-version": search.Options.GetOr("version", "7.10.2").(string),
	}, &request, &searchResponse, nil)
	if err != nil {
		return nil, err
	}

	return elk.GetSearchResult(&kc, search, searchResponse.RawResponse.Hits), nil
}

// buildKibanaCondition builds a single Kibana query condition from a filter leaf.
func buildKibanaCondition(f *client.Filter) ty.MI {
	if f.Field == "" {
		return nil
	}

	op := f.Op
	if op == "" {
		op = operator.Match
	}

	// Handle special "_" sentinel for full-text search
	field := f.Field
	if field == "_" {
		field = "_all"
	}

	switch op {
	case operator.Regex:
		return ty.MI{
			"regexp": ty.MI{
				field: f.Value,
			},
		}
	case operator.Wildcard:
		return ty.MI{
			"wildcard": ty.MI{
				field: f.Value,
			},
		}
	case operator.Exists:
		return ty.MI{
			"exists": ty.MI{
				"field": field,
			},
		}
	case operator.Equals:
		return ty.MI{
			"term": ty.MI{
				field: f.Value,
			},
		}
	default: // match - use match_phrase for Kibana (default behavior)
		return ty.MI{
			"match_phrase": ty.MI{
				field: f.Value,
			},
		}
	}
}

// buildKibanaQuery recursively builds a Kibana bool query from a Filter AST.
func buildKibanaQuery(f *client.Filter) ty.MI {
	if f == nil {
		return nil
	}

	// Handle Leaf (Condition)
	if f.Field != "" {
		return buildKibanaCondition(f)
	}

	// Handle Branch (Group)
	if f.Logic == "" || len(f.Filters) == 0 {
		return nil
	}

	var clauses []ty.MI
	for _, child := range f.Filters {
		clause := buildKibanaQuery(&child)
		if clause != nil {
			clauses = append(clauses, clause)
		}
	}

	if len(clauses) == 0 {
		return nil
	}

	// If only one clause and AND, return it directly
	if len(clauses) == 1 && f.Logic == client.LogicAnd {
		return clauses[0]
	}

	switch f.Logic {
	case client.LogicAnd:
		return ty.MI{
			"bool": ty.MI{
				"must": clauses,
			},
		}
	case client.LogicOr:
		return ty.MI{
			"bool": ty.MI{
				"should":               clauses,
				"minimum_should_match": 1,
			},
		}
	case client.LogicNot:
		return ty.MI{
			"bool": ty.MI{
				"must_not": clauses,
			},
		}
	}

	return nil
}

func getSearchRequest(search *client.LogSearch) (SearchRequest, error) {
	request := SearchRequest{}

	index := search.Options.GetString("index")

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

	// Build conditions from the effective filter
	conditions := []ty.MI{
		{"match_all": ty.MI{}},
	}

	effectiveFilter := search.GetEffectiveFilter()
	if effectiveFilter != nil {
		filterQuery := buildKibanaQuery(effectiveFilter)
		if filterQuery != nil {
			conditions = append(conditions, filterQuery)
		}
	}

	// Add timestamp range
	conditions = append(conditions, elk.GetDateRangeConditon(gte, lte))

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
	client.client = http.GetClient(target.Endpoint)
	return client, nil
}
