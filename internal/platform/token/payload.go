package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenType distinguishes access tokens from refresh tokens. Both are signed
// with the same key, so the type inside the encrypted payload is the only
// thing stopping one from being accepted as the other.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Payload struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      TokenType
	IssuedAt  time.Time
	ExpiresAt time.Time
}

func NewPayload(userID uuid.UUID, tokenType TokenType, duration time.Duration) (*Payload, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(duration)

	payload := &Payload{
		ID:        id,
		UserID:    userID,
		Type:      tokenType,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}

	return payload, nil
}

func (p *Payload) Valid() error {
	if time.Now().After(p.ExpiresAt) {
		return ErrExpiredToken
	}
	return nil
}
