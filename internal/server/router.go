package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Gerry3010/pipepush/internal/config"
	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/push"
	"github.com/Gerry3010/pipepush/internal/server/handlers"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

// NewRouter wires up all routes and middleware.
func NewRouter(cfg *config.ServerConfig, database *db.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(corsMiddleware)

	// Shared components
	hub := handlers.NewSSEHub()
	dispatcher := push.NewDispatcher(database, cfg.VAPIDPublic, cfg.VAPIDPrivate, cfg.VAPIDEmail)

	// Handlers
	authH := handlers.NewAuthHandler(database, cfg.JWTSecret, cfg.JWTExpiry)
	projectH := handlers.NewProjectHandler(database)
	pipelineH := handlers.NewPipelineHandler(database)
	tokenH := handlers.NewTokenHandler(database)
	runH := handlers.NewRunHandler(database)
	webhookH := handlers.NewWebhookHandler(database, dispatcher, hub)
	pushH := handlers.NewPushHandler(database, cfg.VAPIDPublic)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api", func(r chi.Router) {
		// Public
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Post("/webhook", webhookH.Handle)
		r.Get("/push/vapid-key", pushH.VAPIDPublicKey)

		// Authenticated
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(cfg.JWTSecret))

			r.Get("/projects", projectH.List)
			r.Post("/projects", projectH.Create)
			r.Delete("/projects/{id}", projectH.Delete)

			r.Get("/projects/{projectID}/pipelines", pipelineH.List)
			r.Post("/projects/{projectID}/pipelines", pipelineH.Create)
			r.Delete("/pipelines/{id}", pipelineH.Delete)

			r.Get("/projects/{projectID}/tokens", tokenH.List)
			r.Post("/tokens", tokenH.Create)
			r.Delete("/tokens/{id}", tokenH.Revoke)
			r.Delete("/tokens/{id}/permanent", tokenH.Delete)

			r.Get("/pipelines/{pipelineID}/runs", runH.List)
			r.Get("/runs/{id}", runH.Get)

			r.Post("/push/subscribe", pushH.Subscribe)
			r.Delete("/push/subscribe", pushH.Unsubscribe)

			r.Get("/events", hub.Handler)
		})
	})

	// Optionally serve the built web app (SPA) with history-API fallback.
	if cfg.StaticDir != "" {
		serveSPA(r, cfg.StaticDir)
	}

	return r
}

// serveSPA serves static files from dir, falling back to index.html for client
// routes (so React Router deep links work on refresh).
func serveSPA(r chi.Router, dir string) {
	fs := http.FileServer(http.Dir(dir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// API and health are handled above; everything else is the SPA.
		path := filepath.Join(dir, filepath.Clean(req.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, req)
			return
		}
		if strings.HasPrefix(req.URL.Path, "/assets/") {
			http.NotFound(w, req)
			return
		}
		http.ServeFile(w, req, filepath.Join(dir, "index.html"))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
