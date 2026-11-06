package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/joekings2k/staypass-be/internal/app"
)

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, http.StatusBadRequest, "bad_request", msg)
}

func Unauthorized(w http.ResponseWriter, msg string) {
	Error(w, http.StatusUnauthorized, "unauthorized", msg)
}

func Forbidden(w http.ResponseWriter, msg string) {
	Error(w, http.StatusForbidden, "forbidden", msg)
}

func NotFound(w http.ResponseWriter, msg string) {
	Error(w, http.StatusNotFound, "not_found", msg)
}

func Conflict(w http.ResponseWriter, msg string) {
	Error(w, http.StatusConflict, "conflict", msg)
}

func Internal(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "internal", "something went wrong")
}

func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("internal server error",
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", middleware.GetReqID(r.Context()),
	)
	Internal(w)
}

func statusFor(k app.Kind) int {
	switch k {
	case app.KindInvalid:
		return http.StatusBadRequest
	case app.KindNotFound:
		return http.StatusNotFound
	case app.KindConflict:
		return http.StatusConflict
	case app.KindUnauthorized:
		return http.StatusUnauthorized
	case app.KindForbidden:
		return http.StatusForbidden
	case app.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func codeFor(k app.Kind) string {
	switch k {
	case app.KindInvalid:
		return "bad_request"
	case app.KindNotFound:
		return "not_found"
	case app.KindConflict:
		return "conflict"
	case app.KindUnauthorized:
		return "unauthorized"
	case app.KindForbidden:
		return "forbidden"
	case app.KindUnavailable:
		return "unavailable"
	default:
		return "internal"
	}
}

func RespondWithError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := app.From(err)
	if !ok || appErr.Kind == app.KindInternal {

		ServerError(w, r, err)
		return
	}

	if appErr.Err != nil {
		slog.Warn("request failed",
			"error", appErr.Err,
			"kind", appErr.Kind.String(),
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", middleware.GetReqID(r.Context()),
		)
	}

	code := appErr.Code
	if code == "" {
		code = codeFor(appErr.Kind)
	}
	Error(w, statusFor(appErr.Kind), code, appErr.Message)
}
