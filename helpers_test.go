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

	if !resp.Success || resp.Status != 200 || resp.Data == nil {
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

	if resp.Success || resp.Error == nil || resp.Status != 400 {
		t.Errorf("unexpected error JSON: %+v", resp)
	}
}

func TestJSON_WithMsg(t *testing.T) {
	rec := httptest.NewRecorder()
	data := Msg{Message: "hello", StatusCode: 201}
	JSON(rec, http.StatusOK, data)

	body := rec.Body.String()
	if !strings.Contains(body, "hello") || rec.Result().StatusCode != 201 {
		t.Errorf("unexpected Msg JSON output: %s", body)
	}
}

func TestText(t *testing.T) {
	rec := httptest.NewRecorder()
	Text(rec, http.StatusAccepted, "Hello, world!")

	result := rec.Result()
	body, _ := io.ReadAll(result.Body)

	if string(body) != "Hello, world!" || result.StatusCode != http.StatusAccepted {
		t.Errorf("unexpected text response: %s", string(body))
	}
}

func TestDirectoryExists_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "testdir_utils")
	defer os.RemoveAll(dir)

	err := directoryExists(dir)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Errorf("directory was not created properly")
	}
}

func TestOpenFile_CreatesFile(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "/")
	defer os.RemoveAll(dir)

	file := openFile(dir, "logfile.txt")
	if file == nil {
		t.Fatalf("file was not created")
	}
	defer closeFile(file)

	info, err := os.Stat(filepath.Join(dir, "logfile.txt"))
	if err != nil || info.IsDir() {
		t.Errorf("file was not created correctly")
	}
}

func TestCloseFile(t *testing.T) {
	defer os.RemoveAll("./var")

	tmpfile, err := os.CreateTemp("", "closefile_test")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}

	closeFile(tmpfile)
}
