package client_test

import (
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/operator"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func TestFilterValidate(t *testing.T) {
	t.Run("nil filter is valid", func(t *testing.T) {
		var f *client.Filter
		assert.NoError(t, f.Validate())
	})

	t.Run("valid leaf with equals", func(t *testing.T) {
		f := &client.Filter{
			Field: "level",
			Op:    operator.Equals,
			Value: "ERROR",
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("valid leaf with default op", func(t *testing.T) {
		f := &client.Filter{
			Field: "level",
			Value: "ERROR",
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("valid leaf with exists (no value needed)", func(t *testing.T) {
		f := &client.Filter{
			Field: "trace_id",
			Op:    operator.Exists,
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid - leaf missing value", func(t *testing.T) {
		f := &client.Filter{
			Field: "level",
			Op:    operator.Equals,
		}
		assert.Error(t, f.Validate())
	})

	t.Run("invalid - unknown operator", func(t *testing.T) {
		f := &client.Filter{
			Field: "level",
			Op:    "unknown_op",
			Value: "ERROR",
		}
		assert.Error(t, f.Validate())
	})

	t.Run("valid AND branch", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicAnd,
			Filters: []client.Filter{
				{Field: "level", Value: "ERROR"},
				{Field: "app", Value: "myapp"},
			},
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("valid OR branch", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicOr,
			Filters: []client.Filter{
				{Field: "level", Value: "ERROR"},
				{Field: "level", Value: "WARN"},
			},
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("valid NOT branch", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicNot,
			Filters: []client.Filter{
				{Field: "level", Value: "DEBUG"},
			},
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid - NOT with no children", func(t *testing.T) {
		f := &client.Filter{
			Logic:   client.LogicNot,
			Filters: []client.Filter{},
		}
		assert.Error(t, f.Validate())
	})

	t.Run("invalid - both field and logic set", func(t *testing.T) {
		f := &client.Filter{
			Field: "level",
			Value: "ERROR",
			Logic: client.LogicAnd,
		}
		assert.Error(t, f.Validate())
	})

	t.Run("empty filter is valid (matches all)", func(t *testing.T) {
		f := &client.Filter{}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid - leaf with nested filters", func(t *testing.T) {
		f := &client.Filter{
			Field: "level",
			Value: "ERROR",
			Filters: []client.Filter{
				{Field: "app", Value: "myapp"},
			},
		}
		assert.Error(t, f.Validate())
	})

	t.Run("invalid - branch with value", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicAnd,
			Value: "shouldnt be here",
			Filters: []client.Filter{
				{Field: "level", Value: "ERROR"},
			},
		}
		assert.Error(t, f.Validate())
	})

	t.Run("valid nested structure", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicAnd,
			Filters: []client.Filter{
				{Field: "app", Value: "myapp"},
				{
					Logic: client.LogicOr,
					Filters: []client.Filter{
						{Field: "level", Value: "ERROR"},
						{Field: "level", Value: "WARN"},
					},
				},
			},
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid nested structure - child invalid", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicAnd,
			Filters: []client.Filter{
				{Field: "app", Value: "myapp"},
				{Field: "level"}, // missing value
			},
		}
		assert.Error(t, f.Validate())
	})
}

func TestFilterMatch(t *testing.T) {
	entry := client.LogEntry{
		Message: "An error occurred in the application",
		Level:   "ERROR",
		Fields: ty.MI{
			"app":      "myapp",
			"env":      "production",
			"status":   "500",
			"trace_id": "abc123",
		},
	}

	t.Run("nil filter matches everything", func(t *testing.T) {
		var f *client.Filter
		assert.True(t, f.Match(entry))
	})

	t.Run("empty filter matches everything", func(t *testing.T) {
		f := &client.Filter{}
		assert.True(t, f.Match(entry))
	})

	t.Run("equals - match", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Equals, Value: "myapp"}
		assert.True(t, f.Match(entry))
	})

	t.Run("equals - no match", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Equals, Value: "otherapp"}
		assert.False(t, f.Match(entry))
	})

	t.Run("equals - field not found", func(t *testing.T) {
		f := &client.Filter{Field: "nonexistent", Op: operator.Equals, Value: "value"}
		assert.False(t, f.Match(entry))
	})

	t.Run("regex - match", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Regex, Value: "my.*"}
		assert.True(t, f.Match(entry))
	})

	t.Run("regex - no match", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Regex, Value: "^other.*"}
		assert.False(t, f.Match(entry))
	})

	t.Run("wildcard - match", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Wildcard, Value: "my*"}
		assert.True(t, f.Match(entry))
	})

	t.Run("wildcard - no match", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Wildcard, Value: "other*"}
		assert.False(t, f.Match(entry))
	})

	t.Run("match (contains) - match", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Match, Value: "yap"}
		assert.True(t, f.Match(entry))
	})

	t.Run("match (contains) - case insensitive", func(t *testing.T) {
		f := &client.Filter{Field: "app", Op: operator.Match, Value: "MYAPP"}
		assert.True(t, f.Match(entry))
	})

	t.Run("exists - field exists", func(t *testing.T) {
		f := &client.Filter{Field: "trace_id", Op: operator.Exists}
		assert.True(t, f.Match(entry))
	})

	t.Run("exists - field does not exist", func(t *testing.T) {
		f := &client.Filter{Field: "nonexistent", Op: operator.Exists}
		assert.False(t, f.Match(entry))
	})

	t.Run("_ sentinel - searches message", func(t *testing.T) {
		f := &client.Filter{Field: "_", Op: operator.Match, Value: "error"}
		assert.True(t, f.Match(entry))
	})

	t.Run("_ sentinel - regex on message", func(t *testing.T) {
		f := &client.Filter{Field: "_", Op: operator.Regex, Value: ".*error.*application.*"}
		assert.True(t, f.Match(entry))
	})

	t.Run("level field access via LogEntry.Field", func(t *testing.T) {
		f := &client.Filter{Field: "level", Op: operator.Equals, Value: "ERROR"}
		assert.True(t, f.Match(entry))
	})

	t.Run("AND - all match", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicAnd,
			Filters: []client.Filter{
				{Field: "app", Value: "myapp"},
				{Field: "env", Value: "production"},
			},
		}
		assert.True(t, f.Match(entry))
	})

	t.Run("AND - one doesn't match", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicAnd,
			Filters: []client.Filter{
				{Field: "app", Value: "myapp"},
				{Field: "env", Value: "staging"},
			},
		}
		assert.False(t, f.Match(entry))
	})

	t.Run("OR - one matches", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicOr,
			Filters: []client.Filter{
				{Field: "env", Value: "staging"},
				{Field: "env", Value: "production"},
			},
		}
		assert.True(t, f.Match(entry))
	})

	t.Run("OR - none match", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicOr,
			Filters: []client.Filter{
				{Field: "env", Value: "staging"},
				{Field: "env", Value: "development"},
			},
		}
		assert.False(t, f.Match(entry))
	})

	t.Run("NOT - inverts match", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicNot,
			Filters: []client.Filter{
				{Field: "level", Value: "DEBUG"},
			},
		}
		assert.True(t, f.Match(entry)) // Entry level is ERROR, NOT DEBUG = true
	})

	t.Run("NOT - inverts non-match", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicNot,
			Filters: []client.Filter{
				{Field: "level", Value: "ERROR"},
			},
		}
		assert.False(t, f.Match(entry)) // Entry level is ERROR, NOT ERROR = false
	})

	t.Run("complex nested: (A OR B) AND C", func(t *testing.T) {
		f := &client.Filter{
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
		}
		assert.True(t, f.Match(entry))
	})

	t.Run("complex nested: (A OR B) AND C - fails", func(t *testing.T) {
		f := &client.Filter{
			Logic: client.LogicAnd,
			Filters: []client.Filter{
				{
					Logic: client.LogicOr,
					Filters: []client.Filter{
						{Field: "level", Value: "INFO"},
						{Field: "level", Value: "DEBUG"},
					},
				},
				{Field: "app", Value: "myapp"},
			},
		}
		assert.False(t, f.Match(entry))
	})

	t.Run("empty AND group matches", func(t *testing.T) {
		f := &client.Filter{
			Logic:   client.LogicAnd,
			Filters: []client.Filter{},
		}
		assert.True(t, f.Match(entry))
	})

	t.Run("empty OR group matches", func(t *testing.T) {
		f := &client.Filter{
			Logic:   client.LogicOr,
			Filters: []client.Filter{},
		}
		assert.True(t, f.Match(entry)) // Empty group returns true
	})
}

func TestGetEffectiveFilter(t *testing.T) {
	t.Run("empty search returns nil filter", func(t *testing.T) {
		s := &client.LogSearch{}
		assert.Nil(t, s.GetEffectiveFilter())
	})

	t.Run("legacy fields only", func(t *testing.T) {
		s := &client.LogSearch{
			Fields: ty.MS{"level": "ERROR", "app": "myapp"},
		}
		f := s.GetEffectiveFilter()
		assert.NotNil(t, f)
		assert.Equal(t, client.LogicAnd, f.Logic)
		assert.Len(t, f.Filters, 2)
	})

	t.Run("legacy fields with conditions", func(t *testing.T) {
		s := &client.LogSearch{
			Fields:          ty.MS{"message": "error.*"},
			FieldsCondition: ty.MS{"message": operator.Regex},
		}
		f := s.GetEffectiveFilter()
		assert.NotNil(t, f)
		assert.Equal(t, "message", f.Field)
		assert.Equal(t, operator.Regex, f.Op)
	})

	t.Run("new filter only", func(t *testing.T) {
		s := &client.LogSearch{
			Filter: &client.Filter{
				Logic: client.LogicOr,
				Filters: []client.Filter{
					{Field: "level", Value: "ERROR"},
					{Field: "level", Value: "WARN"},
				},
			},
		}
		f := s.GetEffectiveFilter()
		assert.NotNil(t, f)
		assert.Equal(t, client.LogicOr, f.Logic)
	})

	t.Run("combined legacy and new filter", func(t *testing.T) {
		s := &client.LogSearch{
			Fields: ty.MS{"app": "myapp"},
			Filter: &client.Filter{
				Logic: client.LogicOr,
				Filters: []client.Filter{
					{Field: "level", Value: "ERROR"},
					{Field: "level", Value: "WARN"},
				},
			},
		}
		f := s.GetEffectiveFilter()
		assert.NotNil(t, f)
		assert.Equal(t, client.LogicAnd, f.Logic)
		assert.Len(t, f.Filters, 2) // legacy + new filter
	})

	t.Run("single legacy field returns leaf directly", func(t *testing.T) {
		s := &client.LogSearch{
			Fields: ty.MS{"level": "ERROR"},
		}
		f := s.GetEffectiveFilter()
		assert.NotNil(t, f)
		assert.Equal(t, "level", f.Field)
		assert.Equal(t, "ERROR", f.Value)
	})
}

func TestMergeIntoWithFilter(t *testing.T) {
	t.Run("merge filter into empty", func(t *testing.T) {
		parent := &client.LogSearch{}
		child := &client.LogSearch{
			Filter: &client.Filter{Field: "level", Value: "ERROR"},
		}
		_ = parent.MergeInto(child)
		assert.NotNil(t, parent.Filter)
		assert.Equal(t, "level", parent.Filter.Field)
	})

	t.Run("merge filter into existing - creates AND", func(t *testing.T) {
		parent := &client.LogSearch{
			Filter: &client.Filter{Field: "app", Value: "myapp"},
		}
		child := &client.LogSearch{
			Filter: &client.Filter{Field: "level", Value: "ERROR"},
		}
		_ = parent.MergeInto(child)
		assert.NotNil(t, parent.Filter)
		assert.Equal(t, client.LogicAnd, parent.Filter.Logic)
		assert.Len(t, parent.Filter.Filters, 2)
	})

	t.Run("merge nil filter doesn't affect existing", func(t *testing.T) {
		parent := &client.LogSearch{
			Filter: &client.Filter{Field: "app", Value: "myapp"},
		}
		child := &client.LogSearch{}
		_ = parent.MergeInto(child)
		assert.NotNil(t, parent.Filter)
		assert.Equal(t, "app", parent.Filter.Field)
	})
}
