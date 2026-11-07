package web

import (
	"context"

	"github.com/google/uuid"
	"github.com/joekings2k/staypass-be/internal/platform/token"
)

type authContextKey int

const (
	payloadKey authContextKey = iota
	rolesKey
)

func WithAuthPayload(ctx context.Context, payload *token.Payload) context.Context {
	return context.WithValue(ctx, payloadKey, payload)
}

func AuthPayloadFrom(ctx context.Context) (*token.Payload, bool) {
	payload, ok := ctx.Value(payloadKey).(*token.Payload)
	return payload, ok
}

func UserIdFrom(ctx context.Context) (uuid.UUID, bool) {
	payload, ok := AuthPayloadFrom(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return payload.UserID, true
}

// WithRoles and RolesFrom are generic so this platform package never imports a
// domain package (doing so creates import cycles). Callers must use the same T
// on both sides, e.g. web.RolesFrom[membership.MembershipRole](ctx).
func WithRoles[T any](ctx context.Context, roles []T) context.Context {
	return context.WithValue(ctx, rolesKey, roles)
}

func RolesFrom[T any](ctx context.Context) ([]T, bool) {
	roles, ok := ctx.Value(rolesKey).([]T)
	return roles, ok
}
