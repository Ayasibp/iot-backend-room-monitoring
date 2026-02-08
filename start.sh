#!/bin/bash
# Quick start script for IoT Backend Room Monitoring

echo "================================"
echo "IoT Backend Room Monitoring"
echo "Quick Start Script"
echo "================================"
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo "⚠️  .env file not found. Creating from .env.example..."
    cp .env.example .env
    echo "✅ Created .env file. Please edit it with your database credentials."
    echo ""
    echo "Press Enter after you've configured .env, or Ctrl+C to exit..."
    read
fi

# Check if binary exists, if not build it
if [ ! -f bin/server ]; then
    echo "📦 Building application..."
    go build -o bin/server cmd/server/main.go
    if [ $? -ne 0 ]; then
        echo "❌ Build failed. Please check errors above."
        exit 1
    fi
    echo "✅ Build successful"
    echo ""
fi

echo "🚀 Starting server..."
echo ""
./bin/server
