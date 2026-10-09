package rulesengine_test

import (
	"encoding/json"
	"testing"

	"github.com/schematichq/rulesengine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// quantity_rates on a credit condition and a company entitlement has to
// survive a decode and re-encode: the replicator caches these types for the
// SDKs, whose WASM engine prices event_quantities from the rates.
func TestQuantityRatesRoundTrip(t *testing.T) {
	t.Run("condition", func(t *testing.T) {
		const raw = `{"id":"cond-1","condition_type":"credit","operator":"lt","credit_id":"credit-abc",` +
			`"consumption_rate":0.5,"quantity_rates":{"input_tokens":0.001,"output_tokens":0.01}}`

		var condition rulesengine.Condition
		require.NoError(t, json.Unmarshal([]byte(raw), &condition))
		assert.Equal(t, map[string]float64{"input_tokens": 0.001, "output_tokens": 0.01}, condition.QuantityRates)

		out, err := json.Marshal(&condition)
		require.NoError(t, err)
		assert.Contains(t, string(out), `"quantity_rates":{"input_tokens":0.001,"output_tokens":0.01}`)
	})

	t.Run("company entitlement", func(t *testing.T) {
		const raw = `{"id":"comp-1","entitlements":[{"feature_id":"feat-1","feature_key":"chat",` +
			`"value_type":"credit","quantity_rates":{"input_tokens":0.001}}]}`

		var company rulesengine.Company
		require.NoError(t, json.Unmarshal([]byte(raw), &company))
		require.Len(t, company.Entitlements, 1)
		assert.Equal(t, map[string]float64{"input_tokens": 0.001}, company.Entitlements[0].QuantityRates)

		out, err := json.Marshal(&company)
		require.NoError(t, err)
		assert.Contains(t, string(out), `"quantity_rates":{"input_tokens":0.001}`)
	})

	// An empty schedule is omitted, as the API sends it.
	t.Run("empty is omitted", func(t *testing.T) {
		out, err := json.Marshal(&rulesengine.Condition{ID: "cond-2"})
		require.NoError(t, err)
		assert.NotContains(t, string(out), "quantity_rates")

		out, err = json.Marshal(&rulesengine.FeatureEntitlement{FeatureID: "feat-1"})
		require.NoError(t, err)
		assert.NotContains(t, string(out), "quantity_rates")
	})
}
