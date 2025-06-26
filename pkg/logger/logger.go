package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// ZerologLogger implements the Logger interface using zerolog
type ZerologLogger struct {
	logger zerolog.Logger
}

// New initializes a new zerolog-based logger
func New(LogEnv string) (Logger, error) {
	var output io.Writer = os.Stdout
	if LogEnv == "development" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}
	}

	logger := zerolog.New(output).With().Timestamp().Logger()
	if LogEnv == "development" {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	return &ZerologLogger{logger: logger}, nil
}

// Info logs an info-level message
func (l *ZerologLogger) Info(msg string, fields ...interface{}) {
	l.logWithFields(l.logger.Info(), msg, fields...)
}

// Error logs an error-level message
func (l *ZerologLogger) Error(msg string, fields ...interface{}) {
	l.logWithFields(l.logger.Error(), msg, fields...)
}

// Warn logs a warn-level message
func (l *ZerologLogger) Warn(msg string, fields ...interface{}) {
	l.logWithFields(l.logger.Warn(), msg, fields...)
}

// Debug logs a debug-level message
func (l *ZerologLogger) Debug(msg string, fields ...interface{}) {
	l.logWithFields(l.logger.Debug(), msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *ZerologLogger) Fatal(msg string, fields ...interface{}) {
	l.logWithFields(l.logger.Fatal(), msg, fields...)
}


// Handles converting fields from ...interface{} to actual key-value pairs
func (l *ZerologLogger) logWithFields(event *zerolog.Event, msg string, fields ...interface{}) {
	if len(fields)%2 != 0 {
		fields = append(fields, "(missing)")
	}
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, fields[i+1])
	}
	event.Msg(msg)
}