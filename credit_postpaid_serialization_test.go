package rulesengine_test

import (
	"encoding/json"
	"testing"

	"github.com/schematichq/rulesengine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Presence of a credit's key is the postpaid opt-in, and null never means
// anything on its own: a null map is an empty map, a null config is {}, and a
// null limit is no limit. An earlier shape used a nullable number as the map
// value, and SDKs that strip null map values read "unbounded" as "off".
//
// Mirrors credit_postpaid_serialization_test.rs in rulesengine-rust.
func TestCreditPostpaidSerialization(t *testing.T) {
	const creditID = "test-credit-id"

	t.Run("decodes each encoding", func(t *testing.T) {
		hundred := 100.0
		for _, tc := range []struct {
			name       string
			raw        string
			postpaidOn bool
			limit      *float64
		}{
			{name: "field absent", raw: `{}`},
			{name: "field null", raw: `{"credit_postpaid":null}`},
			{name: "credit absent", raw: `{"credit_postpaid":{}}`},
			{name: "empty config is on and unbounded", raw: `{"credit_postpaid":{"test-credit-id":{}}}`, postpaidOn: true},
			{name: "null limit is on and unbounded", raw: `{"credit_postpaid":{"test-credit-id":{"overdraft_limit":null}}}`, postpaidOn: true},
			{name: "null config is on and unbounded", raw: `{"credit_postpaid":{"test-credit-id":null}}`, postpaidOn: true},
			{name: "numeric limit is on and capped", raw: `{"credit_postpaid":{"test-credit-id":{"overdraft_limit":100}}}`, postpaidOn: true, limit: &hundred},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var company rulesengine.Company
				require.NoError(t, json.Unmarshal([]byte(tc.raw), &company))

				postpaid, postpaidOn := company.CreditPostpaid[creditID]
				require.Equal(t, tc.postpaidOn, postpaidOn, "presence in the map is the opt-in")
				assert.Equal(t, tc.limit, postpaid.OverdraftLimit)
			})
		}
	})

	// Round-tripping must not turn "no limit" into a zero limit, which would
	// deny every draw past the balance.
	t.Run("round-trips", func(t *testing.T) {
		limit := 100.0
		for name, postpaid := range map[string]map[string]rulesengine.CreditPostpaidConfig{
			"unbounded": {creditID: {}},
			"capped":    {creditID: {OverdraftLimit: &limit}},
			"empty":     {},
			"nil map":   nil,
		} {
			t.Run(name, func(t *testing.T) {
				encoded, err := json.Marshal(rulesengine.Company{CreditPostpaid: postpaid})
				require.NoError(t, err)

				var decoded rulesengine.Company
				require.NoError(t, json.Unmarshal(encoded, &decoded))

				assert.Len(t, decoded.CreditPostpaid, len(postpaid))

				want, wantOn := postpaid[creditID]
				got, gotOn := decoded.CreditPostpaid[creditID]
				require.Equal(t, wantOn, gotOn)
				assert.Equal(t, want, got)
			})
		}
	})
}

// omitempty is load-bearing rather than cosmetic. A client generated from a spec
// that predates this field marks every property it knows as required and decodes
// strictly; if an empty map serialised as "credit_postpaid":{} the key would be
// an unknown property to that client. A company with no postpaid grant is the
// common case, so this is the shape most payloads take.
func TestCreditPostpaidWireShape(t *testing.T) {
	for name, postpaid := range map[string]map[string]rulesengine.CreditPostpaidConfig{
		"nil map":   nil,
		"empty map": {},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(rulesengine.Company{CreditPostpaid: postpaid})
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), "credit_postpaid",
				"an empty map must not put the key on the wire")
		})
	}

	t.Run("an unbounded credit sends an empty config", func(t *testing.T) {
		encoded, err := json.Marshal(rulesengine.Company{
			CreditPostpaid: map[string]rulesengine.CreditPostpaidConfig{"c": {}},
		})
		require.NoError(t, err)
		assert.Contains(t, string(encoded), `"credit_postpaid":{"c":{}}`)
	})

	t.Run("a capped credit sends its limit", func(t *testing.T) {
		limit := 100.0
		encoded, err := json.Marshal(rulesengine.Company{
			CreditPostpaid: map[string]rulesengine.CreditPostpaidConfig{"c": {OverdraftLimit: &limit}},
		})
		require.NoError(t, err)
		assert.Contains(t, string(encoded), `"credit_postpaid":{"c":{"overdraft_limit":100}}`)
	})
}
