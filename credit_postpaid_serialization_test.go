package rulesengine_test

import (
	"encoding/json"
	"testing"

	"github.com/schematichq/rulesengine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The postpaid map leans on a distinction JSON can express but is easy to lose in
// a hand-written client: a key present with a null value (postpaid on, unbounded)
// against the key being absent (postpaid off). Collapsing those turns "gate at
// zero" into "never gate", which is why it is asserted rather than assumed.
func TestCreditPostpaidLimitSerialization(t *testing.T) {
	const creditID = "test-credit-id"

	t.Run("decodes the three states distinctly", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			raw        string
			postpaidOn bool
			unbounded  bool
			limit      float64
		}{
			{name: "field absent", raw: `{}`},
			{name: "field null", raw: `{"credit_postpaid_limit":null}`},
			{name: "map empty", raw: `{"credit_postpaid_limit":{}}`},
			{name: "value null is on and unbounded", raw: `{"credit_postpaid_limit":{"test-credit-id":null}}`, postpaidOn: true, unbounded: true},
			{name: "value set is on and capped", raw: `{"credit_postpaid_limit":{"test-credit-id":100}}`, postpaidOn: true, limit: 100},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var company rulesengine.Company
				require.NoError(t, json.Unmarshal([]byte(tc.raw), &company))

				overdraftLimit, postpaidOn := company.CreditPostpaidLimit[creditID]
				require.Equal(t, tc.postpaidOn, postpaidOn, "presence in the map is the opt-in")

				if !tc.postpaidOn {
					return
				}

				if tc.unbounded {
					assert.Nil(t, overdraftLimit, "a null value must stay nil, not become zero")
					return
				}

				require.NotNil(t, overdraftLimit)
				assert.Equal(t, tc.limit, *overdraftLimit)
			})
		}
	})

	// Round-tripping must not quietly promote nil to 0. A zero cap denies every
	// draw past the balance, which is the opposite of what nil means.
	t.Run("round-trips without collapsing nil to zero", func(t *testing.T) {
		limit := float64(100)
		for name, postpaid := range map[string]map[string]*float64{
			"unbounded": {creditID: nil},
			"capped":    {creditID: &limit},
			"empty":     {},
			"nil map":   nil,
		} {
			t.Run(name, func(t *testing.T) {
				encoded, err := json.Marshal(rulesengine.Company{CreditPostpaidLimit: postpaid})
				require.NoError(t, err)

				var decoded rulesengine.Company
				require.NoError(t, json.Unmarshal(encoded, &decoded))

				assert.Len(t, decoded.CreditPostpaidLimit, len(postpaid))

				want, wantOn := postpaid[creditID]
				got, gotOn := decoded.CreditPostpaidLimit[creditID]
				require.Equal(t, wantOn, gotOn)

				if !wantOn {
					return
				}

				if want == nil {
					assert.Nil(t, got, "nil cap must not decode as zero")
					return
				}

				require.NotNil(t, got)
				assert.Equal(t, *want, *got)
			})
		}
	})
}

// omitempty is load-bearing rather than cosmetic. A client generated from a spec
// that predates this field marks every property it knows as required and decodes
// strictly; if an empty map serialised as "credit_postpaid_limit":{} the key
// would be an unknown property to that client, and if the field were required a
// payload without it would fail validation. Omitting it when empty keeps the
// payload legal for both, and a company with no postpaid grant is the common
// case, so this is the shape most payloads take.
func TestCreditPostpaidLimitOmittedWhenEmpty(t *testing.T) {
	for name, postpaid := range map[string]map[string]*float64{
		"nil map":   nil,
		"empty map": {},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(rulesengine.Company{CreditPostpaidLimit: postpaid})
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), "credit_postpaid_limit",
				"an empty map must not put the key on the wire")
		})
	}

	t.Run("a populated map is still sent", func(t *testing.T) {
		limit := float64(100)
		encoded, err := json.Marshal(rulesengine.Company{
			CreditPostpaidLimit: map[string]*float64{"c": &limit},
		})
		require.NoError(t, err)
		assert.Contains(t, string(encoded), `"credit_postpaid_limit":{"c":100}`)
	})
}
