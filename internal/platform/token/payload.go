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

type Payload struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	role      string
	ISsuedAt  time.Time
	ExpiresAt time.Time
}

func NewPayload(userID uuid.UUID, role string, duration time.Duration) (*Payload, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(duration)

	payload := &Payload{
		ID:        id,
		UserID:    userID,
		role:      role,
		ISsuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}

	return payload, nil
}

func (p *Payload) valid() error {
	if time.Now().After(p.ExpiresAt) {
		return ErrExpiredToken
	}
	return nil
}
