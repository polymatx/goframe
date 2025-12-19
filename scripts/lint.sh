#!/bin/bash
set -e

echo "Running linters..."

if ! command -v golangci-lint &> /dev/null; then
    echo "golangci-lint not found. Installing..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

golangci-lint run ./...

echo "Linting complete!"
