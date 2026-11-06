package server

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/joekings2k/staypass-be/internal/estate"
	"github.com/joekings2k/staypass-be/internal/membership"
	"github.com/joekings2k/staypass-be/internal/platform/token"
	"github.com/joekings2k/staypass-be/internal/platform/web"
)

const (
	authorizationHeaderKey  = "Authorization"
	authorizationTypeBearer = "bearer"
)

func RequireAuth(maker token.Maker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get(authorizationHeaderKey)
			if header == "" {
				web.Unauthorized(w, "authorization header is required")
				return
			}
			fields := strings.Fields(header)
			if len(fields) < 2 {
				web.Unauthorized(w, "invalid authorization header")
				return
			}

			authorizationType := strings.ToLower(fields[0])
			if authorizationType != authorizationTypeBearer {
				web.Unauthorized(w, "unsupported authorization type")
				return
			}
			accessToken := fields[1]
			payload, err := maker.VerifyToken(accessToken)
			if err != nil {
				web.Unauthorized(w, err.Error())
				return
			}
			ctx := web.WithAuthPayload(r.Context(), payload)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type EstateAccessReader interface {
	GetEstateRolesForUser(ctx context.Context, params membership.GetEstateRolesForUserParams) (membership.GetEstateRolesForUserResult, error)
}
type EstateReader interface {
	FindEstateByID(ctx context.Context, estateID uuid.UUID) (*estate.Estate, error)
}

func RequireEstate(membershipRepo EstateAccessReader, estateRepo EstateReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userId, ok := web.UserIdFrom(r.Context())
			if !ok {
				web.Unauthorized(w, "user not found")
				return
			}
			estateID := chi.URLParam(r, "estate_id")
			if estateID == "" {
				web.BadRequest(w, "estate id is required")
				return
			}
			estateIDUUID, err := uuid.Parse(estateID)
			if err != nil {
				web.BadRequest(w, "invalid estate id")
				return
			}
			estateResult, err := estateRepo.FindEstateByID(r.Context(), estateIDUUID)
			if err != nil {
				web.Internal(w)
				return
			}
			if estateResult.IsActive == false {
				web.Forbidden(w, "estate is not active")
				return
			}

			roles, err := membershipRepo.GetEstateRolesForUser(r.Context(), membership.GetEstateRolesForUserParams{
				UserID:   userId,
				EstateID: estateIDUUID,
			})
			if err != nil {
				web.Internal(w)
				return
			}
			if len(roles.Roles) == 0 {
				web.Forbidden(w, "user does not have access to this estate")
				return
			}
			ctx := web.WithRoles(r.Context(), roles.Roles)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role membership.MembershipRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, ok := web.RolesFrom(r.Context())
			if !ok {
				web.Internal(w)
				return
			}
			if !slices.Contains(roles, role) {
				web.Forbidden(w, "user does not have access to this resource")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
