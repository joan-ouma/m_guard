package logger

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Setup(logPath string) func() {
	if logPath == "" {
		logPath = "/var/log/fleet/fleet.log"
	}

	if dir := filepath.Dir(logPath); dir != "." && dir != "/" {
		_ = os.MkdirAll(dir, 0755)
	}

	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    50,
		MaxBackups: 10,
		MaxAge:     30,
		LocalTime:  true,
		Compress:   true,
	}

	mw := io.MultiWriter(os.Stdout, rotator)
	handler := slog.NewJSONHandler(mw, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	log.SetFlags(0)
	log.SetOutput(mw)

	slog.Info("Fleet API backend started successfully", "log_file", logPath)

	return func() {
		_ = rotator.Close()
	}
}
