#!/bin/bash
set -e

echo "Building all packages..."
go build -v ./...

echo "Build successful!"
