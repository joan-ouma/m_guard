package logger

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetup_WritesJSONToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logtest-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFilePath := filepath.Join(tmpDir, "fleet.log")
	cleanup := Setup(logFilePath)
	defer cleanup()

	slog.Info("test message", "component", "auth", "status", "ok")

	cleanup()

	data, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := strings.TrimSpace(string(data))
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		t.Fatalf("expected log lines, got none")
	}

	var found bool
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid json line: %s", line)
		}
		if entry["msg"] == "test message" {
			found = true
			if entry["component"] != "auth" {
				t.Errorf("expected component auth, got %v", entry["component"])
			}
		}
	}

	if !found {
		t.Errorf("did not find test message in log file: %s", content)
	}
}
