package logger

import (
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

var Logger *slog.Logger
var errorSampleRate int32 = 100 // Log 1 out of every 100 errors by default

// Error counters for metrics endpoint
var (
	TotalErrors     atomic.Int64
	TotalWarnings   atomic.Int64
	Total5xxErrors  atomic.Int64
	Total4xxErrors  atomic.Int64
	SlowRequests    atomic.Int64
	ConnPoolWarnings atomic.Int64
)

func init() {
	// Get log level from environment variable (default: INFO)
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "INFO"
	}

	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Get error sample rate (default: 100 = 1%)
	if sampleStr := os.Getenv("ERROR_SAMPLE_RATE"); sampleStr != "" {
		if rate, err := strconv.Atoi(sampleStr); err == nil && rate > 0 {
			atomic.StoreInt32(&errorSampleRate, int32(rate))
		}
	}

	// Create handler with JSON formatting for production
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Use JSON handler for Railway (easier to parse)
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)

	slog.SetDefault(Logger)
}

// Helper functions for common logging patterns
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	TotalWarnings.Add(1)
	// Sample warnings to reduce log spam
	if shouldSample() {
		Logger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	TotalErrors.Add(1)
	// Sample errors to reduce log spam
	if shouldSample() {
		Logger.Error(msg, args...)
	}
}

func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Fatal logs an error and exits (always logged, never sampled)
func Fatal(msg string, args ...any) {
	Logger.Error(msg, args...)
	os.Exit(1)
}

// ErrorHttp logs HTTP errors with sampling and increments counters
func ErrorHttp5xx() {
	Total5xxErrors.Add(1)
	TotalErrors.Add(1)
}

func WarnHttp4xx() {
	Total4xxErrors.Add(1)
	TotalWarnings.Add(1)
}

func WarnSlowRequest() {
	SlowRequests.Add(1)
	TotalWarnings.Add(1)
}

func WarnConnPool() {
	ConnPoolWarnings.Add(1)
	TotalWarnings.Add(1)
}

// shouldSample returns true if we should log this message
// Uses sampling to reduce log volume (1 out of every N messages)
func shouldSample() bool {
	rate := atomic.LoadInt32(&errorSampleRate)
	if rate <= 1 {
		return true // Always log if rate is 1 or less
	}
	return rand.Intn(int(rate)) == 0
}
