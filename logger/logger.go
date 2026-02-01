// Package logger provides a structured logging interface for the application.
package logger

import (
	"context"
)

// Logger defines the logging interface.
type Logger interface {
	// Debug logs a debug message.
	Debug(msg string, fields ...Field)

	// Info logs an info message.
	Info(msg string, fields ...Field)

	// Warn logs a warning message.
	Warn(msg string, fields ...Field)

	// Error logs an error message.
	Error(msg string, fields ...Field)

	// Fatal logs a fatal message and exits the application.
	Fatal(msg string, fields ...Field)

	// With creates a new logger with the given fields attached.
	With(fields ...Field) Logger

	// WithRequestID creates a new logger with a request_id field.
	WithRequestID(requestID string) Logger

	// WithContext creates a new logger with context.
	WithContext(ctx context.Context) Logger
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new Field with the given key and value.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field.
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with any value.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// NoOpLogger is a logger that does nothing.
type NoOpLogger struct{}

// NewNoOpLogger creates a new no-op logger.
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}

func (l *NoOpLogger) Debug(msg string, fields ...Field) {}
func (l *NoOpLogger) Info(msg string, fields ...Field)  {}
func (l *NoOpLogger) Warn(msg string, fields ...Field)  {}
func (l *NoOpLogger) Error(msg string, fields ...Field) {}
func (l *NoOpLogger) Fatal(msg string, fields ...Field) {}
func (l *NoOpLogger) With(fields ...Field) Logger       { return l }
func (l *NoOpLogger) WithRequestID(requestID string) Logger {
	return l
}
func (l *NoOpLogger) WithContext(ctx context.Context) Logger {
	return l
}
