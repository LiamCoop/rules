#!/bin/bash
# Setup script for OTEL logging

# OTEL Collector Configuration
# Adjust these based on your collector setup

# If collector is running in Docker on the same machine:
export OTEL_ENABLED=true
export OTEL_SERVICE_NAME="rules-engine"
export OTEL_EXPORTER_OTLP_ENDPOINT="host.docker.internal:4317"

# If collector is running in Docker on a different machine:
# export OTEL_EXPORTER_OTLP_ENDPOINT="collector-host:4317"

# If using Docker Desktop on Mac and accessing from host:
# export OTEL_EXPORTER_OTLP_ENDPOINT="host.docker.internal:4317"

# Log level (optional)
export LOG_LEVEL="INFO"

# Sampling (optional - default is 100)
export ERROR_SAMPLE_RATE="100"

echo "OTEL Environment Variables:"
echo "  OTEL_ENABLED=$OTEL_ENABLED"
echo "  OTEL_SERVICE_NAME=$OTEL_SERVICE_NAME"
echo "  OTEL_EXPORTER_OTLP_ENDPOINT=$OTEL_EXPORTER_OTLP_ENDPOINT"
echo "  LOG_LEVEL=$LOG_LEVEL"
echo "  ERROR_SAMPLE_RATE=$ERROR_SAMPLE_RATE"
echo ""
echo "To use these settings, run:"
echo "  source scripts/setup-otel.sh"
echo "  ./cmd/server/server"
