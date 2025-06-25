package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	// Test production mode
	os.Setenv("LOG_ENV", "production")
	defer os.Unsetenv("LOG_ENV")

	logger, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}

	// Test development mode
	os.Setenv("LOG_ENV", "development")
	logger, err = New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestZerologLogger(t *testing.T) {
	var buf bytes.Buffer
	zerologLogger := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.DebugLevel),
	}

	tests := []struct {
		method func(string, ...interface{})
		msg    string
		fields []interface{}
		want   string
	}{
		{zerologLogger.Info, "info message", []interface{}{"key", "value"}, `"level":"info","key":"value","message":"info message"`},
		{zerologLogger.Error, "error message", []interface{}{"error", "test"}, `"level":"error","error":"test","message":"error message"`},
		{zerologLogger.Warn, "warn message", nil, `"level":"warn","message":"warn message"`},
		{zerologLogger.Debug, "debug message", []interface{}{"key", "value"}, `"level":"debug","key":"value","message":"debug message"`},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			buf.Reset()
			tt.method(tt.msg, tt.fields...)
			output := strings.TrimSpace(buf.String())
			if !strings.Contains(output, tt.want) {
				t.Errorf("Log output = %s, want contains %s", output, tt.want)
			}
		})
	}
}
