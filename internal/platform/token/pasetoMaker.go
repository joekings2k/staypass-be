package token

import (
	"errors"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/google/uuid"
	"github.com/o1egl/paseto"
)

var (
	ErrInvalidKeySize = errors.New("Invalid Key Size")
)

type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, ErrInvalidKeySize
	}
	token := &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}
	return token, nil
}

func (maker *PasetoMaker) CreateToken(userID uuid.UUID, tokenType TokenType, duration time.Duration) (string, Payload, error) {
	payload, err := NewPayload(userID, tokenType, duration)
	if err != nil {
		return "", Payload{}, err
	}

	token, err := maker.paseto.Encrypt(maker.symmetricKey, payload, nil)
	if err != nil {
		return "", Payload{}, err
	}
	return token, *payload, nil
}

func (maker *PasetoMaker) VerifyToken(token string, expected TokenType) (*Payload, error) {
	payload := &Payload{}
	err := maker.paseto.Decrypt(token, maker.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}
	// Tokens minted before Type existed decode with an empty Type and are
	// rejected here too; those users simply log in again.
	if payload.Type != expected {
		return nil, ErrInvalidToken
	}
	err = payload.Valid()
	if err != nil {
		return nil, err

	}
	return payload, nil
}
