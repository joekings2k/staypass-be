package app

import (
	"errors"
	"fmt"
)

type Kind uint8

const (
	KindInternal Kind = iota
	KindInvalid
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
	KindUnavailable
)

func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "invalid"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindUnauthorized:
		return "unauthorized"
	case KindForbidden:
		return "forbidden"
	case KindUnavailable:
		return "unavailable"
	default:
		return "internal"
	}
}

type Error struct {
	Kind Kind

	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) WithCode(code string) *Error {
	clone := *e
	clone.Code = code
	return &clone
}

func Invalid(msg string) *Error      { return &Error{Kind: KindInvalid, Message: msg} }
func NotFound(msg string) *Error     { return &Error{Kind: KindNotFound, Message: msg} }
func Conflict(msg string) *Error     { return &Error{Kind: KindConflict, Message: msg} }
func Unauthorized(msg string) *Error { return &Error{Kind: KindUnauthorized, Message: msg} }
func Forbidden(msg string) *Error    { return &Error{Kind: KindForbidden, Message: msg} }
func Unavailable(msg string) *Error  { return &Error{Kind: KindUnavailable, Message: msg} }
func Internal(msg string) *Error     { return &Error{Kind: KindInternal, Message: msg} }

func From(err error) (*Error, bool) {
	var appErr *Error
	ok := errors.As(err, &appErr)
	return appErr, ok
}

func KindOf(err error) Kind {
	if appErr, ok := From(err); ok {
		return appErr.Kind
	}
	return KindInternal
}

func Wrap(sentinel error, cause error) error {
	if cause == nil {
		return sentinel
	}
	return fmt.Errorf("%w: %w", sentinel, cause)
}
