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
	"github.com/jackc/pgx/v5"

	wiantMiddleware "github.com/heywinit/wiant/server/internal/middleware"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	handler := prepareHandler(logger)
	server := &http.Server{
		Addr: ":3000",
		Handler: handler,
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))

	if err != nil {
		logger.Info("unable to connect to database", "err" , err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	go func() {
		logger.Info("started http server", "port", 3000)

		err := http.ListenAndServe(":3000", handler)

		if err != nil {
			logger.Error("error while trying to start the server", "error", err.Error())
		}
	}()

	<-ctx.Done();
	stop();

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel();

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to gracefully shut down server", "error", err)
	}
}


func prepareHandler(logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(wiantMiddleware.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("wiant"))
	})

	return r
}
