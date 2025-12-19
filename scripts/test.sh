#!/bin/bash
set -e

echo "Running tests..."
go test -v -race -coverprofile=coverage.out ./pkg/...

echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo "Coverage report generated: coverage.html"
