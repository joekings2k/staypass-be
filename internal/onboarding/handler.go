package onboarding

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/joekings2k/staypass-be/internal/estate"
	"github.com/joekings2k/staypass-be/internal/membership"
	"github.com/joekings2k/staypass-be/internal/platform/utils"
	"github.com/joekings2k/staypass-be/internal/platform/web"
	"github.com/joekings2k/staypass-be/internal/user"
)

type OnboardingServicer interface {
	Register(
		ctx context.Context,
		params RegisterParams,
	) (*estate.Estate, *user.User, *membership.Membership, error)
}
type Handler struct{ service OnboardingServicer }

func NewHandler(s OnboardingServicer) *Handler {
	return &Handler{
		service: s,
	}
}

// RegisterRoutes mounts the onboarding endpoints on the given router. Called by
// the server package under the /api/v1 prefix.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/estate-onboarding", h.RegisterEstate)
}

type registerEstateRequest struct {
	FullName   string `json:"fullName" validate:"required"`
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"phone" validate:"required,min=8"`
	Password   string `json:"password" validate:"required,min=8,max=30"`
	EstateName string `json:"estateName" validate:"required"`
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
}

type registerEstateResponse struct {
	EstateID string
	UserID   string
}

func (h *Handler) RegisterEstate(w http.ResponseWriter, r *http.Request) {
	var req registerEstateRequest
	if err := web.DecodeAndValidate(w, r, &req); err != nil {
		slog.Error("Invalid request", "error", err)
		web.InvalidRequest(w, err)
		return
	}
	params := RegisterParams{
		Phone:      req.Phone,
		FullName:   req.FullName,
		Password:   req.Password,
		Email:      req.Email,
		EstateName: req.EstateName,
		Slug:       utils.GenerateSlug(req.EstateName),
		Address:    req.Address,
		City:       req.City,
		State:      req.State,
	}

	e, u, _, err := h.service.Register(r.Context(), params)
	if err != nil {
		web.RespondWithError(w, r, err)
		return
	}

	web.JSON(w, http.StatusCreated, registerEstateResponse{
		EstateID: e.ID.String(),
		UserID:   uuid.UUID(u.ID).String(),
	})
}
