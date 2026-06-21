package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/auth"
	"github.com/snowykami/games-platform/server/internal/config"
	"github.com/snowykami/games-platform/server/internal/games"
	"github.com/snowykami/games-platform/server/internal/gomoku"
	"github.com/snowykami/games-platform/server/internal/httpx"
	"github.com/snowykami/games-platform/server/internal/i18n"
	"github.com/snowykami/games-platform/server/internal/mahjong"
	"github.com/snowykami/games-platform/server/internal/runtimecheck"
	"github.com/snowykami/games-platform/server/internal/socialdeduction"
	"github.com/snowykami/games-platform/server/internal/uno"
	frontend "github.com/snowykami/games-platform/server/internal/web"
	"github.com/snowykami/games-platform/server/internal/xiangqi"
)

type gamesResponse struct {
	Games []games.Definition `json:"games"`
}

func main() {
	configureLogger()
	cfg := config.Load()
	if err := runtimecheck.RequireStartup(context.Background(), cfg); err != nil {
		slog.Error("startup dependency check failed", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           routes(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info(
		"server listening",
		"addr", server.Addr,
		"dbConfigured", cfg.Database.Enabled(),
		"redisConfigured", cfg.Redis.Enabled(),
		"llmConfigured", cfg.AI.Enabled(),
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "error", err)
		panic(err)
	}
}

func routes(cfg config.Config) http.Handler {
	authStore, err := auth.NewPostgresStore(context.Background(), cfg.Database.URL)
	if err != nil {
		slog.Error("auth store initialization failed", "error", err)
		os.Exit(1)
	}
	gameUsageStore, err := games.NewUsageStore(context.Background(), cfg.Database.URL)
	if err != nil {
		slog.Error("game usage store initialization failed", "error", err)
		os.Exit(1)
	}
	authHandler := auth.NewHandler(authStore, cfg.OIDC, cfg.HTTP.SecureSessionCookie)
	aiProvider := aiplayer.NewOpenAIProvider(cfg.AI)
	aiHandler := aiplayer.NewHandler(aiProvider)
	gomokuHandler := gomoku.NewHandler(gomoku.NewManager(aiProvider))
	mahjongHandler := mahjong.NewHandler(mahjong.NewManager(aiProvider))
	werewolfHandler := socialdeduction.NewHandler(socialdeduction.NewManager(socialdeduction.GameWerewolf, aiProvider))
	avalonHandler := socialdeduction.NewHandler(socialdeduction.NewManager(socialdeduction.GameAvalon, aiProvider))
	undercoverHandler := socialdeduction.NewHandler(socialdeduction.NewManager(socialdeduction.GameUndercover, aiProvider))
	unoHandler := uno.NewHandler(uno.NewManager(aiProvider))
	xiangqiHandler := xiangqi.NewHandler(xiangqi.NewManager(aiProvider))

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(securityHeaders)
	router.Use(auth.Middleware(authStore))

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Route("/api", func(api chi.Router) {
		api.Mount("/auth", authHandler.Routes())
		api.Get("/games", func(w http.ResponseWriter, r *http.Request) {
			gameList := games.ListForLocale(i18n.FromRequest(r))
			if user, ok := auth.UserFromContext(r.Context()); ok {
				stats, err := gameUsageStore.StatsForUser(r.Context(), user.ID)
				if err != nil {
					slog.Warn("game usage lookup failed", "user", user.ID, "error", err)
				} else {
					gameList = games.SortByUsage(gameList, stats)
				}
			}
			httpx.WriteJSON(w, http.StatusOK, gamesResponse{Games: gameList})
		})
		api.Group(func(protected chi.Router) {
			protected.Use(auth.RequireUser)
			protected.Post("/games/{gameSlug}/usage", func(w http.ResponseWriter, r *http.Request) {
				gameSlug := chi.URLParam(r, "gameSlug")
				if !games.Exists(gameSlug) {
					httpx.WriteErrorKey(w, r, http.StatusNotFound, "game_not_found")
					return
				}
				user, _ := auth.UserFromContext(r.Context())
				if err := gameUsageStore.RecordUse(r.Context(), user.ID, gameSlug); err != nil {
					slog.Warn("game usage record failed", "user", user.ID, "game", gameSlug, "error", err)
					httpx.WriteErrorKey(w, r, http.StatusInternalServerError, "game_usage_record_failed")
					return
				}
				httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
			})
			protected.Mount("/ai", aiHandler.Routes())
			protected.Mount("/gomoku", gomokuHandler.Routes())
			protected.Mount("/avalon", avalonHandler.Routes())
			protected.Mount("/mahjong", mahjongHandler.Routes())
			protected.Mount("/uno", unoHandler.Routes())
			protected.Mount("/undercover", undercoverHandler.Routes())
			protected.Mount("/werewolf", werewolfHandler.Routes())
			protected.Mount("/xiangqi", xiangqiHandler.Routes())
		})
		api.Group(func(admin chi.Router) {
			admin.Use(auth.RequireAdmin)
			admin.Mount("/admin", authHandler.AdminRoutes())
		})
	})

	router.With(auth.RequireUser).Get("/ws/gomoku", gomokuHandler.WebSocket)
	router.With(auth.RequireUser).Get("/ws/avalon", avalonHandler.WebSocket)
	router.With(auth.RequireUser).Get("/ws/mahjong", mahjongHandler.WebSocket)
	router.With(auth.RequireUser).Get("/ws/uno", unoHandler.WebSocket)
	router.With(auth.RequireUser).Get("/ws/undercover", undercoverHandler.WebSocket)
	router.With(auth.RequireUser).Get("/ws/werewolf", werewolfHandler.WebSocket)
	router.With(auth.RequireUser).Get("/ws/xiangqi", xiangqiHandler.WebSocket)
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || r.URL.Path == "/ws" || strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			http.NotFound(w, r)
			return
		}

		frontend.Handler().ServeHTTP(w, r)
	})

	return router
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' ws: wss:; frame-ancestors 'self'; base-uri 'self'; form-action 'self'")
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
