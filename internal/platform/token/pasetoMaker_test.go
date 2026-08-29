package token

import (
	"strings"
	"testing"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/joekings2k/staypass-be/internal/platform/utils"
)

// randomSymmetricKey returns a key of exactly chacha20poly1305.KeySize bytes.
// RandomString hex-encodes n random bytes, so it yields 2n characters.
func randomSymmetricKey(t *testing.T) string {
	t.Helper()
	key := utils.RandomString(chacha20poly1305.KeySize / 2)
	require.Len(t, key, chacha20poly1305.KeySize, "test key must match the cipher key size")
	return key
}

func TestNewPasetoMakerRejectsWrongKeySize(t *testing.T) {
	testCases := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "too short", key: strings.Repeat("x", chacha20poly1305.KeySize-1)},
		{name: "too long", key: strings.Repeat("x", chacha20poly1305.KeySize+1)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker, err := NewPasetoMaker(tc.key)
			require.ErrorIs(t, err, ErrInvalidKeySize)
			require.Nil(t, maker)
		})
	}
}

func TestPasetoMakerRoundTrip(t *testing.T) {
	maker, err := NewPasetoMaker(randomSymmetricKey(t))
	require.NoError(t, err)
	require.NotNil(t, maker)

	userID := uuid.New()
	duration := time.Minute

	issuedAt := time.Now()
	expiresAt := issuedAt.Add(duration)

	tokenStr, created, err := maker.CreateToken(userID, duration)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	verified, err := maker.VerifyToken(tokenStr)
	require.NoError(t, err)
	require.NotNil(t, verified)

	require.Equal(t, userID, verified.UserID)

	require.NotEqual(t, uuid.Nil, verified.ID)
	require.WithinDuration(t, issuedAt, verified.IssuedAt, time.Second)
	require.WithinDuration(t, expiresAt, verified.ExpiresAt, time.Second)

	// what CreateToken reported must be what VerifyToken reads back
	require.Equal(t, created.ID, verified.ID)
	require.Equal(t, created.UserID, verified.UserID)
	require.WithinDuration(t, created.ExpiresAt, verified.ExpiresAt, time.Second)
}

func TestPasetoMakerExpiredToken(t *testing.T) {
	maker, err := NewPasetoMaker(randomSymmetricKey(t))
	require.NoError(t, err)

	tokenStr, _, err := maker.CreateToken(uuid.New(), -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	payload, err := maker.VerifyToken(tokenStr)
	require.ErrorIs(t, err, ErrExpiredToken)
	require.Nil(t, payload)
}

func TestPasetoMakerTamperedToken(t *testing.T) {
	maker, err := NewPasetoMaker(randomSymmetricKey(t))
	require.NoError(t, err)

	tokenStr, _, err := maker.CreateToken(uuid.New(), time.Minute)
	require.NoError(t, err)

	// Flip one character inside the encrypted body. Both 'a' and 'b' are valid
	// base64url characters, so this stays well-formed and it is the
	// Poly1305 auth tag - not a decode error - that rejects it.
	body := []byte(tokenStr)
	mid := len(body) / 2
	require.Greater(t, mid, len("v2.local."), "mutation must land inside the payload, not the header")
	if body[mid] == 'a' {
		body[mid] = 'b'
	} else {
		body[mid] = 'a'
	}

	testCases := []struct {
		name  string
		token string
	}{
		{name: "flipped byte", token: string(body)},
		{name: "empty", token: ""},
		{name: "not a paseto token", token: "not-a-token"},
		{name: "truncated", token: tokenStr[:len(tokenStr)-5]},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := maker.VerifyToken(tc.token)
			require.ErrorIs(t, err, ErrInvalidToken)
			require.Nil(t, payload)
		})
	}
}

func TestPasetoMakerRejectsTokenFromAnotherKey(t *testing.T) {
	issuer, err := NewPasetoMaker(randomSymmetricKey(t))
	require.NoError(t, err)

	attacker, err := NewPasetoMaker(randomSymmetricKey(t))
	require.NoError(t, err)
	tokenStr, _, err := issuer.CreateToken(uuid.New(), time.Minute)
	require.NoError(t, err)

	payload, err := attacker.VerifyToken(tokenStr)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, payload)
}
