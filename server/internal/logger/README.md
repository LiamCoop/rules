# Logger Package

Unified logging package supporting both standard JSON output and OpenTelemetry (OTEL) integration.

## Features

- **Dual Mode Support**: JSON logging (default) or OpenTelemetry logging
- **Sampling**: Reduces log volume for errors/warnings while preserving metrics
- **Metrics Counters**: Track errors, warnings, and HTTP-specific metrics
- **Log Levels**: TRACE, DEBUG, INFO, WARN, ERROR, FATAL
- **HTTP Helpers**: Specialized functions for HTTP error tracking

---

## Quick Start

### Default Mode (JSON Logging)

No configuration needed. The logger automatically uses JSON output to stdout.

```go
import "github.com/liamcoop/rules/server/internal/logger"

logger.Info("Server started", "port", 8080)
logger.Error("Failed to connect", "error", err)
```

### OpenTelemetry Mode

Enable OpenTelemetry by setting the environment variable:

```bash
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=my-service
export OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
```

The logger will automatically:
- Connect to the OTEL collector
- Send logs via OTLP gRPC
- Include service name in all logs
- Provide a shutdown hook for graceful shutdown

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_ENABLED` | `false` | Set to `true` to enable OpenTelemetry logging |
| `OTEL_SERVICE_NAME` | `unknown-service` | Service name for OTEL (required if OTEL_ENABLED) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (OTEL default) | OTLP collector endpoint |
| `LOG_LEVEL` | `INFO` | Minimum log level: TRACE, DEBUG, INFO, WARN, ERROR, FATAL |
| `ERROR_SAMPLE_RATE` | `100` | Sampling rate: 1=all logs, 100=1%, 1000=0.1% |

---

## Logging Functions

### Basic Logging

```go
// Always logged (never sampled)
logger.Trace("Detailed trace message", "key", value)
logger.Debug("Debug information", "key", value)
logger.Info("Informational message", "key", value)

// Sampled (1 in ERROR_SAMPLE_RATE logged)
logger.Warn("Warning message", "key", value)
logger.Error("Error occurred", "error", err)

// Always logged, then exits
logger.Fatal("Fatal error", "error", err)
```

### HTTP-Specific Helpers

These functions increment metrics counters but don't produce log output:

```go
// Increment 5xx error counters
logger.ErrorHttp5xx()

// Increment 4xx warning counters
logger.WarnHttp4xx(404) // Tracks specific codes: 400, 404, 429

// Increment slow request counter
logger.WarnSlowRequest()

// Increment connection pool warning counter
logger.WarnConnPool()
```

---

## Sampling Behavior

**Why Sampling?**
In high-volume production environments, logging every error/warning can:
- Overwhelm log storage
- Increase costs
- Make it harder to find important issues

**How It Works:**
- `ERROR_SAMPLE_RATE=100` means 1 out of every 100 errors is logged
- Metrics counters are **always incremented** (no sampling)
- You get accurate metrics but reduced log volume

**Example:**
```go
// This increments TotalErrors every time
// But only logs 1% of the time (if ERROR_SAMPLE_RATE=100)
logger.Error("Database connection failed", "db", dbName)

// Metrics are always accurate
totalErrors := logger.TotalErrors.Load() // Actual count
```

**Levels that are sampled:**
- `logger.Warn()` - Sampled
- `logger.Error()` - Sampled

**Levels never sampled:**
- `logger.Trace()` - Always logged
- `logger.Debug()` - Always logged
- `logger.Info()` - Always logged
- `logger.Fatal()` - Always logged

---

## Metrics Counters

Access metrics for monitoring/health endpoints:

```go
import "github.com/liamcoop/rules/server/internal/logger"

func getMetrics() map[string]int64 {
    return map[string]int64{
        "total_errors":       logger.TotalErrors.Load(),
        "total_warnings":     logger.TotalWarnings.Load(),
        "total_5xx_errors":   logger.Total5xxErrors.Load(),
        "total_4xx_errors":   logger.Total4xxErrors.Load(),
        "total_400_errors":   logger.Total400Errors.Load(),
        "total_404_errors":   logger.Total404Errors.Load(),
        "total_429_errors":   logger.Total429Errors.Load(),
        "slow_requests":      logger.SlowRequests.Load(),
        "conn_pool_warnings": logger.ConnPoolWarnings.Load(),
    }
}
```

---

## Shutdown (OTEL Only)

When using OpenTelemetry, call `Shutdown()` during graceful shutdown:

```go
import (
    "context"
    "github.com/liamcoop/rules/server/internal/logger"
)

func main() {
    // Your application code...

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := logger.Shutdown(ctx); err != nil {
        log.Printf("Logger shutdown error: %v", err)
    }
}
```

**Note:** `Shutdown()` is safe to call even when OTEL is disabled (it's a no-op).

---

## Examples

### Example 1: Standard JSON Logging

```bash
# Start with default JSON logging
export LOG_LEVEL=INFO
export ERROR_SAMPLE_RATE=100
./server
```

Output:
```json
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"Server started","port":8080}
{"time":"2024-01-15T10:30:01Z","level":"ERROR","msg":"Connection failed","error":"timeout"}
```

### Example 2: OpenTelemetry Logging

```bash
# Enable OTEL
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=rules-engine
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export LOG_LEVEL=DEBUG
./server
```

Logs are sent to the OTEL collector instead of stdout.

### Example 3: High-Volume Production

```bash
# Reduce log volume with aggressive sampling
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME=production-api
export LOG_LEVEL=WARN
export ERROR_SAMPLE_RATE=1000  # Only log 0.1% of errors
./server
```

This configuration:
- Only logs WARN and above
- Samples errors at 0.1% (1 in 1000)
- Preserves all metrics counters

### Example 4: Development Mode

```bash
# See everything
export LOG_LEVEL=TRACE
export ERROR_SAMPLE_RATE=1  # Log all errors/warnings
./server
```

---

## Migration from Old Logger

If you were using the old logger, no changes needed! The API is backward compatible.

**Old code still works:**
```go
logger.Info("message", "key", value)
logger.Error("error", "err", err)
```

**New features available:**
```go
logger.Trace("detailed trace")  // New trace level
logger.Shutdown(ctx)             // New shutdown for OTEL
```

---

## OTEL Collector Configuration

Example collector config for receiving logs:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:

exporters:
  logging:
    loglevel: debug
  # Add your exporters (e.g., to Loki, Elasticsearch, etc.)

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]
```

---

## Troubleshooting

### OTEL Not Connecting

If you see: `Failed to setup OTEL logging, falling back to JSON`

**Checklist:**
1. Is `OTEL_ENABLED=true` set?
2. Is the OTEL collector running?
3. Is `OTEL_EXPORTER_OTLP_ENDPOINT` correct?
4. Can the server reach the collector (network/firewall)?

**Test Connection:**
```bash
# Check if collector is reachable
curl http://localhost:4317
```

### No Logs Appearing

**If using JSON mode:**
- Logs go to stdout - check your log aggregation

**If using OTEL mode:**
- Logs go to the collector - check collector logs
- Verify collector configuration
- Check service name matches: `OTEL_SERVICE_NAME`

### Too Many/Few Logs

Adjust sampling:
```bash
# More logs
export ERROR_SAMPLE_RATE=10  # 10% of errors

# Fewer logs
export ERROR_SAMPLE_RATE=1000  # 0.1% of errors

# All logs
export ERROR_SAMPLE_RATE=1
```

---

## Best Practices

1. **Use INFO for normal operations**
   ```go
   logger.Info("Request processed", "duration_ms", duration)
   ```

2. **Use WARN for recoverable issues**
   ```go
   logger.Warn("Retry successful after failure", "attempts", 3)
   ```

3. **Use ERROR for actual errors**
   ```go
   logger.Error("Failed to process request", "error", err)
   ```

4. **Include context in structured fields**
   ```go
   logger.Error("Database query failed",
       "query", queryName,
       "duration_ms", duration,
       "error", err,
   )
   ```

5. **Don't log sensitive data**
   ```go
   // BAD
   logger.Info("User login", "password", password)

   // GOOD
   logger.Info("User login", "username", username)
   ```

6. **Use HTTP helpers for HTTP errors**
   ```go
   if status >= 500 {
       logger.ErrorHttp5xx()
   } else if status >= 400 {
       logger.WarnHttp4xx(status)
   }
   ```

---

## Performance Notes

- Sampling reduces I/O overhead significantly
- Metrics counters use atomic operations (very fast)
- OTEL batching reduces network calls
- JSON mode is faster than OTEL (no network)
- Log level filtering happens before formatting (efficient)

---

## See Also

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/languages/go/)
- [slog Package](https://pkg.go.dev/log/slog)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
