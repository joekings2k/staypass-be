package token

import (
	"time"

	"github.com/google/uuid"
)

type Maker interface {
	CreateToken(userID uuid.UUID, tokenType TokenType, duration time.Duration) (string, Payload, error)
	// VerifyToken rejects a token whose type is not expected, so a refresh
	// token can never be used where an access token is required (or vice versa).
	VerifyToken(token string, expected TokenType) (*Payload, error)
}
