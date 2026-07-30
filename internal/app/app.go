// Package app is the composition root and process lifecycle.
//
// It is the only package that knows both the concrete adapters and the
// features. Features never import each other; they meet here.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/esrid/mon-template-go/internal/feature/readiness"
	"github.com/esrid/mon-template-go/internal/feature/subscriber"
	"github.com/esrid/mon-template-go/internal/platform/config"
	"github.com/esrid/mon-template-go/internal/platform/sqlite"
)

type App struct {
	server          *http.Server
	database        io.Closer
	shutdownTimeout time.Duration

	// One field per feature service. Wiring a new feature is: build it here,
	// mount it in routes.go.
	readiness   *readiness.Service
	subscribers *subscriber.Service
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Choosing SQLite is a decision of this root, not of any feature.
	// Swapping in Postgres means changing this line and implementing the
	// same feature-owned ports.
	database, err := sqlite.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	// Features are built here and nowhere else. readiness.Store is satisfied by
	// *sqlite.DB directly; subscriber needs its own SQL, so it gets a store
	// built from the same connection, plus the clock it depends on.
	app := &App{
		database:        database,
		shutdownTimeout: cfg.ShutdownTimeout,
		readiness:       readiness.New(database),
		subscribers:     subscriber.New(subscriber.NewSQLiteStore(database), time.Now),
	}
	app.server = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.routes(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	return app, nil
}

func Run(ctx context.Context, cfg config.Config) error {
	app, err := New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Close(); err != nil {
			slog.Error("close application", "err", err)
		}
	}()
	return app.Run(ctx)
}

func (a *App) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	serverResult := make(chan error, 1)
	go func() {
		err := a.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverResult <- err
	}()
	slog.Info("http server started", "addr", a.server.Addr)

	select {
	case err := <-serverResult:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		_ = a.server.Close()
		return fmt.Errorf("http server shutdown: %w", err)
	}
	if err := <-serverResult; err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	slog.Info("http server stopped")
	return nil
}

func (a *App) Close() error {
	return a.database.Close()
}
