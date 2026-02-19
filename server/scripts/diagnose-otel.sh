#!/bin/bash
# Comprehensive OTEL diagnostic script

echo "========================================"
echo "OTEL Configuration Diagnosis"
echo "========================================"
echo ""

# 1. Check environment variables
echo "1. Environment Variables"
echo "   -----------------------"
if [ "$OTEL_ENABLED" = "true" ]; then
    echo "   ✅ OTEL_ENABLED: $OTEL_ENABLED"
else
    echo "   ❌ OTEL_ENABLED: ${OTEL_ENABLED:-not set}"
    echo "      Run: export OTEL_ENABLED=true"
fi

if [ -n "$OTEL_SERVICE_NAME" ]; then
    echo "   ✅ OTEL_SERVICE_NAME: $OTEL_SERVICE_NAME"
else
    echo "   ⚠️  OTEL_SERVICE_NAME: not set (will use 'unknown-service')"
    echo "      Run: export OTEL_SERVICE_NAME=rules-engine"
fi

ENDPOINT="${OTEL_EXPORTER_OTLP_ENDPOINT:-localhost:4317}"
echo "   ℹ️  OTEL_EXPORTER_OTLP_ENDPOINT: $ENDPOINT"

echo ""

# 2. Check collector connectivity
echo "2. Collector Connectivity"
echo "   ----------------------"
HOST=$(echo $ENDPOINT | cut -d: -f1)
PORT=$(echo $ENDPOINT | cut -d: -f2)

if command -v nc &> /dev/null; then
    if nc -zv -w 2 "$HOST" "$PORT" 2>&1 | grep -q succeeded; then
        echo "   ✅ Collector reachable at $ENDPOINT"
    else
        echo "   ❌ Collector NOT reachable at $ENDPOINT"
        echo "      Troubleshooting:"
        echo "      - Check if collector is running: docker ps | grep otel"
        echo "      - Check if using correct endpoint"
        echo "      - If collector in Docker: export OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317"
    fi
else
    echo "   ⚠️  Cannot test connectivity (nc not installed)"
fi

echo ""

# 3. Check if collector is running (if Docker is available)
echo "3. Collector Status"
echo "   ----------------"
if command -v docker &> /dev/null; then
    COLLECTOR=$(docker ps --filter "ancestor=otel/opentelemetry-collector" --format "{{.ID}}" 2>/dev/null | head -1)
    if [ -z "$COLLECTOR" ]; then
        COLLECTOR=$(docker ps | grep -i otel | awk '{print $1}' | head -1)
    fi

    if [ -n "$COLLECTOR" ]; then
        echo "   ✅ Collector container found: $COLLECTOR"
        echo "   Ports:"
        docker port "$COLLECTOR" 2>/dev/null | sed 's/^/      /'
    else
        echo "   ❌ No OTEL collector container found"
        echo "      Start with: docker run -p 4317:4317 otel/opentelemetry-collector-contrib"
    fi
else
    echo "   ⚠️  Docker not available - cannot check collector status"
fi

echo ""

# 4. Test server startup
echo "4. Quick Server Test"
echo "   -----------------"
if [ -f "./cmd/server/server" ]; then
    echo "   ✅ Server binary found"
    echo "   To test, run:"
    echo "      source scripts/setup-otel.sh"
    echo "      ./cmd/server/server"
    echo ""
    echo "   Look for: '✅ OpenTelemetry logging enabled'"
else
    echo "   ⚠️  Server binary not found"
    echo "      Build with: go build ./cmd/server"
fi

echo ""

# 5. Summary and next steps
echo "========================================"
echo "Summary & Next Steps"
echo "========================================"
echo ""

ERRORS=0
WARNINGS=0

if [ "$OTEL_ENABLED" != "true" ]; then
    ERRORS=$((ERRORS + 1))
fi

if nc -zv -w 2 "$HOST" "$PORT" 2>&1 | grep -q -v succeeded; then
    ERRORS=$((ERRORS + 1))
fi

if [ -z "$OTEL_SERVICE_NAME" ]; then
    WARNINGS=$((WARNINGS + 1))
fi

if [ $ERRORS -eq 0 ]; then
    echo "✅ Configuration looks good!"
    echo ""
    echo "To enable OTEL logging:"
    echo "  1. source scripts/setup-otel.sh"
    echo "  2. ./cmd/server/server"
    echo ""
    if [ $WARNINGS -gt 0 ]; then
        echo "⚠️  $WARNINGS warning(s) - check above for details"
    fi
else
    echo "❌ Found $ERRORS error(s) - fix the issues above"
    echo ""
    echo "Quick fixes:"
    echo "  export OTEL_ENABLED=true"
    echo "  export OTEL_SERVICE_NAME=rules-engine"
    echo "  export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317"
    echo ""
    echo "Or use the setup script:"
    echo "  source scripts/setup-otel.sh"
fi

echo ""
echo "For detailed troubleshooting, see:"
echo "  docs/OTEL_TROUBLESHOOTING.md"
