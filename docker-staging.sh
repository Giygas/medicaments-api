#!/bin/bash
# Quick start script for Docker staging environment

set -e

echo "🐳 Medicaments API - Docker Staging Quick Start"
echo "================================================"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed or not in PATH"
    echo "Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Error: Docker Compose is not installed or not in PATH"
    echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi

# Create logs directory if it doesn't exist
if [ ! -d "logs" ]; then
    echo "📁 Creating logs directory..."
    mkdir -p logs
    chmod 755 logs
fi

# Check if .env.docker exists
if [ ! -f ".env.docker" ]; then
    echo "❌ Error: .env.docker file not found"
    echo "Please create .env.docker from .env.example"
    exit 1
fi

echo "✅ Docker and Docker Compose are installed"
echo "📝 Using .env.docker configuration"
echo "🌐 API will be available at: http://localhost:8030"
echo ""

# Ask user what to do
echo "What would you like to do?"
echo "1) Build and start (recommended for first time)"
echo "2) Start (if already built)"
echo "3) Stop"
echo "4) View logs"
echo "5) Restart"
echo "6) Remove everything"
echo ""

read -p "Enter choice [1-6]: " choice

case $choice in
    1)
        echo ""
        echo "🔨 Building Docker image..."
        docker-compose build
        echo ""
        echo "🚀 Starting container..."
        docker-compose up -d
        echo ""
        echo "✅ Container started successfully!"
        echo ""
        echo "⏳ Waiting for application to be ready (this may take 30-60 seconds)..."
        echo ""
        
        # Wait for health check to pass
        max_attempts=30
        attempt=1
        while [ $attempt -le $max_attempts ]; do
            if curl -sf http://localhost:8030/health > /dev/null 2>&1; then
                echo "✅ Application is ready!"
                echo ""
                echo "📊 Health check:"
                curl -s http://localhost:8030/health | python3 -m json.tool 2>/dev/null || curl -s http://localhost:8030/health
                echo ""
                echo "🔍 View logs: docker-compose logs -f"
                echo "🛑 Stop: docker-compose down"
                break
            fi
            echo "⏳ Waiting... ($attempt/$max_attempts)"
            sleep 2
            attempt=$((attempt + 1))
        done
        
        if [ $attempt -gt $max_attempts ]; then
            echo "❌ Application failed to start within expected time"
            echo "Check logs: docker-compose logs"
        fi
        ;;
    2)
        echo ""
        echo "🚀 Starting container..."
        docker-compose up -d
        echo "✅ Container started!"
        echo ""
        echo "🔍 View logs: docker-compose logs -f"
        ;;
    3)
        echo ""
        echo "🛑 Stopping container..."
        docker-compose stop
        echo "✅ Container stopped!"
        ;;
    4)
        echo ""
        echo "📋 Following logs (Ctrl+C to exit)..."
        docker-compose logs -f
        ;;
    5)
        echo ""
        echo "🔄 Restarting container..."
        docker-compose restart
        echo "✅ Container restarted!"
        echo ""
        echo "🔍 View logs: docker-compose logs -f"
        ;;
    6)
        echo ""
        read -p "Are you sure you want to remove everything? [y/N]: " confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            echo "🗑️  Removing containers and volumes..."
            docker-compose down -v
            echo "✅ Removed!"
        else
            echo "❌ Cancelled"
        fi
        ;;
    *)
        echo "❌ Invalid choice"
        exit 1
        ;;
esac
