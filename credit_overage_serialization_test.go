package rulesengine_test

import (
	"encoding/json"
	"testing"

	"github.com/schematichq/rulesengine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The overage map leans on a distinction JSON can express but is easy to lose in
// a hand-written client: a key present with a null value (overage on, uncapped)
// against the key being absent (overage off). Collapsing those turns "gate at
// zero" into "never gate", which is why it is asserted rather than assumed.
func TestCreditOverageSerialization(t *testing.T) {
	const creditID = "test-credit-id"

	t.Run("decodes the three states distinctly", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			raw       string
			overageOn bool
			uncapped  bool
			cap       float64
		}{
			{name: "field absent", raw: `{}`},
			{name: "field null", raw: `{"credit_overage":null}`},
			{name: "map empty", raw: `{"credit_overage":{}}`},
			{name: "value null is on and uncapped", raw: `{"credit_overage":{"test-credit-id":null}}`, overageOn: true, uncapped: true},
			{name: "value set is on and capped", raw: `{"credit_overage":{"test-credit-id":100}}`, overageOn: true, cap: 100},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var company rulesengine.Company
				require.NoError(t, json.Unmarshal([]byte(tc.raw), &company))

				overageCap, overageOn := company.CreditOverage[creditID]
				require.Equal(t, tc.overageOn, overageOn, "presence in the map is the opt-in")

				if !tc.overageOn {
					return
				}

				if tc.uncapped {
					assert.Nil(t, overageCap, "a null value must stay nil, not become zero")
					return
				}

				require.NotNil(t, overageCap)
				assert.Equal(t, tc.cap, *overageCap)
			})
		}
	})

	// Round-tripping must not quietly promote nil to 0. A zero cap denies every
	// draw past the balance, which is the opposite of what nil means.
	t.Run("round-trips without collapsing nil to zero", func(t *testing.T) {
		limit := float64(100)
		for name, overage := range map[string]map[string]*float64{
			"uncapped": {creditID: nil},
			"capped":   {creditID: &limit},
			"empty":    {},
			"nil map":  nil,
		} {
			t.Run(name, func(t *testing.T) {
				encoded, err := json.Marshal(rulesengine.Company{CreditOverage: overage})
				require.NoError(t, err)

				var decoded rulesengine.Company
				require.NoError(t, json.Unmarshal(encoded, &decoded))

				assert.Len(t, decoded.CreditOverage, len(overage))

				want, wantOn := overage[creditID]
				got, gotOn := decoded.CreditOverage[creditID]
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
