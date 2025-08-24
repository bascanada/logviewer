package logclient

import (
	"context"
	"testing"
	"time"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/ty"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

func TestSplunkLogClient(t *testing.T) {

	gock.New("http://splunk.com:8080").
		Post("/search/jobs").
		MatchType("application/x-www-form-urlencoded").
		Reply(200).
		JSON(ty.MI{"Sid": "mycid"})

	gock.New("http://splunk.com:8080").
		Get("/search/jobs/mycid").
		Reply(200).
		JSON(ty.MI{
			"entry": []ty.MI{
				{
					"content": ty.MI{
						"isDone": true,
					},
				},
			},
		})

	gock.New("http://splunk.com:8080").
		Get("/search/jobs/mycid/events").
		Reply(200).
		JSON(ty.MI{
			"results": []ty.MS{
				{
					"_raw":             "mylogentry",
					"_subsecond":       ".681",
					"_time":            "2024-06-21T08:56:05.681-07:00",
					"application_name": "wq.services.pet",
					"cat":              "BusinessExceptionHandler",
					"handler":          "CreatePet",
				},
				{
					"_raw":             "mylogentry",
					"_subsecond":       ".681",
					"_time":            "2024-06-21T08:56:05.681-07:00",
					"application_name": "wq.services.pet",
					"cat":              "BusinessExceptionHandler",
					"handler":          "DeletePet",
				},
			},
		})

	logClient, err := GetClient(SplunkLogSearchClientOptions{
		Url: "http://splunk.com:8080",
	})

	if err != nil {
		t.Error(err)
	}

	logSearch := client.LogSearch{
		Fields:  ty.MS{},
		Options: ty.MI{},
	}
	logSearch.Range.Gte.S("24h@h")
	logSearch.Range.Lte.S("now")

	logSearch.Fields["application_name"] = "wq.services.pet"
	logSearch.Options["index"] = "prd3392"

	result, err := logClient.Get(context.Background(), &logSearch)
	if err != nil {
		t.Error(err)
	}

	fields, _, err := result.GetFields(context.Background())
	if err != nil {
		t.Error(err)
	}

	logEntry, _, err := result.GetEntries(context.Background())
	if err != nil {
		t.Error(err)
	}

	logTimestamp, _ := time.Parse(time.RFC3339, "2024-06-21T08:56:05.681-07:00")

	assert.Equal(t, []string{"CreatePet", "DeletePet"}, fields["handler"])
	assert.Equal(t, "mylogentry", logEntry[0].Message)
	assert.Equal(t, "CreatePet", logEntry[0].Fields.GetString("handler"))
	assert.Equal(t, logTimestamp, logEntry[0].Timestamp)

}
