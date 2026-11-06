package user

import "github.com/joekings2k/staypass-be/internal/app"

var (
	ErrUserNotFound = app.NotFound("user not found")
	ErrEmailTaken   = app.Conflict("email already registered")
)
