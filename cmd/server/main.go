package main

import (
	"context"
	"os"

	"github.com/oziev02/help-desk/internal/app"
	"github.com/oziev02/help-desk/internal/config"
)

func main() {
	logger := app.NewLogger()
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	application, err := app.New(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("init app", "err", err)
		os.Exit(1)
	}
	if err := application.Run(context.Background()); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
