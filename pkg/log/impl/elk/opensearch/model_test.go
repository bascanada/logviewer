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
