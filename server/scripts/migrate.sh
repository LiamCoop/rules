#!/bin/bash

# Script to run database migrations
# Loads DATABASE_URL from .env file if it exists

set -e

# Load .env file if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "ERROR: DATABASE_URL is not set"
    echo "Please set DATABASE_URL environment variable or create a .env file"
    exit 1
fi

# Run migration command
COMMAND=${1:-up}

echo "Running migration command: $COMMAND"
echo "Database URL: ${DATABASE_URL%%@*}@***" # Hide password in logs

go run cmd/migrate/main.go -command "$COMMAND" "${@:2}"
