package main

import (
	"log/slog"
	"os"

	"github.com/esrid/mon-template-go/internal/di"
)

func main() {
	if err := di.Run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}
