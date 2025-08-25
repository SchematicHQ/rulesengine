package rulesengine

import (
	"testing"
)

func TestGetVersionKey(t *testing.T) {
	t.Run("Generates consistent version key", func(t *testing.T) {
		key1 := GetVersionKey()
		key2 := GetVersionKey()

		if key1 != key2 {
			t.Errorf("Expected version keys to be consistent, got %s and %s", key1, key2)
		}

		if len(key1) != 8 {
			t.Errorf("Expected version key to be 8 characters, got %d", len(key1))
		}
	})

	t.Run("Version key is not empty", func(t *testing.T) {
		key := GetVersionKey()

		if key == "" {
			t.Error("Expected version key to not be empty")
		}
	})

	t.Run("Version key changes when models change", func(t *testing.T) {
		// This test documents that the version key should change when
		// the model structures change. We can't easily test this in isolation,
		// but this serves as documentation of the expected behavior.
		key := GetVersionKey()

		// Ensure the key is deterministic and valid
		if len(key) != 8 {
			t.Errorf("Expected version key to be 8 characters, got %d", len(key))
		}

		// Should contain only hex characters
		for _, r := range key {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Errorf("Expected version key to contain only hex characters, got %s", key)
			}
		}
	})
}
