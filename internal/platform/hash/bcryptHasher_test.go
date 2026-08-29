package hash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBcryptHasher(t *testing.T) {
	h := NewBcryptHasher()
	password := "mysecretpassword"
	hashed, err := h.Hash(password)

	require.NoError(t, err)
	require.NotEqual(t, password, hashed, "must not be equal")

	require.NoError(t, h.Compare(hashed, password))
	require.Error(t, h.Compare(hashed, "wrongpassword"))

	hashedAgain, err := h.Hash(password)
	require.NoError(t, err)
	require.NotEqual(t, hashed, hashedAgain, "must not be equal")
}
