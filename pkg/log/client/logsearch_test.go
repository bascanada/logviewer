package client_test

import (
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func TestMerging(t *testing.T) {

	searchParent := client.LogSearch{
		Refresh: client.RefreshOptions{},
		Size:    ty.OptWrap(100),
	}

	searchChild := client.LogSearch{
		Refresh: client.RefreshOptions{
			Duration: ty.OptWrap("15s"),
		},
	}

	_ = searchParent.MergeInto(&searchChild)

	str, _ := ty.ToJSONString(&searchParent)

	restoreParent := client.LogSearch{}

	_ = ty.FromJSONString(str, &restoreParent)

	assert.Equal(t, searchParent.Refresh.Duration.Value, "15s", "should be the same")
	// assert.Equal(t, searchParent)

}

func TestMergingFollow(t *testing.T) {
	searchParent := client.LogSearch{
		Follow: false,
	}

	searchChild := client.LogSearch{
		Follow: true,
	}

	_ = searchParent.MergeInto(&searchChild)

	assert.True(t, searchParent.Follow, "Follow should be true after merge")

}

func TestMergingPrinterOptions(t *testing.T) {

	searchParent := client.LogSearch{

		PrinterOptions: client.PrinterOptions{

			Template: ty.OptWrap("template1"),
		},
	}

	searchChild := client.LogSearch{

		PrinterOptions: client.PrinterOptions{

			Color: ty.OptWrap(true),
		},
	}

	_ = searchParent.MergeInto(&searchChild)

	assert.Equal(t, "template1", searchParent.PrinterOptions.Template.Value)

	assert.True(t, searchParent.PrinterOptions.Color.Value)

}
