package security

import (
	"strings"
	"testing"
)

// TestNewIDFitsVarchar32 guards the users/bikes primary keys, which are
// declared VARCHAR(32). A plain UUID (36 chars) overflows them and makes
// registration and bike creation fail with a 500.
func TestNewIDFitsVarchar32(t *testing.T) {
	for _, prefix := range []string{"usr_", "bik_"} {
		for i := 0; i < 500; i++ {
			id := NewID(prefix)
			if len(id) > 32 {
				t.Fatalf("NewID(%q) produced %d chars, exceeds VARCHAR(32): %q", prefix, len(id), id)
			}
			if !strings.HasPrefix(id, prefix) {
				t.Fatalf("NewID(%q) is missing the prefix: %q", prefix, id)
			}
		}
	}
}

func TestNewIDIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := NewID("usr_")
		if _, dup := seen[id]; dup {
			t.Fatalf("NewID returned a duplicate identifier: %q", id)
		}
		seen[id] = struct{}{}
	}
}
