package logger

import "log"

var (
	Info  = log.New(log.Writer(), "INFO: ", log.LstdFlags)
	Warn  = log.New(log.Writer(), "WARN: ", log.LstdFlags)
	Error = log.New(log.Writer(), "ERROR: ", log.LstdFlags)
	Debug = log.New(log.Writer(), "DEBUG: ", log.LstdFlags)
)
