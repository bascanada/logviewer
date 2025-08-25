package opensearch

import (
	"encoding/json"
	"fmt"
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
		request, err := GetSearchRequest(logSearch)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if request.From != 0 {
			t.Errorf("expected From to be 0, but got %d", request.From)
		}
	})
}
