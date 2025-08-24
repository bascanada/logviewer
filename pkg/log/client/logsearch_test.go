package client

import (
	"testing"

	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func TestMerging(t *testing.T) {

	searchParent := LogSearch{
		Refresh: RefreshOptions{},
		Size:    ty.OptWrap(100),
	}

	searchChild := LogSearch{
		Refresh: RefreshOptions{
			Duration: ty.OptWrap("15s"),
		},
	}

	searchParent.MergeInto(&searchChild)

	str, _ := ty.ToJsonString(&searchParent)

	restoreParent := LogSearch{}

	ty.FromJsonString(str, &restoreParent)

	assert.Equal(t, searchParent.Refresh.Duration.Value, "15s", "should be the same")
	//assert.Equal(t, searchParent)

}
