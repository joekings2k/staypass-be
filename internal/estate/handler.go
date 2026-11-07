package estate

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/joekings2k/staypass-be/internal/platform/web"
)

type Handler struct {
	repo EstateRepository
}

func NewHandler(repo EstateRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.Get)
}

type estateResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Address   *string   `json:"address"`
	City      *string   `json:"city"`
	State     *string   `json:"state"`
	Timezone  string    `json:"timezone"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "estate_id"))
	if err != nil {
		web.BadRequest(w, "invalid estate id")
		return
	}
	e, err := h.repo.FindEstateByID(r.Context(), id)
	if err != nil {
		web.RespondWithError(w, r, err)
		return
	}
	web.JSON(w, http.StatusOK, estateResponse{
		ID:        e.ID.String(),
		Name:      e.Name,
		Slug:      e.Slug,
		Address:   e.Address,
		City:      e.City,
		State:     e.State,
		Timezone:  e.Timezone,
		IsActive:  e.IsActive,
		CreatedAt: e.CreatedAt,
	})
}
