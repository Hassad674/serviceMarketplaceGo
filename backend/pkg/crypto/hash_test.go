package crypto

import (
	"testing"

	"marketplace-backend/internal/domain/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBcryptHasher_Hash_ProducesNonEmptyString(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("MyPassword1")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestBcryptHasher_Hash_DifferentPasswordsDifferentHashes(t *testing.T) {
	hasher := NewBcryptHasher()

	hash1, err := hasher.Hash("PasswordOne1")
	require.NoError(t, err)

	hash2, err := hasher.Hash("PasswordTwo2")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
}

func TestBcryptHasher_Hash_SamePasswordDifferentHashes(t *testing.T) {
	hasher := NewBcryptHasher()
	password := "SamePassword1"

	hash1, err := hasher.Hash(password)
	require.NoError(t, err)

	hash2, err := hasher.Hash(password)
	require.NoError(t, err)

	// bcrypt produces different hashes due to random salt
	assert.NotEqual(t, hash1, hash2)
}

func TestBcryptHasher_Compare_CorrectPassword(t *testing.T) {
	hasher := NewBcryptHasher()
	password := "CorrectPassword1"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	err = hasher.Compare(hash, password)
	assert.NoError(t, err)
}

func TestBcryptHasher_Compare_WrongPassword(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("OriginalPassword1")
	require.NoError(t, err)

	err = hasher.Compare(hash, "WrongPassword1")
	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
}

func TestBcryptHasher_Compare_EmptyPassword(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("SomePassword1")
	require.NoError(t, err)

	err = hasher.Compare(hash, "")
	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
}

func TestBcryptHasher_Hash_UsesCost12(t *testing.T) {
	// bcrypt cost 12 produces hashes starting with "$2a$12$"
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("TestPassword1")
	require.NoError(t, err)

	// bcrypt hashes with cost 12 contain "$12$" in the prefix
	assert.Contains(t, hash, "$12$", "hash should use bcrypt cost 12")
}

func TestBcryptHasher_Hash_ProducesValidBcryptFormat(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("TestPassword1")
	require.NoError(t, err)

	// bcrypt hashes are 60 characters long
	assert.Len(t, hash, 60)

	// bcrypt hashes start with "$2a$" or "$2b$"
	assert.True(t,
		hash[:4] == "$2a$" || hash[:4] == "$2b$",
		"hash should start with $2a$ or $2b$, got: %s", hash[:4],
	)
}

// bcrypt's GenerateFromPassword rejects inputs longer than 72 bytes —
// the err != nil branch in Hash. We use a 73-byte password to surface
// it. This is the documented bcrypt limit and the only easy way to
// reach the error path without monkey-patching the standard library.
func TestBcryptHasher_Hash_PasswordTooLong_Surfaces(t *testing.T) {
	hasher := NewBcryptHasher()

	tooLong := make([]byte, 73) // 73 > 72-byte bcrypt limit
	for i := range tooLong {
		tooLong[i] = 'a'
	}

	_, err := hasher.Hash(string(tooLong))
	require.Error(t, err, "bcrypt MUST reject inputs longer than 72 bytes — the error MUST surface, not be swallowed")
}

// Compare must reject an unparseable hash (corrupt DB row, mismatched
// hash format) with the user-facing ErrInvalidCredentials so callers
// don't leak crypto internals upstream.
func TestBcryptHasher_Compare_CorruptHash_ReturnsCanonicalError(t *testing.T) {
	hasher := NewBcryptHasher()

	err := hasher.Compare("not-a-bcrypt-hash", "any-password")
	assert.ErrorIs(t, err, user.ErrInvalidCredentials,
		"corrupt hash MUST surface as ErrInvalidCredentials, not as a bcrypt-specific error")
}
