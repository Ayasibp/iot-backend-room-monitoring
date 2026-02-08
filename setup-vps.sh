#!/bin/bash

# Complete VPS Setup Script
# Run this script on your VPS to set up everything automatically

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}  IoT Backend VPS Setup Script${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo -e "${YELLOW}Warning: Running as root. This is not recommended.${NC}"
    echo "Consider running as a regular user with sudo privileges."
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Function to print status
print_status() {
    echo -e "${BLUE}[*]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Update system
print_status "Updating system packages..."
sudo apt update && sudo apt upgrade -y
print_success "System updated"

# Install required packages
print_status "Installing required packages..."
sudo apt install -y curl git wget nano
print_success "Required packages installed"

# Install Docker
print_status "Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
    print_success "Docker installed"
else
    print_success "Docker already installed"
fi

# Install Docker Compose
print_status "Installing Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    print_success "Docker Compose installed"
else
    print_success "Docker Compose already installed"
fi

# Verify installations
echo ""
print_status "Verifying installations..."
docker --version
docker-compose --version
print_success "All tools verified"

# Setup application directory
echo ""
print_status "Setting up application directory..."
APP_DIR="/opt/iot-backend"
sudo mkdir -p $APP_DIR
sudo chown $USER:$USER $APP_DIR
print_success "Application directory created at $APP_DIR"

# Setup backup directory
print_status "Setting up backup directory..."
BACKUP_DIR="/opt/backups/mysql"
sudo mkdir -p $BACKUP_DIR
sudo chown $USER:$USER /opt/backups
print_success "Backup directory created at $BACKUP_DIR"

# Prompt for repository URL
echo ""
echo -e "${YELLOW}Repository Setup${NC}"
read -p "Do you want to clone from git? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -p "Enter repository URL: " REPO_URL
    read -p "Enter branch name (default: main): " BRANCH
    BRANCH=${BRANCH:-main}
    
    print_status "Cloning repository..."
    cd $APP_DIR
    git clone $REPO_URL .
    git checkout $BRANCH
    print_success "Repository cloned"
else
    print_status "Manual file upload required"
    echo "Please upload your files to: $APP_DIR"
    read -p "Press Enter when files are uploaded..."
fi

# Generate secrets
echo ""
echo -e "${YELLOW}Security Configuration${NC}"
print_status "Generating secure secrets..."

DB_PASSWORD=$(openssl rand -base64 32)
MYSQL_ROOT_PASSWORD=$(openssl rand -base64 32)
JWT_ACCESS_SECRET=$(openssl rand -base64 64)
JWT_REFRESH_SECRET=$(openssl rand -base64 64)

print_success "Secrets generated"

# Get domain for CORS
read -p "Enter your frontend domain (e.g., https://yourdomain.com) or press Enter for localhost: " DOMAIN
if [ -z "$DOMAIN" ]; then
    ALLOWED_ORIGINS="http://localhost:3000,http://localhost:5173"
else
    ALLOWED_ORIGINS="$DOMAIN,https://$DOMAIN,http://$DOMAIN"
fi

# Create .env file
print_status "Creating .env file..."
cd $APP_DIR

cat > .env << EOF
# Database Configuration
DB_PASSWORD=$DB_PASSWORD
MYSQL_ROOT_PASSWORD=$MYSQL_ROOT_PASSWORD

# JWT Secrets
JWT_ACCESS_SECRET=$JWT_ACCESS_SECRET
JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET

# CORS
ALLOWED_ORIGINS=$ALLOWED_ORIGINS

# Server Configuration
PORT=8080
GIN_MODE=release
ACCESS_TOKEN_EXPIRY=15m
REFRESH_TOKEN_EXPIRY=168h
EOF

print_success ".env file created"

# Save credentials to file
CREDS_FILE="/root/iot-backend-credentials.txt"
sudo bash -c "cat > $CREDS_FILE" << EOF
IoT Backend Credentials - $(date)
====================================

Database Password: $DB_PASSWORD
MySQL Root Password: $MYSQL_ROOT_PASSWORD

JWT Access Secret: $JWT_ACCESS_SECRET
JWT Refresh Secret: $JWT_REFRESH_SECRET

CORS Allowed Origins: $ALLOWED_ORIGINS

Default Application Users (CHANGE THESE!):
- Admin: username=admin, password=admin123
- User: username=user, password=user123

Application Directory: $APP_DIR
Backup Directory: $BACKUP_DIR

To view this file again: sudo cat $CREDS_FILE
====================================
EOF

sudo chmod 600 $CREDS_FILE
print_success "Credentials saved to $CREDS_FILE"

# Make scripts executable
if [ -f deploy.sh ]; then
    chmod +x deploy.sh
fi
if [ -f backup.sh ]; then
    chmod +x backup.sh
fi

# Setup firewall
echo ""
read -p "Do you want to configure the firewall? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Configuring firewall..."
    sudo ufw allow 22/tcp
    sudo ufw allow 80/tcp
    sudo ufw allow 443/tcp
    sudo ufw --force enable
    print_success "Firewall configured"
fi

# Setup automated backups
echo ""
read -p "Do you want to setup automated daily backups? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -f backup.sh ]; then
        print_status "Setting up cron job for backups..."
        (crontab -l 2>/dev/null; echo "0 2 * * * $APP_DIR/backup.sh >> /var/log/iot-backup.log 2>&1") | crontab -
        print_success "Daily backup scheduled at 2 AM"
    else
        print_error "backup.sh not found. Skipping cron setup."
    fi
fi

# Nginx setup
echo ""
read -p "Do you want to install Nginx reverse proxy? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Installing Nginx..."
    sudo apt install -y nginx
    
    read -p "Enter your domain for API (e.g., api.yourdomain.com): " API_DOMAIN
    
    if [ ! -z "$API_DOMAIN" ]; then
        sudo bash -c "cat > /etc/nginx/sites-available/iot-backend" << EOF
server {
    listen 80;
    server_name $API_DOMAIN;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }
}
EOF
        
        sudo ln -sf /etc/nginx/sites-available/iot-backend /etc/nginx/sites-enabled/
        sudo nginx -t && sudo systemctl restart nginx
        print_success "Nginx configured for $API_DOMAIN"
        
        # SSL setup
        read -p "Do you want to setup SSL with Let's Encrypt? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installing Certbot..."
            sudo apt install -y certbot python3-certbot-nginx
            sudo certbot --nginx -d $API_DOMAIN
            print_success "SSL certificate obtained"
        fi
    fi
fi

# Deploy application
echo ""
echo -e "${YELLOW}Application Deployment${NC}"
read -p "Do you want to deploy the application now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -f deploy.sh ]; then
        print_status "Deploying application..."
        ./deploy.sh
    else
        print_status "Starting application manually..."
        docker-compose up -d
        echo ""
        print_status "Waiting for application to be ready..."
        sleep 10
        
        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            print_success "Application is running!"
        else
            print_error "Application might not be ready yet. Check logs with: docker-compose logs -f"
        fi
    fi
fi

# Summary
echo ""
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}  Setup Complete!${NC}"
echo -e "${GREEN}=====================================${NC}"
echo ""
echo "📁 Application Directory: $APP_DIR"
echo "💾 Backup Directory: $BACKUP_DIR"
echo "🔑 Credentials saved to: $CREDS_FILE"
echo ""
echo -e "${YELLOW}Important Next Steps:${NC}"
echo "1. Change default passwords (admin/admin123, user/user123)"
echo "2. Review logs: docker-compose logs -f app"
echo "3. Test health: curl http://localhost:8080/health"
echo "4. Access credentials: sudo cat $CREDS_FILE"
echo ""
echo -e "${YELLOW}Useful Commands:${NC}"
echo "  cd $APP_DIR"
echo "  docker-compose logs -f app     # View logs"
echo "  docker-compose ps              # Check status"
echo "  docker-compose restart app     # Restart app"
echo "  ./backup.sh                    # Backup database"
echo ""
echo -e "${BLUE}For more information, see:${NC}"
echo "  - DOCKER_DEPLOYMENT.md"
echo "  - DEPLOYMENT_QUICKSTART.md"
echo "  - VPS_DEPLOYMENT_SUMMARY.md"
echo ""
print_success "Your IoT backend is ready! 🚀"
