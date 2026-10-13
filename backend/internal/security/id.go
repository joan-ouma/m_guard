package security

import (
	"strings"

	"github.com/google/uuid"
)

// NewID returns a prefixed identifier sized for the VARCHAR(32) primary keys
// used by the users and bikes tables (for example "usr_1f2e3d4c5b6a7988...").
// A bare UUID is 36 characters and does not fit those columns.
func NewID(prefix string) string {
	return prefix + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
}
