package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/snowykami/games-platform/server/internal/i18n"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

func WriteErrorKey(w http.ResponseWriter, r *http.Request, status int, key string) {
	WriteJSON(w, status, ErrorResponse{Error: i18n.T(i18n.FromRequest(r), key)})
}

func DecodeJSON(r *http.Request, value any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(value)
}
