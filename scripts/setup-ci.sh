#!/bin/bash

# CI Setup Script
# This script helps set up the local development environment for CI

set -e

echo "🚀 Setting up CI environment..."
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

echo "✅ Go version: $(go version)"
echo ""

# Install golangci-lint
echo "📦 Installing golangci-lint..."
if ! command -v golangci-lint &> /dev/null; then
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
    echo "✅ golangci-lint installed"
else
    echo "✅ golangci-lint already installed"
fi
echo ""

# Install golang-migrate
echo "📦 Installing golang-migrate..."
if ! command -v migrate &> /dev/null; then
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    echo "✅ golang-migrate installed"
else
    echo "✅ golang-migrate already installed"
fi
echo ""

# Install air (hot reload)
echo "📦 Installing air..."
if ! command -v air &> /dev/null; then
    go install github.com/cosmtrek/air@latest
    echo "✅ air installed"
else
    echo "✅ air already installed"
fi
echo ""

# Install dependencies
echo "📦 Installing Go dependencies..."
go mod download
echo "✅ Dependencies installed"
echo ""

# Setup pre-commit (optional)
if command -v python3 &> /dev/null || command -v pip &> /dev/null; then
    echo "🔧 Pre-commit hooks available"
    echo "   Install: pip install pre-commit && pre-commit install"
else
    echo "⚠️  Python not found - pre-commit hooks not available (optional)"
fi
echo ""

# Run CI checks
echo "🧪 Running CI checks..."
make ci
echo ""

echo "✅ CI environment setup complete!"
echo ""
echo "Available commands:"
echo "  make ci          - Run all CI checks"
echo "  make ci-lint     - Run linter only"
echo "  make ci-test     - Run tests only"
echo "  make ci-build    - Build only"
echo "  make help        - Show all available commands"
echo ""
