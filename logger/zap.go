// Package logger provides Zap implementation of the Logger interface.
package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"memci/config"
)

// ZapLogger is the Zap implementation of Logger.
type ZapLogger struct {
	logger *zap.Logger
}

// New creates a new logger based on the provided configuration.
func New(cfg config.LogConfig) (Logger, error) {
	// Configure encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	// Determine writers
	var writers []zapcore.WriteSyncer

	switch cfg.Output {
	case "console":
		writers = append(writers, zapcore.AddSync(os.Stdout))
	case "file":
		writer, err := newFileWriter(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %w", err)
		}
		writers = append(writers, writer)
	case "both":
		writers = append(writers, zapcore.AddSync(os.Stdout))
		writer, err := newFileWriter(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %w", err)
		}
		writers = append(writers, writer)
	default:
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// Create core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writers...),
		getLogLevel(cfg.Level),
	)

	// Create logger
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return &ZapLogger{
		logger: logger,
	}, nil
}

// newFileWriter creates a rotating file writer.
func newFileWriter(cfg config.LogConfig) (zapcore.WriteSyncer, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Use lumberjack for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   true,
	}

	return zapcore.AddSync(lumberjackLogger), nil
}

// getLogLevel converts string level to zapcore.Level.
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// toZapFields converts logger fields to zap fields.
func (l *ZapLogger) toZapFields(fields ...Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = zap.Any(f.Key, f.Value)
	}
	return zapFields
}

// Debug logs a debug message.
func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, l.toZapFields(fields...)...)
}

// Info logs an info message.
func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, l.toZapFields(fields...)...)
}

// Warn logs a warning message.
func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, l.toZapFields(fields...)...)
}

// Error logs an error message.
func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, l.toZapFields(fields...)...)
}

// Fatal logs a fatal message and exits.
func (l *ZapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, l.toZapFields(fields...)...)
	os.Exit(1)
}

// With creates a new logger with the given fields attached.
func (l *ZapLogger) With(fields ...Field) Logger {
	return &ZapLogger{
		logger: l.logger.With(l.toZapFields(fields...)...),
	}
}

// WithRequestID creates a new logger with a request_id field.
func (l *ZapLogger) WithRequestID(requestID string) Logger {
	return l.With(F("request_id", requestID))
}

// WithContext creates a new logger with context.
func (l *ZapLogger) WithContext(ctx context.Context) Logger {
	// Extract request_id from context if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		return l.WithRequestID(requestID.(string))
	}
	return l
}

// Sync flushes any buffered log entries.
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}
