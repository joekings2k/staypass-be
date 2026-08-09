package auth

import (
	"fmt"
	"net/http"

	"github.com/joekings2k/staypass-be/internal/platform/web"
)

type Handler struct {
	service *AuthService
}

func NewHandlerService(service *AuthService) *Handler {
	return &Handler{
		service: service,
	}
}

type registerRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=50"`
	Phone     string `json:"phone" validate:"required"`
	FirstName string `json:"firstname" validate:"required"`
	LastName  string `json:"lastname" validate:"required"`
}

func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := web.DecodeAndValidate(w, r, &req); err != nil {
		web.BadRequest(w, "invalid request")
	}

	err := h.service.Register(r.Context(), RegisterParams{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	})
	fmt.Println(err)

}
