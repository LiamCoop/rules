#!/bin/bash
# Test connectivity to OTEL collector

echo "Testing OTEL Collector Connectivity..."
echo ""

# Check if collector endpoint is set
ENDPOINT="${OTEL_EXPORTER_OTLP_ENDPOINT:-localhost:4317}"
HOST=$(echo $ENDPOINT | cut -d: -f1)
PORT=$(echo $ENDPOINT | cut -d: -f2)

echo "Testing connection to: $ENDPOINT"
echo "  Host: $HOST"
echo "  Port: $PORT"
echo ""

# Test if port is accessible
if command -v nc &> /dev/null; then
    echo "Testing with netcat..."
    if nc -zv -w 5 "$HOST" "$PORT" 2>&1 | grep -q succeeded; then
        echo "✅ Port $PORT is open on $HOST"
    else
        echo "❌ Port $PORT is NOT accessible on $HOST"
        echo ""
        echo "Troubleshooting steps:"
        echo "1. Check if collector is running:"
        echo "   docker ps | grep otel"
        echo ""
        echo "2. Check if ports are exposed:"
        echo "   docker port <container-id>"
        echo ""
        echo "3. If collector is in Docker, try:"
        echo "   export OTEL_EXPORTER_OTLP_ENDPOINT=\"host.docker.internal:4317\""
        exit 1
    fi
elif command -v telnet &> /dev/null; then
    echo "Testing with telnet..."
    (echo quit; sleep 1) | telnet "$HOST" "$PORT" 2>&1 | grep -q Connected
    if [ $? -eq 0 ]; then
        echo "✅ Port $PORT is open on $HOST"
    else
        echo "❌ Port $PORT is NOT accessible on $HOST"
        exit 1
    fi
else
    echo "⚠️  Neither nc nor telnet found - cannot test connectivity"
    echo "   Install netcat: brew install netcat (Mac) or apt-get install netcat (Linux)"
fi

echo ""
echo "Next steps:"
echo "1. Source the environment setup:"
echo "   source scripts/setup-otel.sh"
echo ""
echo "2. Run your server:"
echo "   ./cmd/server/server"
echo ""
echo "3. Watch for OTEL connection messages in the logs"
