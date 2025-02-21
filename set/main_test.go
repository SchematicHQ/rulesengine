package set_test

import (
	"testing"

	"github.com/schematichq/rulesengine/set"
	"github.com/stretchr/testify/assert"
)

func TestLen(t *testing.T) {
	t.Run("Set length operations", func(t *testing.T) {
		// Initialize
		s := set.NewSet[int]()
		assert.Equal(t, 0, s.Len())
		assert.Equal(t, 0, len(s))

		// Add element
		s.Add(1)
		assert.Equal(t, 1, s.Len())
		assert.Equal(t, 1, len(s))

		// Remove element
		s.Remove(1)
		assert.Equal(t, 0, s.Len())
		assert.Equal(t, 0, len(s))
	})
}
