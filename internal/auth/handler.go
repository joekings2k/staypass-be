package auth

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joekings2k/staypass-be/internal/platform/web"
)

const (
	CookieName = "refreshToken"
	// CookiePath must cover every route that reads the refresh cookie, and must
	// be identical on every write so rotation replaces the cookie in place.
	CookiePath = "/api/v1/auth"
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

type loginRequest struct {
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required,min=8,max=50"`
}
type LoginResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expiresat"`
}

func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := web.DecodeAndValidate(w, r, &req); err != nil {
		web.RespondWithError(w, r, err)
		return
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

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := web.DecodeAndValidate(w, r, &req); err != nil {
		web.RespondWithError(w, r, err)
		return
	}
	userAgent := r.UserAgent()

	result, err := h.service.Login(r.Context(), LoginParams{
		Identifier: req.Identifier,
		Password:   req.Password,
		UserAgent:  &userAgent,
		IP:         clientIP(r),
	})
	if err != nil {
		web.RespondWithError(w, r, err)
		return
	}
	setRefreshCookie(w, result.RefereshToken, result.RefereshTokenExpiresAt)
	response := LoginResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	}

	web.JSON(w, http.StatusOK, response)

}

type RefreshTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expiresat"`
}

func (h Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := web.UserIdFrom(r.Context())
	if !ok {
		web.RespondWithError(w, r, ErrUnAuthorized)
		return
	}

	response, err := h.service.GetUserAndEstates(r.Context(), userID)
	if err != nil {
		web.RespondWithError(w, r, err)
		return
	}

	web.JSON(w, http.StatusOK, response)
}

func (h Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		web.RespondWithError(w, r, ErrUnAuthorized)
		return
	}

	userAgent := r.UserAgent()

	result, err := h.service.Refresh(r.Context(), RefreshParams{
		RefreshToken: cookie.Value,
		UserAgent:    &userAgent,
		IP:           clientIP(r),
	})
	if err != nil {
		web.RespondWithError(w, r, err)
		return
	}

	setRefreshCookie(w, result.RefereshToken, result.RefereshTokenExpiresAt)

	web.JSON(w, http.StatusOK, RefreshTokenResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})

}

// Logout revokes the session behind the refresh cookie and clears the cookie.
// It always answers 204 so a client can call it unconditionally.
func (h Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(CookieName); err == nil && cookie.Value != "" {
		if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
			web.RespondWithError(w, r, err)
			return
		}
	}
	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     CookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func setRefreshCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     CookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func clientIP(r *http.Request) *netip.Addr {
	addr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		addr = host
	}

	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return nil
	}
	return &ip
}

// RegisterRoutes mounts the public auth endpoints. The server mounts them under
// CookiePath so the refresh cookie is sent to all three.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
}
