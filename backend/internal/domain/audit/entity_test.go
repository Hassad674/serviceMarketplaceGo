package audit

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewEntry_MinimalInput verifies the happy path with only the
// mandatory Action field populated.
func TestNewEntry_MinimalInput(t *testing.T) {
	entry, err := NewEntry(NewEntryInput{
		Action: ActionLoginSuccess,
	})
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.Equal(t, ActionLoginSuccess, entry.Action)
	assert.NotNil(t, entry.Metadata) // never nil — empty map
	assert.False(t, entry.CreatedAt.IsZero())
}

// TestNewEntry_ActionRequired verifies the action validation rule.
func TestNewEntry_ActionRequired(t *testing.T) {
	_, err := NewEntry(NewEntryInput{})
	assert.ErrorIs(t, err, ErrActionRequired)

	_, err = NewEntry(NewEntryInput{Action: "   "})
	assert.ErrorIs(t, err, ErrActionRequired)
}

// TestNewEntry_InvalidIPIsIgnored verifies that a malformed IP string
// does NOT fail entry creation — audit events are too important to
// drop because of a non-critical field.
func TestNewEntry_InvalidIPIsIgnored(t *testing.T) {
	entry, err := NewEntry(NewEntryInput{
		Action:    ActionLoginSuccess,
		IPAddress: "not-an-ip",
	})
	assert.NoError(t, err)
	assert.Nil(t, entry.IPAddress)
}

// TestNewEntry_ValidIPIsParsed verifies that a well-formed IP string
// is parsed into a net.IP.
func TestNewEntry_ValidIPIsParsed(t *testing.T) {
	entry, err := NewEntry(NewEntryInput{
		Action:    ActionLoginSuccess,
		IPAddress: "192.168.1.42",
	})
	assert.NoError(t, err)
	assert.NotNil(t, entry.IPAddress)
	assert.Equal(t, "192.168.1.42", entry.IPAddress.String())
}

// TestNewEntry_MetadataNeverNil guarantees the invariant downstream
// code relies on: metadata is always a valid map.
func TestNewEntry_MetadataNeverNil(t *testing.T) {
	entry, _ := NewEntry(NewEntryInput{Action: ActionLoginSuccess})
	assert.NotNil(t, entry.Metadata)
	assert.Empty(t, entry.Metadata)

	entry2, _ := NewEntry(NewEntryInput{
		Action:   ActionLoginSuccess,
		Metadata: map[string]any{"foo": "bar"},
	})
	assert.Equal(t, "bar", entry2.Metadata["foo"])
}
