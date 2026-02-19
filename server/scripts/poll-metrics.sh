#!/bin/bash

# Poll /api/v1/metrics during load testing to see what's happening
# This bypasses Railway's log rate limit

URL="${1:-https://powerful-alignment-production-09a7.up.railway.app}"
INTERVAL="${2:-5}"

echo "Polling $URL/api/v1/metrics every ${INTERVAL} seconds"
echo "Press Ctrl+C to stop"
echo ""

while true; do
    TIMESTAMP=$(date +"%H:%M:%S")

    # Fetch metrics
    METRICS=$(curl -s "$URL/api/v1/metrics")

    # Parse key metrics using jq if available, otherwise show raw
    if command -v jq &> /dev/null; then
        echo "[$TIMESTAMP] ======================================"

        # Error counts
        TOTAL_5XX=$(echo "$METRICS" | jq -r '.errors.http_5xx')
        TOTAL_4XX=$(echo "$METRICS" | jq -r '.errors.http_4xx')
        TOTAL_400=$(echo "$METRICS" | jq -r '.errors.http_400')
        TOTAL_404=$(echo "$METRICS" | jq -r '.errors.http_404')
        TOTAL_429=$(echo "$METRICS" | jq -r '.errors.http_429')
        SLOW_REQ=$(echo "$METRICS" | jq -r '.errors.slow_requests')

        # Database stats
        IN_USE=$(echo "$METRICS" | jq -r '.database.in_use')
        IDLE=$(echo "$METRICS" | jq -r '.database.idle')
        UTIL=$(echo "$METRICS" | jq -r '.database.utilization_percent')
        WAIT_COUNT=$(echo "$METRICS" | jq -r '.database.wait_count')
        WAIT_MS=$(echo "$METRICS" | jq -r '.database.wait_duration_ms')

        echo "Errors:   5xx=$TOTAL_5XX  4xx=$TOTAL_4XX (400=$TOTAL_400, 404=$TOTAL_404, 429=$TOTAL_429)  Slow=$SLOW_REQ"
        echo "DB Pool:  InUse=$IN_USE  Idle=$IDLE  Util=${UTIL}%"

        if [ "$WAIT_COUNT" != "0" ]; then
            echo "⚠️  WAITING: $WAIT_COUNT requests waited ${WAIT_MS}ms for connections"
        fi

        if [ "$UTIL" -gt 70 ]; then
            echo "⚠️  HIGH CONNECTION POOL UTILIZATION: ${UTIL}%"
        fi

        if [ "$TOTAL_5XX" -gt 100 ]; then
            echo "🔥 ERRORS DETECTED: ${TOTAL_5XX} server errors"
        fi

        if [ "$TOTAL_429" -gt 0 ]; then
            echo "🚦 RATE LIMITING: ${TOTAL_429} requests rate limited (429)"
        fi

        if [ "$TOTAL_400" -gt 100 ]; then
            echo "⚠️  VALIDATION ERRORS: ${TOTAL_400} bad requests (400)"
        fi

        if [ "$TOTAL_404" -gt 100 ]; then
            echo "❓ NOT FOUND: ${TOTAL_404} requests to unknown endpoints (404)"
        fi

    else
        echo "[$TIMESTAMP] $METRICS" | python3 -m json.tool 2>/dev/null || echo "$METRICS"
    fi

    sleep "$INTERVAL"
done
