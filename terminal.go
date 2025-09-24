package router

import (
	"fmt"
	"time"
)

type FormatText struct {
	color      string
	background string
	text       string
}

var textColors = map[string]string{
	"black":   "\033[30m",
	"red":     "\033[31m",
	"green":   "\033[32m",
	"yellow":  "\033[33m",
	"blue":    "\033[34m",
	"magenta": "\033[35m",
	"cyan":    "\033[36m",
	"white":   "\033[37m",
	"gray":    "\033[90m",
	"default": "\033[39m",
}

var bgColors = map[string]string{
	"black":   "\033[40m",
	"red":     "\033[41m",
	"green":   "\033[42m",
	"yellow":  "\033[43m",
	"blue":    "\033[44m",
	"magenta": "\033[45m",
	"cyan":    "\033[46m",
	"white":   "\033[47m",
	"gray":    "\033[100m",
	"default": "\033[49m",
}

func getTextColor(color string) string {
	if val, ok := textColors[color]; ok {
		return val
	}
	return textColors["default"]
}

func getBgColor(color string) string {
	if val, ok := bgColors[color]; ok {
		return val
	}
	return bgColors["default"]
}

func colors(color string, text string) string {
	return fmt.Sprintf("%s%s\033[0m", getTextColor(color), text)
}

func formatText(data FormatText) string {
	return fmt.Sprintf("%s%s%s\033[0m", getTextColor(data.color), getBgColor(data.background), data.text)
}

func printServerInfo(serverName string, serverVersion string, port int) {
	info := fmt.Sprintf("\n› %s\n› %s\n",
		formatText(FormatText{color: "green", text: serverName + ` ` + serverVersion}),
		`Web servers is running on: `+fmt.Sprintf("http://localhost:%d", port))

	fmt.Println(info)
}

func terminalOutput(path string, method string, message any, errors string) {
	fmt.Printf("\n%s\n%s [%s]\n%s [%s]\n%s %s\n%s",
		getTextColor("green")+time.Now().Format("2006-01-02 15:04:05")+"\033[0m",
		colors("red", "Panic occurred on URL:"), path,
		colors("red", "Method:"), method,
		colors("red", "Error message:"), message, errors)
}

func Log(level string, message string, args ...any) {
	var color string

	switch level {
	case "INFO":
		color = "cyan"
	case "WARN":
		color = "yellow"
	case "ERROR":
		color = "red"
	case "DEBUG":
		color = "magenta"
	default:
		color = "default"
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)

	fmt.Printf("› %s %s %s\n",
		getTextColor("green")+timestamp+"\033[0m",
		colors(color, fmt.Sprintf("[%s]", level)),
		formattedMessage)
}
