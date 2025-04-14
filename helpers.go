package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

type ApiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Status  int         `json:"status"`
}

type Msg struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	Source     string `json:"source"`
	Error      string `json:"error"`
	StatusCode int    `json:"statusCode"`
}

func JSONResponse(w http.ResponseWriter, status int, payload any, errMsg any) {
	response := ApiResponse{
		Success: errMsg == nil,
		Status:  status,
	}

	if errMsg != nil {
		response.Error = errMsg
	} else {
		response.Data = payload
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonData)
	if err != nil {
		fmt.Println(err)
	}
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")

	if data == nil {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{}`))
		return
	}

	if msg, ok := data.(Msg); ok && msg.StatusCode > 0 {
		status = msg.StatusCode
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(jsonData)
}

func directoryExists(path string) error {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		info, err = os.Stat(path)
	}

	if err != nil {
		return fmt.Errorf("failed to check directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	return nil
}

func openFile(directory string, filename string) *os.File {

	err := directoryExists(fmt.Sprintf("%s%s", "./", directory))

	if err != nil {
		log.Printf("Failed to create directory %s", err)
	}

	var filepath = fmt.Sprintf("%s/%s", directory, filename)

	logFile, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		log.Printf("Failed to open or create file %s: %s", filename, err)
	}

	if err != nil {
		log.Printf("Failed to create %s: %s", filename, err)
	}

	return logFile
}

func closeFile(logFile *os.File) {
	if logFile == nil {
		return
	}

	if err := logFile.Close(); err != nil {
		log.Printf("Error closing file: %v", err)
	}
}

func Text(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)

	if _, err := fmt.Fprint(w, message); err != nil {
		log.Printf("Failed to write plain text response: %v", err)
	}
}

func Param(req *http.Request, key string) string {
	params, ok := req.Context().Value(routeParamsKey).(map[string]interface{})

	if !ok {
		return ""
	}

	if val, exists := params[key]; exists {
		return fmt.Sprintf("%v", val)
	}

	return ""
}

func Get(r *http.Request) url.Values {

	return r.URL.Query()
}

func Post(r *http.Request) (url.Values, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	return r.PostForm, nil
}

func Query(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}
