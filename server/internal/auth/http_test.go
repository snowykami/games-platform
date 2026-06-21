package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/snowykami/games-platform/server/internal/config"
)

func TestOIDCStateExpires(t *testing.T) {
	handler := NewHandler(NewStore(), nil, true)
	handler.oidcState["expired"] = oidcStateEntry{
		ReturnTo:  "/games/uno",
		ExpiresAt: time.Now().Add(-time.Minute),
	}
	handler.oidcState["fresh"] = oidcStateEntry{
		ReturnTo:  "/games/xiangqi",
		ExpiresAt: time.Now().Add(time.Minute),
	}

	if _, ok := handler.takeOIDCState("expired"); ok {
		t.Fatal("expected expired OIDC state to be rejected")
	}
	if returnTo, ok := handler.takeOIDCState("fresh"); !ok || returnTo != "/games/xiangqi" {
		t.Fatalf("expected fresh OIDC state, got returnTo=%q ok=%t", returnTo, ok)
	}
}

func TestOIDCLoginRejectsStateOverflow(t *testing.T) {
	handler := NewHandler(NewStore(), []config.OIDCProviderConfig{{
		Key:          "demo",
		DisplayName:  "Demo",
		ClientID:     "client",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	}}, true)
	handler.oidc["demo"] = &oidcProviderRuntime{key: "demo", displayName: "Demo"}
	for index := range maxOIDCStates {
		handler.oidcState[randomHex(8)] = oidcStateEntry{
			ReturnTo:  "/",
			ExpiresAt: time.Now().Add(time.Minute + time.Duration(index)*time.Nanosecond),
		}
	}

	router := chi.NewRouter()
	router.Get("/oidc/{providerKey}/login", handler.oidcLogin)
	request := httptest.NewRequest(http.MethodGet, "/oidc/demo/login", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d body=%s", response.Code, response.Body.String())
	}
}
