package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Gerry3010/pipepush/internal/config"
	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/push"
	"github.com/Gerry3010/pipepush/internal/server"
)

func main() {
	genKeys := flag.Bool("gen-vapid-keys", false, "Generate a VAPID key pair and exit")
	flag.Parse()

	if *genKeys {
		priv, pub, err := push.GenerateVAPIDKeys()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error generating keys:", err)
			os.Exit(1)
		}
		fmt.Println("VAPID_PRIVATE_KEY=" + priv)
		fmt.Println("VAPID_PUBLIC_KEY=" + pub)
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.LoadServerConfig()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	if !cfg.VAPIDEnabled() {
		slog.Warn("VAPID keys not set — Web Push disabled. Generate with: pipepush-server -gen-vapid-keys")
	}

	// Run migrations
	slog.Info("running database migrations")
	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	router := server.NewRouter(cfg, database)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "port", cfg.Port, "baseURL", cfg.BaseURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
