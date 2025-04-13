package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func panicMessage() []string {
	stackTrace := make([]byte, 4096)
	stackSize := runtime.Stack(stackTrace, false)
	regex := regexp.MustCompile(`([[:print:]]+\(.+?\))\s+(/[^:]+:\d+)`)
	matches := regex.FindAllStringSubmatch(string(stackTrace[:stackSize]), -1)

	var panicError []string

	if len(matches) >= 1 {
		if len(matches) > 3 {
			matches = matches[3:]
		}

		for i, pn := range matches {
			if i == 0 {
				if len(pn) >= 3 {
					panicError = append(panicError, pn[2])
				}
			}
		}
		panicError = append(panicError, "\n")
	} else {
		panicError = append(panicError, "No panic information found.")
	}

	return panicError
}

func Error(w http.ResponseWriter, req *http.Request, message string, err error) bool {
	if err != nil {
		logger(message, err, true, req)

		http.Error(w, message, http.StatusInternalServerError)

		return true
	}

	return false
}

func JSONError(w http.ResponseWriter, req *http.Request, message string, err error) bool {
	if err != nil {
		logger(message, err, true, req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   true,
			"message": message,
		})
		return true
	}
	return false
}

func logger(message any, err error, terminal bool, req *http.Request) {
	logFile := openFile("logs", (time.Now().Format("2006-01-02"))+".error.log")

	defer closeFile(logFile)

	errors := strings.Join(panicMessage(), "\n")

	var path, method string
	if req != nil {
		path = req.URL.Path
		method = req.Method
	} else {
		path = "unknown"
		method = "UNKNOWN"
	}

	logger := log.New(logFile, "", log.LstdFlags)
	logger.Printf("Panic occurred on URL %s | method [%s]\nError message: %s\n%s%s\n\n", path, method, message, errors, strings.Repeat("_", 95))

	terminalOutput(path, method, message, errors)
}

func logRequest(req *http.Request, start time.Time) {
	duration := time.Since(start)

	var d string
	switch {
	case duration >= time.Millisecond:
		d = fmt.Sprintf("%dms", duration.Milliseconds())
	case duration >= time.Microsecond:
		d = fmt.Sprintf("%dµs", duration.Microseconds())
	default:
		d = fmt.Sprintf("%dns", duration.Nanoseconds())
	}

	timestamp := colors("green", time.Now().Format("2006-01-02 15:04:05"))
	method := formatText(FormatText{
		color:      "black",
		background: "green",
		text:       fmt.Sprintf(" Method[%s] ", req.Method),
	})
	url := req.Host + req.URL.Path
	fmt.Printf("%s: %s %s in %s\n", timestamp, method, url, d)
}
