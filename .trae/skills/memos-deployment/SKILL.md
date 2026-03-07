---
name: "memos-deployment"
description: "Provides steps for installing, deploying, and restarting Memos. Invoke when user needs to set up or restart Memos service."
---

# Memos Deployment Skill

This skill provides comprehensive steps for installing, deploying, and restarting Memos, a self-hosted knowledge management platform.

**Important Note:** All commands use port 8081 for Memos service. Do not change the port unless absolutely necessary.

## Prerequisites

- Go 1.25 or later
- Nginx (for reverse proxy)
- Domain name (optional, for external access)

## Installation Steps

### 1. Build Memos Binary

```bash
go build -o ./build/memos ./cmd/memos
```

### 2. Start Memos Service

```bash
nohup ./build/memos --port 8081 --addr 0.0.0.0 > memos.log 2>&1 &
sleep 1
```

### 3. Verify Memos Service

```bash
curl -sS -I http://127.0.0.1:8081/ | sed -n '1,40p' || true
```

### 4. Configure Nginx

#### Create Nginx Configuration

```bash
# Example Nginx config file: /usr/local/etc/nginx/servers/memos.conf

server {
    listen 80;
    server_name memos.yingshun.xin;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name memos.yingshun.xin;
    
    ssl_certificate /path/to/certificate.pem;
    ssl_certificate_key /path/to/certificate.key;
    
    location / {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 5. Test and Reload Nginx

```bash
sudo nginx -t && sudo nginx -s reload || (sudo rm -f /usr/local/var/run/nginx.pid && sudo nginx -t && sudo nginx)
```

## Restart Steps

### 1. Stop Existing Memos Process

```bash
pkill -f ./build/memos || true
```

### 2. Rebuild and Restart

```bash
go build -o ./build/memos ./cmd/memos
nohup ./build/memos --port 8081 --addr 0.0.0.0 > memos.log 2>&1 &
sleep 1
```

### 3. Verify Service Status

```bash
curl -sS -I http://127.0.0.1:8081/ | sed -n '1,40p' || true
```

## Troubleshooting

### Check Memos Logs

```bash
tail -f memos.log
```

### Check Nginx Status

```bash
sudo nginx -t
sudo nginx -s reload
```

### DNS Resolution

```bash
dig memos.yingshun.xin
```

## Access Verification

```bash
curl -sS -I https://memos.yingshun.xin
```

This skill provides a complete workflow for deploying and managing Memos service, including building, starting, verifying, and restarting the application.