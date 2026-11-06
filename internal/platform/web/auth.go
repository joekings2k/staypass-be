package web

import (
	"context"

	"github.com/google/uuid"
	"github.com/joekings2k/staypass-be/internal/membership"
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

func WithRoles(ctx context.Context, roles []membership.MembershipRole) context.Context {
	return context.WithValue(ctx, rolesKey, roles)
}

func RolesFrom(ctx context.Context) ([]membership.MembershipRole, bool) {
	roles, ok := ctx.Value(rolesKey).([]membership.MembershipRole)
	return roles, ok
}
