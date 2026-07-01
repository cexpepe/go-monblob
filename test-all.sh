#!/bin/bash
set -euo pipefail

echo "==> Running unit tests..."
go test -v -race -coverprofile=coverage.out ./...

echo "==> Running benchmarks..."
go test -bench=. -benchmem ./...

echo "==> Running fuzz tests (30s each)..."
go test -fuzz=^FuzzParse$ -fuzztime=30s
go test -fuzz=^FuzzParsePrefix$ -fuzztime=30s

echo "==> Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report saved to coverage.html"

echo "==> All tests passed successfully!"