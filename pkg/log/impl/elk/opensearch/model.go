package opensearch

import (
	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/impl/elk"
)

type SearchResult struct {
	Took int `json:"took"`
	//timeout_out
	//_shards
	Hits elk.Hits `json:"hits"`
}

type SortItem map[string]map[string]string
type Map map[string]interface{}

type SearchRequest struct {
	Query Map        `json:"query"`
	Size  int        `json:"size"`
	Sort  []SortItem `json:"sort"`
}

func GetSearchRequest(logSearch *client.LogSearch) (SearchRequest, error) {

	conditions := make([]Map, len(logSearch.Fields)+1)

	index := 0

	gte, lte, err := elk.GetDateRange(logSearch)
	if err != nil {
		return SearchRequest{}, err
	}

	for k, v := range logSearch.Fields {

		op, b := logSearch.FieldsCondition[k]
		if !b || op == "" {
			op = "match"
		}

		conditions[index] = Map{
			op: Map{
				k: v,
			},
		}

		index += 1
	}

	conditions[index] = Map{
		"range": Map{
			"@timestamp": Map{
				"format": "strict_date_optional_time",
				"gte":    gte,
				"lte":    lte,
			},
		},
	}

	query := Map{
		"bool": Map{
			"must": conditions,
		},
	}

	sortItem := SortItem{
		"@timestamp": map[string]string{
			"order":         "desc",
			"unmapped_type": "boolean",
		},
	}

	return SearchRequest{
		Query: query,
		Sort:  []SortItem{sortItem},
		Size:  logSearch.Size.Value,
	}, nil
}
