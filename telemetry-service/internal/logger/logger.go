package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	zapLogger *zap.Logger
	once      sync.Once
)

// LogEntry represents a log entry from external services
type LogEntry struct {
	ServiceName string                 `json:"service_name"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
}

// InitLogger initializes the Zap logger
func InitLogger(level string, encoding string) error {
	var err error
	once.Do(func() {
		config := zap.Config{
			Level:       zap.NewAtomicLevelAt(getLogLevel(level)),
			Development: false,
			Encoding:    encoding,
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "timestamp",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller",
				MessageKey:     "message",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      []string{"stdout", "logs/telemetry.log"},
			ErrorOutputPaths: []string{"stderr", "logs/telemetry-error.log"},
		}

		// Create logs directory if it doesn't exist
		if err = os.MkdirAll("logs", 0755); err != nil {
			return
		}

		zapLogger, err = config.Build()
		if err != nil {
			return
		}
	})
	return err
}

// GetLogger returns the initialized Zap logger
func GetLogger() *zap.Logger {
	if zapLogger == nil {
		_ = InitLogger("info", "json")
	}
	return zapLogger
}

// LogFromService processes and logs entries from external services
func LogFromService(entry LogEntry) error {
	logger := GetLogger()

	// Add service name and trace info as fields
	fields := []zap.Field{
		zap.String("service_name", entry.ServiceName),
	}

	if entry.TraceID != "" {
		fields = append(fields, zap.String("trace_id", entry.TraceID))
	}
	if entry.SpanID != "" {
		fields = append(fields, zap.String("span_id", entry.SpanID))
	}

	// Add custom fields
	for key, value := range entry.Fields {
		fields = append(fields, zap.Any(key, value))
	}

	// Log based on level
	switch entry.Level {
	case "debug":
		logger.Debug(entry.Message, fields...)
	case "info":
		logger.Info(entry.Message, fields...)
	case "warn", "warning":
		logger.Warn(entry.Message, fields...)
	case "error":
		logger.Error(entry.Message, fields...)
	case "fatal":
		logger.Fatal(entry.Message, fields...)
	case "panic":
		logger.Panic(entry.Message, fields...)
	default:
		logger.Info(entry.Message, fields...)
	}

	return nil
}

// LogBatch processes multiple log entries at once
func LogBatch(entries []LogEntry) error {
	for _, entry := range entries {
		if err := LogFromService(entry); err != nil {
			return fmt.Errorf("failed to log entry: %w", err)
		}
	}
	return nil
}

// ParseLogEntry parses raw JSON into LogEntry
func ParseLogEntry(data []byte) (*LogEntry, error) {
	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse log entry: %w", err)
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	return &entry, nil
}

// getLogLevel converts string level to zapcore.Level
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sync flushes any buffered log entries
func Sync() error {
	if zapLogger != nil {
		return zapLogger.Sync()
	}
	return nil
}
