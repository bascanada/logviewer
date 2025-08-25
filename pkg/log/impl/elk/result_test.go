package elk

import (
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func TestElkSearchResult_GetPaginationInfo(t *testing.T) {
	t.Run("no size set, no pagination", func(t *testing.T) {
		search := &client.LogSearch{}
		result := ElkSearchResult{search: search}
		assert.Nil(t, result.GetPaginationInfo())
	})

	t.Run("results less than size, no more pages", func(t *testing.T) {
		search := &client.LogSearch{Size: ty.Opt[int]{Value: 10, Set: true}}
		result := ElkSearchResult{
			search: search,
			result: Hits{Hits: make([]Hit, 5)},
		}
		assert.Nil(t, result.GetPaginationInfo())
	})

	t.Run("results equal size, more pages", func(t *testing.T) {
		search := &client.LogSearch{Size: ty.Opt[int]{Value: 10, Set: true}}
		result := ElkSearchResult{
			search: search,
			result: Hits{Hits: make([]Hit, 10)},
		}
		paginationInfo := result.GetPaginationInfo()
		assert.NotNil(t, paginationInfo)
		assert.True(t, paginationInfo.HasMore)
		assert.Equal(t, "10", paginationInfo.NextPageToken)
	})

	t.Run("with existing page token", func(t *testing.T) {
		search := &client.LogSearch{
			Size:      ty.Opt[int]{Value: 10, Set: true},
			PageToken: ty.Opt[string]{Value: "10", Set: true},
		}
		result := ElkSearchResult{
			search: search,
			result: Hits{Hits: make([]Hit, 10)},
		}
		paginationInfo := result.GetPaginationInfo()
		assert.NotNil(t, paginationInfo)
		assert.True(t, paginationInfo.HasMore)
		assert.Equal(t, "20", paginationInfo.NextPageToken)
	})

	t.Run("invalid page token", func(t *testing.T) {
		search := &client.LogSearch{
			Size:      ty.Opt[int]{Value: 10, Set: true},
			PageToken: ty.Opt[string]{Value: "invalid", Set: true},
		}
		result := ElkSearchResult{
			search: search,
			result: Hits{Hits: make([]Hit, 10)},
		}
		paginationInfo := result.GetPaginationInfo()
		assert.NotNil(t, paginationInfo)
		assert.True(t, paginationInfo.HasMore)
		assert.Equal(t, "10", paginationInfo.NextPageToken)
	})
}
