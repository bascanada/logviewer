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

	_ = searchParent.MergeInto(&searchChild)

	str, _ := ty.ToJSONString(&searchParent)

	restoreParent := LogSearch{}

	_ = ty.FromJSONString(str, &restoreParent)

	assert.Equal(t, searchParent.Refresh.Duration.Value, "15s", "should be the same")
	// assert.Equal(t, searchParent)

}

func TestMergingFollow(t *testing.T) {
	searchParent := LogSearch{
		Follow: false,
	}

	searchChild := LogSearch{
		Follow: true,
	}

	_ = searchParent.MergeInto(&searchChild)

	assert.True(t, searchParent.Follow, "Follow should be true after merge")
}
