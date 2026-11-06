package auth

import "github.com/joekings2k/staypass-be/internal/app"

// The auth package's error vocabulary.
var (
	// ErrInvalidCredentials is returned for every failed login: unknown
	// phone, unknown email, wrong password, account never claimed. They are
	// deliberately indistinguishable — telling them apart would let an
	// unauthenticated caller work out which phone numbers are registered.
	ErrInvalidCredentials = app.Unauthorized("invalid credentials")

	// ErrUnAuthorized covers a request that carries no usable identity at all.
	ErrUnAuthorized = app.Unauthorized("unauthorized")

	ErrUserNotFound = app.NotFound("user not found")
)
