package onboarding

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/joekings2k/staypass-be/internal/estate"
	"github.com/joekings2k/staypass-be/internal/platform/web"
	"github.com/joekings2k/staypass-be/internal/user"
)

type Handler struct{ service *OnboardingService }

func NewHandler(s *OnboardingService) *Handler {
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
	Slug       string `json:"slug"       validate:"required"`
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
		Slug:       req.Slug,
		Address:    req.Address,
		City:       req.City,
		State:      req.State,
	}

	e, u, _, err := h.service.Register(r.Context(), params)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrEmailTaken):
			web.Conflict(w, "email already registered")
		case errors.Is(err, estate.ErrEstateAlreadyExists):
			web.Conflict(w, "estate already exists")
		default:
			web.ServerError(w, r, err)
		}
		return
	}

	web.JSON(w, http.StatusCreated, registerEstateResponse{
		EstateID: e.ID.String(),
		UserID:   uuid.UUID(u.ID).String(),
	})
}
