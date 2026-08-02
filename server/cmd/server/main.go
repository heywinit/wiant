package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/heywinit/wiant/server/config"
	wiantMiddleware "github.com/heywinit/wiant/server/internal/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("unable to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err = pool.Ping(ctx); err != nil {
		logger.Error("unable to connect to database", "error", err)
		os.Exit(1)
	}

	server := &http.Server{Addr: cfg.Address, Handler: prepareHandler(logger), ReadHeaderTimeout: 10 * time.Second}

	go func() {
		logger.Info("started http server", "address", cfg.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to gracefully shut down server", "error", err)
	}
}

func prepareHandler(logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID, middleware.ClientIPFromRemoteAddr, wiantMiddleware.RequestLogger(logger), middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("wiant"))
	})

	return r
}
