package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
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

// Internal deliberately hides the real error from the client. Log the
// underlying err in your handler or middleware; never leak it in the body.
func Internal(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "internal", "something went wrong")
}

// ServerError logs the real error (with the request id and route so it can be
// traced) and then returns the generic 500 to the client. Use this for any
// unexpected error path instead of calling Internal directly, so no 500 ever
// goes unlogged.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("internal server error",
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", middleware.GetReqID(r.Context()),
	)
	Internal(w)
}
