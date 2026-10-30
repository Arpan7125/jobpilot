package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// fieldErrors maps a request field to what is wrong with it.
type fieldErrors map[string]string

func writeValidation(w http.ResponseWriter, errs fieldErrors) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "validation failed", "fields": errs})
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

// decode reads a JSON body into dst, rejecting unknown fields and trailing data.
// It writes the error response itself and returns false on failure.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		switch {
		case errors.As(err, &tooBig):
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			writeError(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), "json: "))
		default:
			writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		}
		return false
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "request body must be a single JSON object")
		return false
	}
	return true
}
