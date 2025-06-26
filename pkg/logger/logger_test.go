package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	// test production mode
	logger, err := New("production")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}

	// test development mode
	logger, err = New("development")
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
		expect map[string]string
	}{
		{
			zerologLogger.Info,
			"info message",
			[]interface{}{"key", "value"},
			map[string]string{
				"level":   "info",
				"key":     "value",
				"message": "info message",
			},
		},
		{
			zerologLogger.Error,
			"error message",
			[]interface{}{"error", "test"},
			map[string]string{
				"level":   "error",
				"error":   "test",
				"message": "error message",
			},
		},
		{
			zerologLogger.Warn,
			"warn message",
			nil,
			map[string]string{
				"level":   "warn",
				"message": "warn message",
			},
		},
		{
			zerologLogger.Debug,
			"debug message",
			[]interface{}{"key", "value"},
			map[string]string{
				"level":   "debug",
				"key":     "value",
				"message": "debug message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			buf.Reset()
			tt.method(tt.msg, tt.fields...)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log JSON: %v", err)
			}

			for key, want := range tt.expect {
				got, ok := logEntry[key]
				if !ok {
					t.Errorf("Missing expected key %q", key)
				} else if gotStr := toString(got); gotStr != want {
					t.Errorf("Value for key %q = %q, want %q", key, gotStr, want)
				}
			}
		})
	}
}

func toString(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	default:
		return ""
	}
}

/*
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
		{zerologLogger.Info, "info message", []interface{}{"key", "value"}, `"level":"info"`},
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
*/
