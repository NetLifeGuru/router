package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONResponse_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONResponse(rec, http.StatusOK, map[string]string{"msg": "ok"}, nil)

	var resp ApiResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !resp.Success || resp.Status != http.StatusOK || resp.Data == nil {
		t.Errorf("unexpected JSON response: %+v", resp)
	}
}

func TestJSONResponse_Error(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONResponse(rec, http.StatusBadRequest, nil, "error message")

	var resp ApiResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Success || resp.Error == nil || resp.Status != http.StatusBadRequest {
		t.Errorf("unexpected error JSON: %+v", resp)
	}
}

func TestJSON_WithMsg(t *testing.T) {
	rec := httptest.NewRecorder()
	data := Msg{Message: "hello", StatusCode: 201}
	JSON(rec, http.StatusOK, data)

	body := rec.Body.String()
	if !strings.Contains(body, "hello") {
		t.Errorf("expected body to contain %q, got %q", "hello", body)
	}

	if rec.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected status code 201, got %d", rec.Result().StatusCode)
	}
}

func TestText(t *testing.T) {
	rec := httptest.NewRecorder()
	Text(rec, http.StatusAccepted, "Hello, world!")

	result := rec.Result()
	body, _ := io.ReadAll(result.Body)

	if string(body) != "Hello, world!" {
		t.Errorf("unexpected text response body: %q", string(body))
	}
	if result.StatusCode != http.StatusAccepted {
		t.Errorf("unexpected status code: %d", result.StatusCode)
	}
}

func TestDirectoryExists_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "testdir_utils")
	defer os.RemoveAll(dir)

	if err := ensureDirectory(dir); err != nil {
		t.Fatalf("ensureDirectory failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected a directory, got something else")
	}
}

func TestOpenFile_CreatesFile(t *testing.T) {
	dir := "testlogs_utils"
	defer os.RemoveAll(dir)

	file := openFile(dir, "logfile.txt")
	if file == nil {
		t.Fatalf("file was not created")
	}
	defer closeFile(file)

	info, err := os.Stat(filepath.Join(dir, "logfile.txt"))
	if err != nil {
		t.Fatalf("file was not created correctly: %v", err)
	}
	if info.IsDir() {
		t.Errorf("expected a file, got directory")
	}
}

func TestCloseFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "closefile_test")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	closeFile(tmpFile)
}
