package rulesengine

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSlice_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name: "nil slice serializes as empty array",
			input: struct {
				Items JSONSlice[string] `json:"items"`
			}{Items: nil},
			expected: `{"items":[]}`,
		},
		{
			name: "empty slice serializes as empty array",
			input: struct {
				Items JSONSlice[string] `json:"items"`
			}{Items: JSONSlice[string]{}},
			expected: `{"items":[]}`,
		},
		{
			name: "populated slice serializes normally",
			input: struct {
				Items JSONSlice[string] `json:"items"`
			}{Items: JSONSlice[string]{"a", "b", "c"}},
			expected: `{"items":["a","b","c"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestJSONSlice_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected JSONSlice[string]
	}{
		{
			name:     "null unmarshals to empty slice",
			input:    `{"items":null}`,
			expected: JSONSlice[string]{},
		},
		{
			name:     "empty array unmarshals to empty slice",
			input:    `{"items":[]}`,
			expected: JSONSlice[string]{},
		},
		{
			name:     "populated array unmarshals correctly",
			input:    `{"items":["a","b","c"]}`,
			expected: JSONSlice[string]{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Items JSONSlice[string] `json:"items"`
			}
			err := json.Unmarshal([]byte(tt.input), &result)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Items)
		})
	}
}

func TestJSONSlice_PointerElements(t *testing.T) {
	type Item struct {
		ID string `json:"id"`
	}

	data, err := json.Marshal(struct {
		Items JSONSlice[*Item] `json:"items"`
	}{Items: nil})
	require.NoError(t, err)
	assert.JSONEq(t, `{"items":[]}`, string(data))
}

// Verifies that an unnamed []T literal is assignable to a JSONSlice[T] field,
// so existing call sites that construct rulesengine types via composite
// literals (Flag{Rules: []*Rule{...}}) continue to compile.
func TestJSONSlice_AssignableFromBareSliceLiteral(t *testing.T) {
	type wrapper struct {
		Items JSONSlice[string] `json:"items"`
	}

	w := wrapper{Items: []string{"a", "b"}}
	assert.Len(t, w.Items, 2)
}

// Verifies that rulesengine wire types emit `[]` instead of `null` for nil
// slice fields after the migration to JSONSlice. This is the contract the
// upstream Fern-generated SDKs depend on.
func TestRulesengineWireTypes_NilSlicesMarshalAsEmpty(t *testing.T) {
	t.Run("Flag with nil Rules", func(t *testing.T) {
		data, err := json.Marshal(&Flag{Key: "k"})
		require.NoError(t, err)
		assert.Contains(t, string(data), `"rules":[]`)
		assert.NotContains(t, string(data), `"rules":null`)
	})

	t.Run("Rule with nil Conditions and ConditionGroups", func(t *testing.T) {
		data, err := json.Marshal(&Rule{ID: "r"})
		require.NoError(t, err)
		assert.Contains(t, string(data), `"conditions":[]`)
		assert.Contains(t, string(data), `"condition_groups":[]`)
	})

	t.Run("Condition with nil ResourceIDs", func(t *testing.T) {
		data, err := json.Marshal(&Condition{ID: "c"})
		require.NoError(t, err)
		assert.Contains(t, string(data), `"resource_ids":[]`)
	})

	t.Run("ConditionGroup with nil Conditions", func(t *testing.T) {
		data, err := json.Marshal(&ConditionGroup{})
		require.NoError(t, err)
		assert.Contains(t, string(data), `"conditions":[]`)
	})

	t.Run("Company with nil slice fields", func(t *testing.T) {
		data, err := json.Marshal(&Company{ID: "co"})
		require.NoError(t, err)
		s := string(data)
		for _, key := range []string{
			`"billing_product_ids":[]`,
			`"plan_ids":[]`,
			`"plan_version_ids":[]`,
			`"rules":[]`,
			`"traits":[]`,
			`"metrics":[]`,
		} {
			assert.Contains(t, s, key)
		}
	})

	t.Run("User with nil slice fields", func(t *testing.T) {
		data, err := json.Marshal(&User{ID: "u"})
		require.NoError(t, err)
		s := string(data)
		assert.Contains(t, s, `"traits":[]`)
		assert.Contains(t, s, `"rules":[]`)
	})
}
