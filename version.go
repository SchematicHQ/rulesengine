package rulesengine

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"reflect"
)

// generateVersionKey creates a version key based on the structure
// of the rules engine types that are cached. This ensures cache invalidation
// when the model structures change.
//
// This approach provides automatic versioning by:
// 1. Examining the struct layout of cached types (Company, User, Flag, CheckFlagResult)
// 2. Hashing field names, types, and tags to create a deterministic fingerprint
// 3. Generating an 8-character hex string as the version key
//
// When any of these types change (fields added/removed/modified), the hash changes,
// automatically invalidating cached data without manual intervention.
//
// Alternative approaches considered:
// - Git commit hash: Requires git and may change for non-model commits
// - Semantic versioning: Requires manual maintenance
// - Build time: Changes too frequently, not specific to model changes
// - File hash: Less precise, includes comments and formatting changes
func generateVersionKey() string {
	hasher := sha256.New()

	// Hash the structure of the types we cache
	addTypeToHash(hasher, reflect.TypeOf((*Company)(nil)).Elem())
	addTypeToHash(hasher, reflect.TypeOf((*User)(nil)).Elem())
	addTypeToHash(hasher, reflect.TypeOf((*Flag)(nil)).Elem())
	addTypeToHash(hasher, reflect.TypeOf((*CheckFlagResult)(nil)).Elem())

	// Get first 8 characters of the hash (similar to current format)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	return hash[:8]
}

// addTypeToHash recursively adds type information to the hash
func addTypeToHash(hasher hash.Hash, t reflect.Type) {
	if t == nil {
		return
	}

	// Write the type name and kind
	hasher.Write([]byte(t.Name()))
	hasher.Write([]byte(t.Kind().String()))

	switch t.Kind() {
	case reflect.Struct:
		// Hash all field names and types
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			hasher.Write([]byte(field.Name))
			hasher.Write([]byte(field.Tag))
			addTypeToHash(hasher, field.Type)
		}
	case reflect.Slice, reflect.Array, reflect.Ptr:
		addTypeToHash(hasher, t.Elem())
	case reflect.Map:
		addTypeToHash(hasher, t.Key())
		addTypeToHash(hasher, t.Elem())
	}
}

// GetVersionKey returns the version key for the current rules engine models
func GetVersionKey() string {
	return generateVersionKey()
}
