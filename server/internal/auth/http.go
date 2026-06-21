package auth

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/snowykami/games-platform/server/internal/config"
	"github.com/snowykami/games-platform/server/internal/httpx"
	"golang.org/x/oauth2"
)

type Handler struct {
	store         *Store
	oidc          map[string]*oidcProviderRuntime
	secureCookies bool
	stateMu       sync.Mutex
	oidcState     map[string]oidcStateEntry
}

const maxOIDCStates = 1024

type guestLoginRequest struct {
	GuestUUID string `json:"guestUuid"`
}

type meResponse struct {
	User *User `json:"user"`
}

type oidcProvider struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Enabled     bool   `json:"enabled"`
}

type oidcProviderRuntime struct {
	key         string
	displayName string
	oauth       oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

type oidcStateEntry struct {
	ReturnTo  string
	ExpiresAt time.Time
}

type oidcClaims struct {
	Subject           string `json:"sub"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
}

func NewHandler(store *Store, providers []config.OIDCProviderConfig, secureCookies bool) *Handler {
	return &Handler{
		store:         store,
		oidc:          initOIDCProviders(providers),
		secureCookies: secureCookies,
		oidcState:     map[string]oidcStateEntry{},
	}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()
	router.Get("/me", h.me)
	router.Post("/guest", h.guest)
	router.Post("/logout", h.logout)
	router.Get("/oidc/providers", h.oidcProviders)
	router.Get("/oidc/{providerKey}/login", h.oidcLogin)
	router.Get("/oidc/{providerKey}/callback", h.oidcCallback)
	return router
}

func (h *Handler) AdminRoutes() http.Handler {
	router := chi.NewRouter()
	router.Get("/users", h.listUsers)
	router.Post("/users/{userID}/ban", h.banUser)
	router.Post("/users/{userID}/unban", h.unbanUser)
	return router
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, meResponse{})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, meResponse{User: user})
}

func (h *Handler) guest(w http.ResponseWriter, r *http.Request) {
	var request guestLoginRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_json_body")
		return
	}

	user, session, err := h.store.CreateGuestSession(request.GuestUUID)
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}

	h.setSessionCookie(w, session)
	httpx.WriteJSON(w, http.StatusOK, meResponse{User: user})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.store.DeleteSession(SessionTokenFromRequest(r))
	h.clearSessionCookie(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) oidcProviders(w http.ResponseWriter, _ *http.Request) {
	providers := make([]oidcProvider, 0, len(h.oidc))
	for _, provider := range h.oidc {
		providers = append(providers, oidcProvider{
			Key:         provider.key,
			DisplayName: provider.displayName,
			Enabled:     true,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string][]oidcProvider{"providers": providers})
}

func (h *Handler) oidcLogin(w http.ResponseWriter, r *http.Request) {
	provider := h.oidc[chi.URLParam(r, "providerKey")]
	if provider == nil {
		httpx.WriteErrorKey(w, r, http.StatusNotFound, "oidc_provider_not_found")
		return
	}

	state := "oidc_" + randomHex(18)
	returnTo := safeReturnTo(r.URL.Query().Get("returnTo"))

	h.stateMu.Lock()
	h.cleanupOIDCStateLocked(time.Now())
	if len(h.oidcState) >= maxOIDCStates {
		h.stateMu.Unlock()
		httpx.WriteErrorKey(w, r, http.StatusTooManyRequests, "too_many_oidc_logins")
		return
	}
	h.oidcState[state] = oidcStateEntry{ReturnTo: returnTo, ExpiresAt: time.Now().Add(10 * time.Minute)}
	h.stateMu.Unlock()

	http.Redirect(w, r, provider.oauth.AuthCodeURL(state), http.StatusFound)
}

func (h *Handler) oidcCallback(w http.ResponseWriter, r *http.Request) {
	provider := h.oidc[chi.URLParam(r, "providerKey")]
	if provider == nil {
		httpx.WriteErrorKey(w, r, http.StatusNotFound, "oidc_provider_not_found")
		return
	}

	returnTo, ok := h.takeOIDCState(r.URL.Query().Get("state"))
	if !ok {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_oidc_state")
		return
	}

	oauthToken, err := provider.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_code_exchange")
		return
	}

	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "oidc_id_token_missing")
		return
	}

	idToken, err := provider.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_id_token")
		return
	}

	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, "invalid_claims")
		return
	}

	user, session, err := h.store.CreateOIDCSession(provider.key, claims.Subject, displayNameFromClaims(claims))
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}

	h.setSessionCookie(w, session)
	_ = user
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (h *Handler) takeOIDCState(state string) (string, bool) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	entry, ok := h.oidcState[state]
	delete(h.oidcState, state)
	if !ok || time.Now().After(entry.ExpiresAt) {
		return "", false
	}
	return entry.ReturnTo, true
}

func (h *Handler) cleanupOIDCStateLocked(now time.Time) {
	for state, entry := range h.oidcState {
		if now.After(entry.ExpiresAt) {
			delete(h.oidcState, state)
		}
	}
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, session Session) {
	setSessionCookie(w, session, h.secureCookies)
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	clearSessionCookie(w, h.secureCookies)
}

func (h *Handler) listUsers(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string][]*User{"users": h.store.ListUsers()})
}

func (h *Handler) banUser(w http.ResponseWriter, r *http.Request) {
	h.setBanned(w, r, true)
}

func (h *Handler) unbanUser(w http.ResponseWriter, r *http.Request) {
	h.setBanned(w, r, false)
}

func (h *Handler) setBanned(w http.ResponseWriter, r *http.Request, banned bool) {
	user, err := h.store.SetBanned(chi.URLParam(r, "userID"), banned)
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusBadRequest, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, meResponse{User: user})
}

func Middleware(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := SessionTokenFromRequest(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			user, ok := store.UserBySession(token)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
		})
	}
}

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			httpx.WriteErrorKey(w, r, http.StatusUnauthorized, "login_required")
			return
		}
		if user.Banned {
			httpx.WriteErrorKey(w, r, http.StatusForbidden, "user_banned")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r.Context())
		if user.Role != RoleAdmin {
			httpx.WriteErrorKey(w, r, http.StatusForbidden, "admin_required")
			return
		}

		next.ServeHTTP(w, r)
	}))
}

func initOIDCProviders(configs []config.OIDCProviderConfig) map[string]*oidcProviderRuntime {
	providers := map[string]*oidcProviderRuntime{}
	for _, cfg := range configs {
		if cfg.Key == "" || cfg.DisplayName == "" || cfg.IssuerURL == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
			continue
		}

		provider, err := oidc.NewProvider(context.Background(), cfg.IssuerURL)
		if err != nil {
			continue
		}

		scopes := []string{oidc.ScopeOpenID, "profile", "email"}
		scopes = append(scopes, cfg.Scopes...)
		providers[cfg.Key] = &oidcProviderRuntime{
			key:         cfg.Key,
			displayName: cfg.DisplayName,
			oauth: oauth2.Config{
				ClientID:     cfg.ClientID,
				ClientSecret: cfg.ClientSecret,
				RedirectURL:  cfg.RedirectURL,
				Endpoint:     provider.Endpoint(),
				Scopes:       scopes,
			},
			verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		}
	}
	return providers
}

func safeReturnTo(returnTo string) string {
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		return "/"
	}
	return returnTo
}

func displayNameFromClaims(claims oidcClaims) string {
	for _, candidate := range []string{claims.Name, claims.PreferredUsername, claims.Email} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return "OIDC Player"
}
