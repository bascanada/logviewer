package opensearch

import (
	"fmt"
	"strconv"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/operator"
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
	From  int        `json:"from,omitempty"`
	Sort  []SortItem `json:"sort"`
}

// buildOpenSearchCondition builds a single OpenSearch query condition from a filter leaf.
func buildOpenSearchCondition(f *client.Filter) Map {
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
		field = "_all" // OpenSearch full-text field
	}

	switch op {
	case operator.Regex:
		return Map{
			"regexp": Map{
				field: f.Value,
			},
		}
	case operator.Wildcard:
		return Map{
			"wildcard": Map{
				field: f.Value,
			},
		}
	case operator.Exists:
		return Map{
			"exists": Map{
				"field": field,
			},
		}
	case operator.Equals:
		return Map{
			"term": Map{
				field: f.Value,
			},
		}
	default: // match
		return Map{
			"match": Map{
				field: f.Value,
			},
		}
	}
}

// buildOpenSearchQuery recursively builds an OpenSearch bool query from a Filter AST.
func buildOpenSearchQuery(f *client.Filter) Map {
	if f == nil {
		return nil
	}

	// Handle Leaf (Condition)
	if f.Field != "" {
		return buildOpenSearchCondition(f)
	}

	// Handle Branch (Group)
	if f.Logic == "" || len(f.Filters) == 0 {
		return nil
	}

	var clauses []Map
	for _, child := range f.Filters {
		clause := buildOpenSearchQuery(&child)
		if clause != nil {
			clauses = append(clauses, clause)
		}
	}

	if len(clauses) == 0 {
		return nil
	}

	// If only one clause, return it directly (no need for bool wrapper)
	if len(clauses) == 1 && f.Logic == client.LogicAnd {
		return clauses[0]
	}

	switch f.Logic {
	case client.LogicAnd:
		return Map{
			"bool": Map{
				"must": clauses,
			},
		}
	case client.LogicOr:
		return Map{
			"bool": Map{
				"should":               clauses,
				"minimum_should_match": 1,
			},
		}
	case client.LogicNot:
		return Map{
			"bool": Map{
				"must_not": clauses,
			},
		}
	}

	return nil
}

func GetSearchRequest(logSearch *client.LogSearch) (SearchRequest, error) {
	gte, lte, err := elk.GetDateRange(logSearch)
	if err != nil {
		return SearchRequest{}, err
	}

	// Build conditions from the effective filter
	var filterConditions []Map

	// 1. Add Native Query if provided (using query_string for raw Lucene syntax)
	if logSearch.NativeQuery.Set && logSearch.NativeQuery.Value != "" {
		filterConditions = append(filterConditions, Map{
			"query_string": Map{
				"query": logSearch.NativeQuery.Value,
			},
		})
	}

	// 2. Add effective filter conditions
	effectiveFilter := logSearch.GetEffectiveFilter()
	if effectiveFilter != nil {
		filterQuery := buildOpenSearchQuery(effectiveFilter)
		if filterQuery != nil {
			filterConditions = append(filterConditions, filterQuery)
		}
	}

	// Add timestamp range condition
	timestampCondition := Map{
		"range": Map{
			"@timestamp": Map{
				"format": "strict_date_optional_time",
				"gte":    gte,
				"lte":    lte,
			},
		},
	}
	filterConditions = append(filterConditions, timestampCondition)

	query := Map{
		"bool": Map{
			"must": filterConditions,
		},
	}

	sortItem := SortItem{
		"@timestamp": map[string]string{
			"order":         "desc",
			"unmapped_type": "boolean",
		},
	}

	from := 0
	if logSearch.PageToken.Set && logSearch.PageToken.Value != "" {
		parsedOffset, err := strconv.Atoi(logSearch.PageToken.Value)
		if err != nil {
			return SearchRequest{}, fmt.Errorf("invalid page token: %w", err)
		}
		from = parsedOffset
	}

	return SearchRequest{
		Query: query,
		Sort:  []SortItem{sortItem},
		Size:  logSearch.Size.Value,
		From:  from,
	}, nil
}
