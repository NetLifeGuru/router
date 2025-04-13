package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPanicMessage(t *testing.T) {
	msg := panicMessage()
	if len(msg) == 0 {
		t.Errorf("expected some panic message output")
	}
}

func TestLogger_WritesToLogFile(t *testing.T) {

	req, _ := http.NewRequest("GET", "/", nil)
	req.Host = "localhost"

	logger("simulated panic", nil, false, req)

	filename := time.Now().Format("2006-01-02") + ".error.log"
	path := filepath.Join("logs", filename)

	defer os.RemoveAll("logs")

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		t.Fatalf("expected log file to be created: %s", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read log file: %s", err)
	}

	if !strings.Contains(string(data), "Panic occurred on URL / | method [GET]") {
		t.Errorf("log content missing expected text:\n%s", string(data))
	}
}
