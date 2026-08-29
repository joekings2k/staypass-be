package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashToken(token string) string {
	hashedToken := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hashedToken[:])
}
