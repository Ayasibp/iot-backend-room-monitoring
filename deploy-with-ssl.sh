#!/bin/bash

# Deploy IoT Backend with Self-Signed SSL Certificate
# Use this when you don't have a domain name yet

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}  IoT Backend with Self-Signed SSL  ${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""

# Get VPS IP
print_info "Detecting VPS IP address..."
VPS_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || hostname -I | awk '{print $1}')

if [ -z "$VPS_IP" ]; then
    print_error "Could not detect VPS IP automatically"
    read -p "Please enter your VPS IP address: " VPS_IP
fi

print_success "VPS IP: $VPS_IP"
echo ""

# Deploy application first
print_info "Deploying application..."
if [ -f ./deploy.sh ]; then
    ./deploy.sh
else
    print_warning "deploy.sh not found, deploying manually..."
    docker-compose up -d
fi
print_success "Application deployed"
echo ""

# Install Nginx
print_info "Installing Nginx..."
if ! command -v nginx &> /dev/null; then
    sudo apt update
    sudo apt install -y nginx
    print_success "Nginx installed"
else
    print_success "Nginx already installed"
fi

# Generate self-signed certificate
print_info "Generating self-signed SSL certificate..."
sudo mkdir -p /etc/ssl/iot-backend

print_info "Certificate will be valid for 365 days"
print_warning "Browsers will show a security warning (this is normal for self-signed certificates)"

sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/ssl/iot-backend/key.pem \
  -out /etc/ssl/iot-backend/cert.pem \
  -subj "/C=ID/ST=Jakarta/L=Jakarta/O=IoT-Backend/CN=$VPS_IP"

print_success "SSL certificate generated"
echo ""

# Configure Nginx
print_info "Configuring Nginx..."
sudo tee /etc/nginx/sites-available/iot-backend > /dev/null <<EOF
# HTTP - Redirect to HTTPS
server {
    listen 80;
    server_name $VPS_IP _;
    return 301 https://\$server_name\$request_uri;
}

# HTTPS
server {
    listen 443 ssl http2;
    server_name $VPS_IP _;

    # SSL Configuration - Self-signed certificate
    ssl_certificate /etc/ssl/iot-backend/cert.pem;
    ssl_certificate_key /etc/ssl/iot-backend/key.pem;
    
    # SSL Security
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security headers
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # Client body size
    client_max_body_size 10M;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        
        # WebSocket support
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        
        # Headers
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint (no logs)
    location /health {
        proxy_pass http://localhost:8080/health;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        access_log off;
    }

    # Access logs
    access_log /var/log/nginx/iot-backend-access.log;
    error_log /var/log/nginx/iot-backend-error.log;
}
EOF

print_success "Nginx configured"
echo ""

# Enable and restart Nginx
print_info "Enabling Nginx configuration..."
sudo ln -sf /etc/nginx/sites-available/iot-backend /etc/nginx/sites-enabled/

# Test Nginx configuration
print_info "Testing Nginx configuration..."
if sudo nginx -t; then
    print_success "Nginx configuration is valid"
else
    print_error "Nginx configuration test failed!"
    exit 1
fi

# Restart Nginx
print_info "Restarting Nginx..."
sudo systemctl restart nginx
sudo systemctl enable nginx
print_success "Nginx restarted"
echo ""

# Update firewall
print_info "Updating firewall rules..."
if command -v ufw &> /dev/null; then
    sudo ufw allow 80/tcp
    sudo ufw allow 443/tcp
    print_success "Firewall updated (ports 80, 443 allowed)"
else
    print_warning "UFW not found, skipping firewall configuration"
fi
echo ""

# Update .env with HTTPS URL
print_info "Updating CORS configuration..."
if [ -f .env ]; then
    # Check if ALLOWED_ORIGINS exists
    if grep -q "ALLOWED_ORIGINS" .env; then
        # Backup .env
        cp .env .env.backup
        
        # Update ALLOWED_ORIGINS to use HTTPS
        sed -i.bak "s|ALLOWED_ORIGINS=.*|ALLOWED_ORIGINS=https://$VPS_IP,http://$VPS_IP|" .env
        
        # Restart app to apply new CORS settings
        docker-compose restart app
        print_success "CORS configuration updated"
    else
        print_warning "ALLOWED_ORIGINS not found in .env, please add manually:"
        echo "ALLOWED_ORIGINS=https://$VPS_IP,http://$VPS_IP"
    fi
else
    print_warning ".env file not found"
fi
echo ""

# Test HTTPS endpoint
print_info "Testing HTTPS endpoint..."
sleep 2
if curl -k -f https://$VPS_IP/health > /dev/null 2>&1; then
    print_success "HTTPS endpoint is responding!"
else
    print_warning "HTTPS endpoint test failed, but this might be normal"
fi
echo ""

# Summary
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}  Deployment Complete!${NC}"
echo -e "${GREEN}=====================================${NC}"
echo ""
echo -e "${BLUE}📍 Your API URLs:${NC}"
echo "  HTTP:  http://$VPS_IP  (redirects to HTTPS)"
echo "  HTTPS: https://$VPS_IP"
echo ""
echo -e "${BLUE}🔑 SSL Certificate:${NC}"
echo "  Type: Self-signed"
echo "  Location: /etc/ssl/iot-backend/"
echo "  Valid for: 365 days"
echo ""
echo -e "${YELLOW}⚠️  IMPORTANT - Browser Security Warning:${NC}"
echo ""
echo "When you access https://$VPS_IP in a browser, you will see"
echo "a security warning because the certificate is self-signed."
echo ""
echo "This is NORMAL and EXPECTED. To continue:"
echo "  • Chrome: Click 'Advanced' → 'Proceed to $VPS_IP (unsafe)'"
echo "  • Firefox: Click 'Advanced' → 'Accept the Risk and Continue'"
echo "  • Safari: Click 'Show Details' → 'Visit this website'"
echo ""
echo "Or type: ${BLUE}thisisunsafe${NC} on the warning page in Chrome"
echo ""
echo -e "${BLUE}🧪 Test Your API:${NC}"
echo "  curl -k https://$VPS_IP/health"
echo ""
echo -e "${BLUE}📱 For Frontend:${NC}"
echo "  Update your frontend to use: https://$VPS_IP"
echo ""
echo -e "${BLUE}🔒 Security Note:${NC}"
echo "  Self-signed certificates are OK for:"
echo "    ✓ Development"
echo "    ✓ Testing"
echo "    ✓ Internal use"
echo ""
echo "  For production, consider:"
echo "    • Get a free domain (duckdns.org, freenom.com)"
echo "    • Use Let's Encrypt for real SSL certificate"
echo "    • See: HTTPS_WITHOUT_DOMAIN.md for options"
echo ""
echo -e "${GREEN}🚀 Your IoT Backend is now running with HTTPS!${NC}"
echo ""
echo "View logs: docker-compose logs -f app"
echo "Restart: docker-compose restart app"
echo "Status: docker-compose ps"
echo ""
