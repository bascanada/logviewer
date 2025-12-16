package opensearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
)

func TestBody(t *testing.T) {

	logSearch := client.LogSearch{
		Fields: map[string]string{
			"instance":        "pod-1234",
			"applicationName": "mfx.services.tsapi",
		},
		Range: client.SearchRange{Last: ty.OptWrap("30m")},
		Size:  ty.OptWrap(100),
	}

	request, err := GetSearchRequest(&logSearch)
	if err != nil {
		t.Error(err)
	}

	b, _ := json.MarshalIndent(&request, "", "    ")

	fmt.Println(string(b))
}

func TestGetSearchRequest_Pagination(t *testing.T) {
	t.Run("no page token", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Range: client.SearchRange{Last: ty.OptWrap("30m")},
		}
		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if request.From != 0 {
			t.Errorf("expected From to be 0, but got %d", request.From)
		}
	})

	t.Run("with page token", func(t *testing.T) {
		logSearch := &client.LogSearch{
			PageToken: ty.Opt[string]{Value: "50", Set: true},
			Range:     client.SearchRange{Last: ty.OptWrap("30m")},
		}
		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if request.From != 50 {
			t.Errorf("expected From to be 50, but got %d", request.From)
		}
	})

	t.Run("with invalid page token", func(t *testing.T) {
		logSearch := &client.LogSearch{
			PageToken: ty.Opt[string]{Value: "invalid", Set: true},
			Range:     client.SearchRange{Last: ty.OptWrap("30m")},
		}
		_, err := GetSearchRequest(logSearch)
		if err == nil {
			t.Errorf("expected error for invalid page token, got nil")
		}
	})
}

func TestGetSearchRequest_RecursiveFilter(t *testing.T) {
	t.Run("simple AND filter", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Filter: &client.Filter{
				Logic: client.LogicAnd,
				Filters: []client.Filter{
					{Field: "app", Value: "myapp"},
					{Field: "env", Value: "prod"},
				},
			},
			Range: client.SearchRange{Last: ty.OptWrap("30m")},
			Size:  ty.OptWrap(100),
		}

		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		b, _ := json.MarshalIndent(&request, "", "    ")
		queryStr := string(b)

		// Should contain bool must with the conditions
		if !strings.Contains(queryStr, "must") {
			t.Errorf("expected query to contain 'must', got: %s", queryStr)
		}
		if !strings.Contains(queryStr, "myapp") {
			t.Errorf("expected query to contain 'myapp', got: %s", queryStr)
		}
		if !strings.Contains(queryStr, "prod") {
			t.Errorf("expected query to contain 'prod', got: %s", queryStr)
		}
	})

	t.Run("simple OR filter", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Filter: &client.Filter{
				Logic: client.LogicOr,
				Filters: []client.Filter{
					{Field: "level", Value: "ERROR"},
					{Field: "level", Value: "WARN"},
				},
			},
			Range: client.SearchRange{Last: ty.OptWrap("30m")},
			Size:  ty.OptWrap(100),
		}

		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		b, _ := json.MarshalIndent(&request, "", "    ")
		queryStr := string(b)

		// Should contain bool should with minimum_should_match
		if !strings.Contains(queryStr, "should") {
			t.Errorf("expected query to contain 'should', got: %s", queryStr)
		}
		if !strings.Contains(queryStr, "minimum_should_match") {
			t.Errorf("expected query to contain 'minimum_should_match', got: %s", queryStr)
		}
	})

	t.Run("NOT filter", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Filter: &client.Filter{
				Logic: client.LogicNot,
				Filters: []client.Filter{
					{Field: "level", Value: "DEBUG"},
				},
			},
			Range: client.SearchRange{Last: ty.OptWrap("30m")},
			Size:  ty.OptWrap(100),
		}

		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		b, _ := json.MarshalIndent(&request, "", "    ")
		queryStr := string(b)

		// Should contain bool must_not
		if !strings.Contains(queryStr, "must_not") {
			t.Errorf("expected query to contain 'must_not', got: %s", queryStr)
		}
	})

	t.Run("nested (A OR B) AND C", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Filter: &client.Filter{
				Logic: client.LogicAnd,
				Filters: []client.Filter{
					{
						Logic: client.LogicOr,
						Filters: []client.Filter{
							{Field: "level", Value: "ERROR"},
							{Field: "level", Value: "WARN"},
						},
					},
					{Field: "app", Value: "myapp"},
				},
			},
			Range: client.SearchRange{Last: ty.OptWrap("30m")},
			Size:  ty.OptWrap(100),
		}

		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		b, _ := json.MarshalIndent(&request, "", "    ")
		queryStr := string(b)

		// Should contain nested structure
		if !strings.Contains(queryStr, "should") {
			t.Errorf("expected query to contain 'should' for OR, got: %s", queryStr)
		}
		if !strings.Contains(queryStr, "myapp") {
			t.Errorf("expected query to contain 'myapp', got: %s", queryStr)
		}
	})

	t.Run("combined legacy fields and new filter", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields: map[string]string{
				"env": "production",
			},
			Filter: &client.Filter{
				Logic: client.LogicOr,
				Filters: []client.Filter{
					{Field: "level", Value: "ERROR"},
					{Field: "level", Value: "WARN"},
				},
			},
			Range: client.SearchRange{Last: ty.OptWrap("30m")},
			Size:  ty.OptWrap(100),
		}

		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		b, _ := json.MarshalIndent(&request, "", "    ")
		queryStr := string(b)

		// Should contain both legacy and new filter conditions
		if !strings.Contains(queryStr, "production") {
			t.Errorf("expected query to contain 'production', got: %s", queryStr)
		}
		if !strings.Contains(queryStr, "should") {
			t.Errorf("expected query to contain 'should', got: %s", queryStr)
		}
	})
}
