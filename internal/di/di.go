// Package di wires the application dependencies. main only calls Run.
package di

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	store "github.com/esrid/mon-template-go/internal/adapters/stores"
)

// App holds the wired dependencies. Add services and handlers here as the
// project grows; keep construction in New so wiring stays in one place.
type App struct {
	DB    *sql.DB
	Store *store.Store
}

func New(dsn string) (*App, error) {
	db, err := store.Open(dsn)
	if err != nil {
		return nil, err
	}
	return &App{DB: db, Store: store.NewStore(db)}, nil
}

func (a *App) Close() error { return a.DB.Close() }

// Run builds the app and blocks until SIGINT/SIGTERM.
// Start long-running components (HTTP server, workers) here.
func Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "app.db"
	}

	app, err := New(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = app.Close() }()

	slog.Info("started", "dsn", dsn)
	<-ctx.Done()
	slog.Info("shutting down")
	return nil
}
