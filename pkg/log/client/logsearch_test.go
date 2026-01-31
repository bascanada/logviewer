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

func TestMergingFollow(t *testing.T) {
	searchParent := LogSearch{
		Follow: false,
	}

	searchChild := LogSearch{
		Follow: true,
	}

	searchParent.MergeInto(&searchChild)

	assert.True(t, searchParent.Follow, "Follow should be true after merge")
}

func TestClone(t *testing.T) {
	t.Run("Clone creates deep copy of LogSearch", func(t *testing.T) {
		original := &LogSearch{
			Follow: true,
			Size:   ty.Opt[int]{Value: 100, Set: true},
			Options: ty.MI{
				"container": "test-container",
				"service":   "test-service",
			},
			Fields: ty.MS{
				"field1": "value1",
				"field2": "value2",
			},
			FieldsCondition: ty.MS{
				"field1": "equals",
			},
			Variables: map[string]VariableDefinition{
				"var1": {Description: "test variable"},
			},
		}

		clone := original.Clone()

		// Verify all fields are copied
		assert.Equal(t, original.Follow, clone.Follow)
		assert.Equal(t, original.Size.Value, clone.Size.Value)
		assert.Equal(t, original.Options["container"], clone.Options["container"])
		assert.Equal(t, original.Fields["field1"], clone.Fields["field1"])
		assert.Equal(t, original.FieldsCondition["field1"], clone.FieldsCondition["field1"])
		assert.Equal(t, original.Variables["var1"].Description, clone.Variables["var1"].Description)

		// Verify deep copy by modifying clone and checking original is unchanged
		clone.Options["container"] = "modified-container"
		clone.Fields["field1"] = "modified-value"
		clone.FieldsCondition["field1"] = "modified-condition"
		clone.Variables["var1"] = VariableDefinition{Description: "modified"}

		assert.Equal(t, "test-container", original.Options["container"], "Original should be unchanged")
		assert.Equal(t, "value1", original.Fields["field1"], "Original should be unchanged")
		assert.Equal(t, "equals", original.FieldsCondition["field1"], "Original should be unchanged")
		assert.Equal(t, "test variable", original.Variables["var1"].Description, "Original should be unchanged")
	})

	t.Run("Clone handles nil LogSearch", func(t *testing.T) {
		var original *LogSearch
		clone := original.Clone()
		assert.Nil(t, clone)
	})

	t.Run("Clone handles empty maps", func(t *testing.T) {
		original := &LogSearch{
			Follow: true,
		}

		clone := original.Clone()
		assert.NotNil(t, clone)
		assert.Equal(t, original.Follow, clone.Follow)
	})

	t.Run("Clone handles Filter field", func(t *testing.T) {
		original := &LogSearch{
			Filter: &Filter{
				Field: "testField",
				Op:    "equals",
				Value: "testValue",
			},
		}

		clone := original.Clone()
		assert.NotNil(t, clone.Filter)
		assert.Equal(t, original.Filter.Field, clone.Filter.Field)
		assert.Equal(t, original.Filter.Op, clone.Filter.Op)
		assert.Equal(t, original.Filter.Value, clone.Filter.Value)

		// Verify it's a deep copy
		clone.Filter.Field = "modifiedField"
		assert.Equal(t, "testField", original.Filter.Field, "Original Filter should be unchanged")
	})

	t.Run("Clone handles nested Filter with sub-filters", func(t *testing.T) {
		original := &LogSearch{
			Filter: &Filter{
				Logic: LogicAnd,
				Filters: []Filter{
					{Field: "level", Op: "equals", Value: "ERROR"},
					{Field: "app", Op: "equals", Value: "myapp"},
				},
			},
		}

		clone := original.Clone()
		assert.NotNil(t, clone.Filter)
		assert.Equal(t, original.Filter.Logic, clone.Filter.Logic)
		assert.Len(t, clone.Filter.Filters, 2)

		// Verify deep copy of nested filters
		clone.Filter.Filters[0].Value = "modified"
		assert.Equal(t, "ERROR", original.Filter.Filters[0].Value, "Original nested Filter should be unchanged")
	})
}
