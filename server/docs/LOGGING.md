# Logging Configuration

## Overview

The application uses structured JSON logging with configurable log levels to reduce log spam while maintaining critical debugging information.

## Log Levels

Set via `LOG_LEVEL` environment variable:

```bash
# Railway dashboard or CLI
LOG_LEVEL=WARN  # Recommended for production (stays under 500 logs/sec limit)
LOG_LEVEL=ERROR # Only errors
LOG_LEVEL=INFO  # Default - includes startup and shutdown
LOG_LEVEL=DEBUG # Verbose - includes all operations
```

### Level Details

| Level | What Gets Logged |
|-------|------------------|
| **ERROR** | Panics, 5xx errors, database connection pool exhaustion, fatal errors |
| **WARN** | 4xx errors, slow requests (>1s), connection pool stress (>70% utilization), rule evaluation errors |
| **INFO** | Server startup/shutdown, tenant loading, successful requests |
| **DEBUG** | Everything - tenant loading details, all operations |

## What's Logged at Each Level

### ERROR (Recommended for Load Testing)
- Application panics
- HTTP 5xx errors (server errors)
- Database connection pool near exhaustion (>190/200 connections)
- Fatal startup errors

### WARN (Recommended for Production)
- HTTP 4xx errors (client errors)
- Slow HTTP requests (>1 second)
- Database connection pool under stress (>70% utilization)
- Rule evaluation errors
- Database connection pool wait times

### INFO (Default)
All of the above plus:
- Server startup/shutdown
- Number of tenants loaded
- Port binding

### DEBUG (Development Only)
All of the above plus:
- Tenant loading details
- Additional debugging context

## Railway Configuration

### Stay Under 500 Logs/Second Limit

```bash
# In Railway dashboard, add environment variable:
LOG_LEVEL=WARN
```

This configuration will only log:
- Errors and warnings
- Slow requests (>1s)
- Connection pool issues

**Expected log volume at 6k RPS**:
- ERROR only: ~10-50 logs/sec (errors only)
- WARN: ~50-200 logs/sec (errors + slow requests + connection warnings)
- INFO: 500+ logs/sec (may hit Railway limit)

## Log Format

All logs are JSON formatted for easy parsing:

```json
{
  "time": "2024-12-19T10:30:45Z",
  "level": "ERROR",
  "msg": "HTTP 5xx error",
  "method": "POST",
  "path": "/api/v1/evaluate",
  "status": 500,
  "duration_ms": 2345,
  "remote_addr": "192.168.1.1:12345"
}
```

## Connection Pool Monitoring

The application automatically monitors database connection pool health every 30 seconds.

### Normal Operation
No logs (connection pool healthy)

### Under Stress (>70% utilization)
```json
{
  "level": "WARN",
  "msg": "Database connection pool under stress",
  "in_use": 145,
  "idle": 20,
  "open": 165,
  "max_open": 200,
  "utilization_percent": 72,
  "wait_count": 150,
  "wait_duration_ms": 1200
}
```

**What this tells you**:
- `in_use`: Currently active connections
- `utilization_percent`: How close to max (>70% = warning)
- `wait_count`: Number of requests that waited for a connection
- `wait_duration_ms`: Total time spent waiting (bottleneck indicator)

### Critical (>95% utilization)
```json
{
  "level": "ERROR",
  "msg": "DATABASE CONNECTION POOL NEAR EXHAUSTION",
  "in_use": 195,
  "max_open": 200,
  "waiting": 500
}
```

**Action required**: Increase `MaxOpenConns` or reduce load

## Debugging "Connection Reset" Errors

If you see "connection reset by peer" in k6 output, check Railway logs for:

1. **PANIC recovered** - Application crashed
   ```json
   {"level":"ERROR","msg":"PANIC recovered","error":"runtime error: ..."}
   ```

2. **DATABASE CONNECTION POOL NEAR EXHAUSTION** - Too many concurrent requests
   ```json
   {"level":"ERROR","msg":"DATABASE CONNECTION POOL NEAR EXHAUSTION"}
   ```

3. **HTTP 5xx error** - Application errors
   ```json
   {"level":"ERROR","msg":"HTTP 5xx error","status":500}
   ```

4. No logs = Out of Memory (OOM) kill
   - Check Railway memory metrics
   - Application restarted without logging

## Quick Commands

### View Recent Errors Only
```bash
railway logs --service=your-service | grep '"level":"ERROR"'
```

### View Connection Pool Warnings
```bash
railway logs --service=your-service | grep 'connection pool'
```

### View Slow Requests
```bash
railway logs --service=your-service | grep 'Slow request'
```

### Count Logs Per Minute
```bash
railway logs --service=your-service | grep -c '"time"'
```

## Load Testing Recommendations

For load testing with 15k VUs:

```bash
# Set Railway environment variables:
LOG_LEVEL=WARN  # Only warnings and errors

# This will log:
# - All 5xx errors (server failures)
# - All 4xx errors (client errors)
# - Slow requests >1s
# - Connection pool stress
# - Panics

# Expected: 50-200 logs/sec even at 15k RPS
```

After the test, check for patterns:
- Many "Slow request" logs → Latency bottleneck
- Many "connection pool" warnings → Need more connections
- Many "PANIC" errors → Application bug under load
- Many "5xx error" logs → Database or application errors

## Summary

| Use Case | LOG_LEVEL | Expected Logs/Sec |
|----------|-----------|-------------------|
| Development | DEBUG | 1000+ |
| Production | WARN | 50-200 |
| Load Testing | WARN | 100-300 |
| Incident Debug | ERROR | 10-100 |
| Extreme Load Test | ERROR | 50-150 |
