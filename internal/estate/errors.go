package estate

import "github.com/joekings2k/staypass-be/internal/app"

// The estate package's error vocabulary. Each one is classified where it is
// declared, so platform/web can render it without importing this package.
// Compare with errors.Is, never by matching on the message.
var (
	ErrEstateNotFound          = app.NotFound("estate not found")
	ErrEstateAlreadyExists     = app.Conflict("estate already exists")
	ErrInvalidEstateData       = app.Invalid("invalid estate data")
	ErrReferenceRecordNotFound = app.NotFound("reference record not found")
)
