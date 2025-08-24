package kibana

import (
	"github.com/bascanada/logviewer/pkg/log/impl/elk"
	"github.com/bascanada/logviewer/pkg/ty"
)

type Body struct {
	Size           int      `json:"size"`
	Sort           []ty.MI  `json:"sort"`
	Aggs           ty.MI    `json:"aggs,omitempty"`
	StoredFields   []string `json:"stored_fields,omitempty"`
	DocValueFields []ty.MI  `json:"docvalue_fields,omitempty"`
	Source         ty.MI    `json:"_source,omitempty"`
	Query          ty.MI    `json:"query"`
}

type Params struct {
	Index string `json:"index"`
	Body  Body   `json:"body"`
}

type SearchRequest struct {
	Params Params `json:"params"`
}

type Response struct {
	Hits elk.Hits `json:"hits"`
}

type SearchResponse struct {
	RawResponse Response `json:"rawResponse"`
}
