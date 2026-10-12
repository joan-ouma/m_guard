package logger

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

func Setup(logPath string) func() {
	if logPath == "" {
		logPath = "/var/log/fleet/fleet.log"
	}

	if dir := filepath.Dir(logPath); dir != "." && dir != "/" {
		_ = os.MkdirAll(dir, 0755)
	}

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open log file, falling back to stdout", "error", err)
		return func() {}
	}

	mw := io.MultiWriter(os.Stdout, logFile)
	handler := slog.NewJSONHandler(mw, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	log.SetFlags(0)
	log.SetOutput(mw)

	slog.Info("Fleet API backend started successfully", "log_file", logPath)

	return func() {
		_ = logFile.Sync()
		_ = logFile.Close()
	}
}
