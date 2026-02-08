#!/bin/bash

# IoT Backend Deployment Script
# This script helps deploy the application to production

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_DIR="/opt/iot-backend"
BACKUP_DIR="/opt/backups/iot-backend"
COMPOSE_FILE="docker-compose.yml"

# Functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    print_success "Docker and Docker Compose are installed"
}

check_env_file() {
    if [ ! -f .env ]; then
        print_warning ".env file not found"
        
        if [ -f .env.production ]; then
            print_info "Copying .env.production to .env"
            cp .env.production .env
            print_warning "IMPORTANT: Edit .env and set secure passwords and secrets!"
            print_info "Generate JWT secrets with: openssl rand -base64 64"
            read -p "Press Enter after you've updated the .env file..."
        else
            print_error "No .env or .env.production file found. Please create one."
            exit 1
        fi
    fi
    
    # Check if secrets need to be changed
    if grep -q "CHANGE_ME" .env; then
        print_error ".env file contains 'CHANGE_ME' placeholders. Please set secure values!"
        exit 1
    fi
    
    print_success ".env file is configured"
}

backup_database() {
    print_info "Creating database backup..."
    
    if [ ! -d "$BACKUP_DIR" ]; then
        mkdir -p "$BACKUP_DIR"
    fi
    
    if docker ps | grep -q iot-mysql; then
        BACKUP_FILE="$BACKUP_DIR/backup_$(date +%Y%m%d_%H%M%S).sql"
        docker-compose exec -T db mysqldump -u root -p"${MYSQL_ROOT_PASSWORD}" iot_theater_monitoring > "$BACKUP_FILE" 2>/dev/null || true
        
        if [ -f "$BACKUP_FILE" ]; then
            gzip "$BACKUP_FILE"
            print_success "Database backed up to $BACKUP_FILE.gz"
        else
            print_warning "Backup skipped (database might be empty or not running)"
        fi
    else
        print_warning "Database container not running. Skipping backup."
    fi
}

pull_latest_code() {
    print_info "Checking for code updates..."
    
    if [ -d .git ]; then
        git pull
        print_success "Code updated"
    else
        print_warning "Not a git repository. Skipping git pull."
    fi
}

build_and_start() {
    print_info "Building and starting containers..."
    
    # Stop existing containers
    docker-compose down
    
    # Build new image
    docker-compose build --no-cache
    
    # Start containers
    docker-compose up -d
    
    print_success "Containers started"
}

wait_for_health() {
    print_info "Waiting for application to be healthy..."
    
    RETRIES=30
    COUNT=0
    
    while [ $COUNT -lt $RETRIES ]; do
        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            print_success "Application is healthy!"
            return 0
        fi
        
        COUNT=$((COUNT + 1))
        echo -n "."
        sleep 2
    done
    
    print_error "Application failed to become healthy"
    print_info "Checking logs..."
    docker-compose logs --tail=50 app
    exit 1
}

show_status() {
    print_info "Deployment Status:"
    echo ""
    docker-compose ps
    echo ""
    print_info "Logs (last 20 lines):"
    docker-compose logs --tail=20 app
}

cleanup_old_images() {
    print_info "Cleaning up old Docker images..."
    docker image prune -f
    print_success "Cleanup complete"
}

# Main deployment flow
main() {
    print_info "Starting IoT Backend Deployment"
    echo ""
    
    # Pre-deployment checks
    check_docker
    check_env_file
    
    # Backup before deployment
    backup_database
    
    # Update code if in git repo
    pull_latest_code
    
    # Deploy
    build_and_start
    
    # Wait for application to be ready
    wait_for_health
    
    # Show status
    show_status
    
    # Cleanup
    cleanup_old_images
    
    echo ""
    print_success "Deployment completed successfully!"
    echo ""
    print_info "Useful commands:"
    echo "  View logs:        docker-compose logs -f app"
    echo "  Restart app:      docker-compose restart app"
    echo "  Stop all:         docker-compose down"
    echo "  Database shell:   docker-compose exec db mysql -u root -p iot_theater_monitoring"
    echo ""
}

# Run main function
main
