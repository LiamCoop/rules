# Logger Migration Summary

## What Changed

We've unified the logging package to support both standard JSON logging and OpenTelemetry (OTEL) in a single implementation.

### Files Modified

- ✅ **logger.go** - Unified implementation (replaced)
- ❌ **logger2.go** - Removed (merged into logger.go)
- ✅ **README.md** - New comprehensive documentation

### Key Features

#### 1. **Backward Compatible** ✓
All existing code continues to work without changes:
```go
logger.Info("message", "key", value)
logger.Error("error", "err", err)
logger.Warn("warning", "key", value)
```

#### 2. **OpenTelemetry Support** ✓
Enable with environment variable:
```bash
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=my-service
```

#### 3. **Sampling Preserved** ✓
Reduces log volume while preserving metrics:
- `ERROR_SAMPLE_RATE=100` → logs 1% of errors/warnings
- Metrics counters **always** incremented
- Works with both JSON and OTEL modes

#### 4. **New Features Added** ✓
- `logger.Trace()` - New trace level
- `logger.Shutdown(ctx)` - Graceful OTEL shutdown
- `logger.ParseLevel()` - Parse level from string
- Better log level support (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)

#### 5. **All Existing Features Preserved** ✓
- Metrics counters (TotalErrors, TotalWarnings, etc.)
- HTTP helpers (ErrorHttp5xx, WarnHttp4xx, etc.)
- Sampling logic (shouldSample)
- Environment variable configuration

---

## Migration Guide

### No Changes Required!

Your existing code continues to work as-is. The logger is backward compatible.

### Optional: Enable OpenTelemetry

If you want to send logs to an OTEL collector:

```bash
# Set these environment variables
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=rules-engine
export OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317

# Optional: Add shutdown in main.go
defer logger.Shutdown(context.Background())
```

---

## Environment Variables

### Current Variables (Still Supported)

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `INFO` | Minimum log level |
| `ERROR_SAMPLE_RATE` | `100` | Sampling rate (1 = all logs, 100 = 1%, 1000 = 0.1%) |

### New Variables (OTEL Support)

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_ENABLED` | `false` | Enable OpenTelemetry logging |
| `OTEL_SERVICE_NAME` | `unknown-service` | Service name for OTEL |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (standard) | OTLP collector endpoint |

---

## Testing

### Test JSON Mode (Default)

```bash
# No configuration needed - this is the default
go run ./cmd/server/main.go
```

**Expected Output:**
```
JSON logging enabled (sampling: 1/100)
```

Logs will appear as JSON on stdout.

### Test OTEL Mode

```bash
# Set environment variables
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=test-service
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

# Run server
go run ./cmd/server/main.go
```

**Expected Output:**
```
OpenTelemetry logging enabled for service: test-service (sampling: 1/100)
```

**If OTEL collector is not running:**
```
Failed to setup OTEL logging, falling back to JSON: ...
JSON logging enabled (sampling: 1/100)
```

The server continues to work with JSON logging as a fallback.

---

## What's Different?

### Sampling Behavior

**Before:** Sampling always active (ERROR_SAMPLE_RATE environment variable)

**After:** Sampling still active and works the same way
- ✅ Works with JSON mode
- ✅ Works with OTEL mode
- ✅ Metrics always accurate
- ✅ Reduces log volume

### Log Levels

**Before:** DEBUG, INFO, WARN, ERROR

**After:** TRACE, DEBUG, INFO, WARN, ERROR, FATAL
- Added TRACE level for very detailed logging
- Added FATAL level (logs and exits)

### New Capabilities

1. **Dynamic Backend**: Switch between JSON and OTEL without code changes
2. **Graceful Shutdown**: `logger.Shutdown(ctx)` for OTEL cleanup
3. **Better Observability**: OTEL integration for modern observability stacks
4. **Fallback Support**: Automatically falls back to JSON if OTEL fails

---

## Production Deployment

### Recommended Configuration

#### Development
```bash
export LOG_LEVEL=DEBUG
export ERROR_SAMPLE_RATE=1  # Log everything
```

#### Staging
```bash
export LOG_LEVEL=INFO
export ERROR_SAMPLE_RATE=10  # 10% of errors
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=rules-engine-staging
```

#### Production
```bash
export LOG_LEVEL=INFO
export ERROR_SAMPLE_RATE=100  # 1% of errors (or higher for less volume)
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=rules-engine-production
export OTEL_EXPORTER_OTLP_ENDPOINT=https://otel-collector.production:4317
```

---

## Rollback Plan

If you need to rollback, restore the old logger.go:

```bash
git checkout HEAD~1 -- internal/logger/logger.go
git restore internal/logger/logger2.go
go build ./...
```

However, this shouldn't be necessary as the new implementation is backward compatible.

---

## Questions?

See the [README.md](README.md) for detailed documentation including:
- All logging functions
- Configuration options
- Examples
- Troubleshooting
- Best practices
