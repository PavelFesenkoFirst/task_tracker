package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PavelFesenkoFirst/task_tracker/internal/config"
	platformlogger "github.com/PavelFesenkoFirst/task_tracker/internal/platform/logger"
	mysqlplatform "github.com/PavelFesenkoFirst/task_tracker/internal/platform/mysql"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	logger := platformlogger.New(cfg.App.Env)

	db, err := mysqlplatform.New(cfg.MySQL)
	if err != nil {
		logger.Error("mysql connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("worker started", "concurrency", cfg.Worker.Concurrency, "queue", cfg.Worker.Queue)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			logger.Info("worker heartbeat")
		case <-stop:
			logger.Info("worker stopped")
			return
		}
	}
}
