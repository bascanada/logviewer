package logclient

import (
	"context"
	"testing"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

func TestSplunkLogSearchResult_GetEntries_Follow(t *testing.T) {
	defer gock.Off()

	sid := "my-follow-sid"
	gock.New("http://splunk.com:8080").
		Get("/search/jobs/" + sid + "/events").
		Reply(200).
		JSON(ty.MI{
			"results": []ty.MS{
				{"_raw": "new log 1"},
			},
		})

	logClient, err := GetClient(SplunkLogSearchClientOptions{
		Url:                       "http://splunk.com:8080",
		FollowPollIntervalSeconds: 1,
	})
	assert.NoError(t, err)

	splunkClient := logClient.(SplunkLogSearchClient)
	searchResult := SplunkLogSearchResult{
		logClient: &splunkClient,
		sid:       sid,
		isFollow:  true,
		search:    &client.LogSearch{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, entryChan, err := searchResult.GetEntries(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, entryChan)

	select {
	case entries, ok := <-entryChan:
		assert.True(t, ok)
		assert.Len(t, entries, 1)
		assert.Equal(t, "new log 1", entries[0].Message)
	case <-ctx.Done():
		t.Fatal("timed out waiting for log entries")
	}

	assert.True(t, gock.IsDone())
}

func TestSplunkLogSearchResult_Close(t *testing.T) {
	defer gock.Off()

	sid := "my-follow-sid"
	gock.New("http://splunk.com:8080").
		Delete("/search/jobs/" + sid).
		Reply(200)

	logClient, err := GetClient(SplunkLogSearchClientOptions{
		Url: "http://splunk.com:8080",
	})
	assert.NoError(t, err)

	splunkClient := logClient.(SplunkLogSearchClient)
	searchResult := SplunkLogSearchResult{
		logClient: &splunkClient,
		sid:       sid,
		isFollow:  true,
		search:    &client.LogSearch{},
	}

	err = searchResult.Close()
	assert.NoError(t, err)

	assert.True(t, gock.IsDone())
}
