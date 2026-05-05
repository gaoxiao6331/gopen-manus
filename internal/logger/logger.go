package logger

import "log"

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorOrange = "\033[38;5;208m"
	colorRed    = "\033[31m"
	colorBlue   = "\033[34m"
)

func coloredPrefix(level, color string) string {
	return color + level + colorReset + ": "
}

var (
	Info  = log.New(log.Writer(), coloredPrefix("INFO", colorGreen), log.LstdFlags)
	Warn  = log.New(log.Writer(), coloredPrefix("WARN", colorOrange), log.LstdFlags)
	Error = log.New(log.Writer(), coloredPrefix("ERROR", colorRed), log.LstdFlags)
	Debug = log.New(log.Writer(), coloredPrefix("DEBUG", colorBlue), log.LstdFlags)
)
