package log

import (
	"io"
	"log"
	"os"
	"strings"
)

// Log level constants
const (
	LevelTrace = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

// currentLevel holds the configured log level
var currentLevel = LevelInfo

type MyLoggerOptions struct {
	// if we output to  stdout
	Stdout bool
	// Path of the file , if present log to it
	Path string
	// What level to log
	Level string
}

func ConfigureMyLogger(options *MyLoggerOptions) {
	var writer io.Writer

	if options.Path != "" {
		logfile, err := os.OpenFile(options.Path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			panic(err)
		}
		if options.Stdout {
			writer = io.MultiWriter(logfile, os.Stdout)
		} else {
			writer = logfile
		}
	} else if options.Stdout {
		writer = os.Stdout
	} else {
		writer, _ = os.OpenFile(os.DevNull, os.O_APPEND, 0666)
	}

	log.SetOutput(writer)

	// Parse and set the log level
	switch strings.ToUpper(options.Level) {
	case "TRACE":
		currentLevel = LevelTrace
	case "DEBUG":
		currentLevel = LevelDebug
	case "INFO":
		currentLevel = LevelInfo
	case "WARN":
		currentLevel = LevelWarn
	case "ERROR":
		currentLevel = LevelError
	default:
		currentLevel = LevelInfo
	}
}

// Debug logs a message at DEBUG level
func Debug(format string, v ...interface{}) {
	if currentLevel <= LevelDebug {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Warn logs a message at WARN level
func Warn(format string, v ...interface{}) {
	if currentLevel <= LevelWarn {
		log.Printf("[WARN] "+format, v...)
	}
}
