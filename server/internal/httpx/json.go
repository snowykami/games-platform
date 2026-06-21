package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/snowykami/games-platform/server/internal/i18n"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

const MaxJSONBodyBytes int64 = 1 << 20

var ErrBodyTooLarge = errors.New("json_body_too_large")

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
	limited := &io.LimitedReader{R: r.Body, N: MaxJSONBodyBytes + 1}
	decoder := json.NewDecoder(limited)
	if err := decoder.Decode(value); err != nil {
		if limited.N <= 0 {
			return ErrBodyTooLarge
		}
		return err
	}
	if limited.N <= 0 {
		return ErrBodyTooLarge
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if limited.N <= 0 {
			return ErrBodyTooLarge
		}
		if err != nil {
			return err
		}
		return errors.New("invalid_json_body")
	}
	return nil
}
