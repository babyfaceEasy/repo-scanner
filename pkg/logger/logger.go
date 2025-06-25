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
func New() (Logger, error) {
	var output io.Writer = os.Stdout
	if os.Getenv("LOG_ENV") == "development" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}
	}

	logger := zerolog.New(output).With().Timestamp().Logger()
	if os.Getenv("LOG_ENV") == "development" {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	return &ZerologLogger{logger: logger}, nil
}

// Info logs an info-level message
func (l *ZerologLogger) Info(msg string, fields ...interface{}) {
	l.logger.Info().Fields(fields).Msg(msg)
}

// Error logs an error-level message
func (l *ZerologLogger) Error(msg string, fields ...interface{}) {
	l.logger.Error().Fields(fields).Msg(msg)
}

// Warn logs a warn-level message
func (l *ZerologLogger) Warn(msg string, fields ...interface{}) {
	l.logger.Warn().Fields(fields).Msg(msg)
}

// Debug logs a debug-level message
func (l *ZerologLogger) Debug(msg string, fields ...interface{}) {
	l.logger.Debug().Fields(fields).Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *ZerologLogger) Fatal(msg string, fields ...interface{}) {
	l.logger.Fatal().Fields(fields).Msg(msg)
}
