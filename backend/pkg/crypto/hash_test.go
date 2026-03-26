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
