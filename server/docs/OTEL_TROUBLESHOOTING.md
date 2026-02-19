# OpenTelemetry Troubleshooting Guide

## Quick Diagnosis

### Step 1: Test Collector Connectivity

```bash
./scripts/test-otel-connection.sh
```

If this fails, the collector isn't reachable. Skip to [Collector Not Reachable](#collector-not-reachable).

### Step 2: Set Environment Variables

```bash
source scripts/setup-otel.sh
```

Verify they're set:
```bash
env | grep OTEL
```

Should show:
```
OTEL_ENABLED=true
OTEL_SERVICE_NAME=rules-engine
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

### Step 3: Run Server with Enhanced Logging

```bash
./cmd/server/server 2>&1 | tee server.log
```

Look for these messages:

**✅ Success:**
```
OTEL Configuration:
  OTEL_ENABLED: true
  OTEL_SERVICE_NAME: rules-engine
  OTEL_EXPORTER_OTLP_ENDPOINT: localhost:4317
  Setting up OTEL exporter...
  ✓ Created OTEL resource
  Connecting to OTLP collector...
  ✓ Connected to OTLP collector
✅ OpenTelemetry logging enabled for service: rules-engine (sampling: 1/100)
```

**❌ Failure - Connection Issue:**
```
❌ Failed to setup OTEL logging, falling back to JSON: failed to create OTLP exporter...
```

**❌ Failure - Not Enabled:**
```
📝 JSON logging enabled (sampling: 1/100)
   To enable OTEL: export OTEL_ENABLED=true
```

---

## Common Issues

### 1. Collector Not Reachable

**Symptoms:**
- `Port 4317 is NOT accessible`
- Server falls back to JSON logging

**Solutions:**

#### a) Collector in Docker, Server on Host (Mac/Linux)

Use `host.docker.internal`:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="host.docker.internal:4317"
```

Or find the Docker IP:
```bash
docker inspect <container-id> | grep IPAddress
export OTEL_EXPORTER_OTLP_ENDPOINT="<ip>:4317"
```

#### b) Collector in Docker, Server Also in Docker

Use the service name from docker-compose:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="otel-collector:4317"
```

#### c) Check if Collector is Running

```bash
docker ps | grep otel
docker logs <collector-container-id>
```

#### d) Verify Port Mapping

```bash
docker port <collector-container-id>
```

Should show:
```
4317/tcp -> 0.0.0.0:4317
4318/tcp -> 0.0.0.0:4318
```

---

### 2. Logs Not Appearing in Grafana

**Even though connection succeeds, logs don't show up**

#### Check 1: Collector Configuration

The collector **must** be configured to receive logs. Check `otel-collector-config.yaml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:          # ← Must have gRPC enabled on 4317
        endpoint: 0.0.0.0:4317
      http:          # Optional, for HTTP on 4318
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  # Example: Export to Loki (for Grafana)
  loki:
    endpoint: http://loki:3100/loki/api/v1/push

  # Or logging exporter for debugging
  logging:
    loglevel: debug

service:
  pipelines:
    logs:            # ← CRITICAL: Must have a logs pipeline!
      receivers: [otlp]
      processors: [batch]
      exporters: [loki, logging]  # Add logging exporter to see logs in collector
```

**Key points:**
- ✅ Must have `receivers.otlp.protocols.grpc`
- ✅ Must have `service.pipelines.logs`
- ✅ Must export logs somewhere (Loki, logging, etc.)

#### Check 2: Verify Collector Receives Logs

Add a `logging` exporter to see logs in collector output:

```yaml
exporters:
  logging:
    loglevel: debug

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, loki]  # Add logging first for debugging
```

Then watch collector logs:
```bash
docker logs -f <collector-container-id>
```

You should see your application's logs appear.

#### Check 3: Verify Logs Reach Loki

If using Loki, check its logs:
```bash
docker logs -f <loki-container-id>
```

Query Loki directly:
```bash
curl 'http://localhost:3100/loki/api/v1/query?query={service_name="rules-engine"}'
```

#### Check 4: Grafana Data Source

In Grafana:
1. Go to Configuration → Data Sources
2. Click on Loki
3. Click "Test" - should be green
4. Try querying: `{service_name="rules-engine"}`

---

### 3. Environment Variables Not Set

**Symptoms:**
```
📝 JSON logging enabled
   To enable OTEL: export OTEL_ENABLED=true
```

**Solution:**
```bash
source scripts/setup-otel.sh
./cmd/server/server
```

**Verify:**
```bash
env | grep OTEL
```

---

### 4. gRPC Connection Errors

**Symptoms:**
```
failed to create OTLP exporter: context deadline exceeded
```

**Possible causes:**

1. **Firewall blocking connection**
   ```bash
   # Test TCP connection
   telnet localhost 4317
   ```

2. **Collector not listening on gRPC**
   - Check collector config has `grpc` enabled
   - Restart collector after config changes

3. **TLS/SSL mismatch**
   - OTLP by default uses insecure connection
   - If collector requires TLS, you need to configure it

---

### 5. Logs in Collector but Not in Grafana

**Symptoms:**
- Collector logs show: "Logs received"
- Grafana shows: No data

**Solutions:**

#### a) Check Loki is Receiving Logs

```bash
# Loki logs endpoint
curl http://localhost:3100/ready

# Query recent logs
curl 'http://localhost:3100/loki/api/v1/query?query={service_name="rules-engine"}&limit=10'
```

#### b) Check Label Mapping

Loki requires labels. Ensure your collector config maps attributes to labels:

```yaml
exporters:
  loki:
    endpoint: http://loki:3100/loki/api/v1/push
    labels:
      resource:
        service.name: "service_name"
      attributes:
        level: "level"
```

#### c) Check Time Range in Grafana

Logs might be there but outside your selected time range:
- Try "Last 5 minutes"
- Or "Last 1 hour"

---

## Verification Checklist

Use this checklist to verify everything is working:

- [ ] Collector is running: `docker ps | grep otel`
- [ ] Port 4317 is accessible: `./scripts/test-otel-connection.sh`
- [ ] Environment variables are set: `env | grep OTEL`
- [ ] OTEL_ENABLED=true
- [ ] Server starts successfully
- [ ] Server logs show "✅ OpenTelemetry logging enabled"
- [ ] Collector logs show received logs: `docker logs otel-collector`
- [ ] Loki is receiving logs: `curl http://localhost:3100/ready`
- [ ] Grafana can query Loki
- [ ] Grafana shows logs from rules-engine

---

## Example Collector Configuration

Here's a complete working example:

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024

  # Add resource attributes
  resource:
    attributes:
      - key: environment
        value: production
        action: insert

exporters:
  # Logging for debugging
  logging:
    loglevel: debug

  # Loki for Grafana
  loki:
    endpoint: http://loki:3100/loki/api/v1/push
    labels:
      resource:
        service.name: "service_name"
      attributes:
        level: "level"

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [logging, loki]

    # If you also want traces/metrics
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]

    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]
```

---

## Docker Compose Example

```yaml
version: '3.8'

services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "8888:8888"   # Prometheus metrics
      - "8889:8889"   # Prometheus exporter metrics
    depends_on:
      - loki

  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    depends_on:
      - loki
```

Start with:
```bash
docker-compose up -d
```

---

## Testing End-to-End

1. **Start the stack:**
   ```bash
   docker-compose up -d
   ```

2. **Verify collector is up:**
   ```bash
   docker logs otel-collector
   ```

3. **Set environment and run server:**
   ```bash
   source scripts/setup-otel.sh
   ./cmd/server/server
   ```

4. **Generate some logs:**
   ```bash
   # Make some API requests to generate logs
   curl http://localhost:8080/api/v1/health
   ```

5. **Check collector received them:**
   ```bash
   docker logs otel-collector | grep "rules-engine"
   ```

6. **Check Grafana:**
   - Open http://localhost:3000
   - Go to Explore
   - Select Loki data source
   - Query: `{service_name="rules-engine"}`

---

## Still Having Issues?

### Enable Debug Logging

```bash
export LOG_LEVEL=DEBUG
./cmd/server/server
```

### Check Collector Health

```bash
# Health check endpoint
curl http://localhost:8888/health

# Metrics
curl http://localhost:8888/metrics
```

### Capture Network Traffic

```bash
# Watch traffic to collector
sudo tcpdump -i any -n port 4317
```

### Review Collector Logs

```bash
# Follow collector logs
docker logs -f otel-collector 2>&1 | grep -i "error\|warn\|fail"
```

---

## Need More Help?

1. Check the collector logs for errors
2. Verify your collector configuration matches the schema
3. Ensure logs pipeline is configured
4. Test with the logging exporter first before adding Loki
5. Verify network connectivity between all components
