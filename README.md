# HTTP Remote

[![Test](https://github.com/pandeptwidyaop/http-remote/actions/workflows/test.yml/badge.svg)](https://github.com/pandeptwidyaop/http-remote/actions/workflows/test.yml)
[![Release](https://github.com/pandeptwidyaop/http-remote/actions/workflows/release.yml/badge.svg)](https://github.com/pandeptwidyaop/http-remote/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/pandeptwidyaop/http-remote/branch/main/graph/badge.svg)](https://codecov.io/gh/pandeptwidyaop/http-remote)
[![Go Report Card](https://goreportcard.com/badge/github.com/pandeptwidyaop/http-remote)](https://goreportcard.com/report/github.com/pandeptwidyaop/http-remote)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Tool DevOps untuk melakukan remote deployment dan eksekusi command pada server private melalui protokol HTTP. Cocok untuk server yang hanya bisa diakses via port 80/443 tanpa VPN.

## Features

- **Modern React SPA UI** - Built with React 18, TypeScript, and Tailwind CSS
- **2FA/TOTP Support** - Two-factor authentication with encrypted secret storage (AES-256-GCM)
- **Backup Codes** - Recovery codes for 2FA with encryption
- **Remote Terminal** - Interactive WebSocket-based shell access with PTY support
- **Web UI & REST API** - Dashboard web dan API untuk otomatisasi
- **App Management** - Kelola multiple aplikasi/project
- **Command Templates** - Simpan command deployment untuk reuse
- **Real-time Output** - Streaming output via SSE (Server-Sent Events)
- **Session Auth** - Login dengan username/password untuk Web UI
- **Token Auth** - Deploy via API menggunakan token (untuk CI/CD)
- **UUID Identifiers** - Semua resource menggunakan UUID
- **SQLite Database** - Portable, tidak perlu database server
- **Single Binary** - Semua assets (HTML, CSS, JS) embedded dalam satu executable
- **Audit Logging** - Track all user actions and executions
- **Rate Limiting** - Protection against brute force attacks

## Requirements

- Go 1.21+
- GCC (untuk kompilasi SQLite)
- Node.js 18+ and npm (for building frontend)

## Installation

### Single Binary (Recommended)

Binary sudah include semua assets (HTML, CSS, JS) - tidak perlu folder `web/` terpisah.

```bash
# Clone repository
git clone https://github.com/pandeptwidyaop/http-remote.git
cd http-remote

# Build frontend (React SPA)
cd web
npm install
npm run build
cd ..

# Build single binary (includes embedded frontend assets)
go build -o http-remote ./cmd/server

# Run dari mana saja (tidak perlu folder web/)
./http-remote

# Atau dengan config custom
./http-remote -config /path/to/config.yaml
```

### Cross-compile untuk Linux

```bash
# Build untuk Linux AMD64
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc \
  go build -ldflags="-s -w" -o http-remote-linux-amd64 ./cmd/server

# Build untuk Linux ARM64
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-musl-gcc \
  go build -ldflags="-s -w" -o http-remote-linux-arm64 ./cmd/server
```

> **Note**: Cross-compile membutuhkan musl-cross toolchain karena CGO (SQLite).
> Install via: `brew install FiloSottile/musl-cross/musl-cross`

## Configuration

Buat file `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  path_prefix: "/devops"
  secure_cookie: false  # Set true for production (requires HTTPS)

database:
  path: "./data/deploy.db"

auth:
  session_duration: "24h"
  bcrypt_cost: 12

execution:
  default_timeout: 300   # seconds
  max_timeout: 3600      # seconds
  max_output_size: 10485760  # 10MB

admin:
  username: "admin"
  password: "your-secure-password"  # REQUIRED: Must NOT be "changeme"

terminal:
  shell: "/bin/bash"    # Shell to use (default: /bin/bash)
  args: ["-l"]          # Shell arguments (default: ["-l"] for login shell)
  env:                  # Additional environment variables
    - "SUDO_ASKPASS=/usr/bin/ssh-askpass"
  # enabled: true       # Set to false to disable terminal feature

# Security settings (REQUIRED)
security:
  # 32-byte encryption key for TOTP secrets (64 hex characters)
  # Generate with: openssl rand -hex 32
  encryption_key: "your-64-char-hex-key-here"

  # Account lockout settings (brute force protection)
  max_login_attempts: 5    # Lock after 5 failed attempts (default: 5)
  lockout_duration: "15m"  # Lock for 15 minutes (default: 15m)

# File browser security settings (optional)
files:
  # Whitelist: Only allow access to these paths (if set, all other paths are blocked)
  allowed_paths:
    - "/home/devops"
    - "/opt/apps"
  # Blacklist: Block access to these paths (in addition to system paths)
  blocked_paths:
    - "/home/devops/.ssh"
    - "/home/devops/.gnupg"
```

### Required Configuration

⚠️ **IMPORTANT**: The following configuration values are **required** for the application to start:

1. **`admin.password`**: Must be changed from "changeme" to a secure password
2. **`security.encryption_key`**: Must be set to a 64-character hex string (32 bytes)

Generate an encryption key with:
```bash
openssl rand -hex 32
```

The application will refuse to start if these requirements are not met.

### Nginx Reverse Proxy Configuration

Jika menggunakan Nginx sebagai reverse proxy untuk HTTP Remote:

```nginx
# /etc/nginx/sites-available/devops.conf

upstream http_remote {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name devops.example.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name devops.example.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/devops.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/devops.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Proxy settings for /devops path
    location /devops {
        proxy_pass http://http_remote;
        proxy_http_version 1.1;

        # Headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (for terminal)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 3600s;  # Long timeout for terminal sessions

        # Buffering (disable for SSE streaming)
        proxy_buffering off;
        proxy_cache off;
    }

    # Health check endpoint (optional)
    location /devops/api/version {
        proxy_pass http://http_remote;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        access_log off;
    }
}
```

**Catatan penting:**
- `proxy_read_timeout 3600s` diperlukan untuk terminal WebSocket yang bisa berjalan lama
- `proxy_buffering off` penting untuk SSE streaming output
- `Upgrade` dan `Connection` headers diperlukan untuk WebSocket terminal
- Gunakan `secure_cookie: true` di config.yaml saat menggunakan HTTPS

## Quick Start

### 1. Setup Configuration

Before running, create your `config.yaml` with required settings:

```bash
# Generate encryption key
ENCRYPTION_KEY=$(openssl rand -hex 32)
echo "Your encryption key: $ENCRYPTION_KEY"
```

### 2. Akses Web UI

Buka browser ke `http://localhost:8080/devops/`

Login dengan:
- Username: `admin`
- Password: (password yang Anda set di config.yaml)

### 3. Buat App

Klik "New App" dan isi:
- **Name**: Nama aplikasi (e.g., `my-webapp`)
- **Working Directory**: Path ke aplikasi (e.g., `/opt/apps/my-webapp`)

### 4. Buat Command

Pada halaman app, klik "New Command" dan isi:
- **Name**: Nama command (e.g., `deploy`)
- **Command**: Shell command yang akan dieksekusi
- **Timeout**: Batas waktu eksekusi (seconds)

Contoh command:
```bash
git pull origin main && docker-compose up -d --build
```

### 5. Execute

Klik "Execute" pada command untuk menjalankannya. Output akan ditampilkan secara real-time.

### 6. Enable Two-Factor Authentication (Optional)

1. Navigate to **Settings** page
2. Click "Enable 2FA"
3. Scan QR code with authenticator app (Google Authenticator, Authy, etc.)
4. Enter 6-digit code to verify
5. Save backup codes securely for account recovery

### 7. Remote Terminal Access

1. Navigate to **Terminal** page in the navigation menu
2. Interactive shell session will open automatically
3. Execute commands directly on the server with real-time output
4. **Security Warning**: Terminal provides full shell access - use with caution

---

## Deploy via API (Token Auth)

Setiap app memiliki token unik yang bisa digunakan untuk trigger deployment tanpa login. Token bisa dilihat di halaman detail app.

### Trigger Deployment

```bash
# Deploy menggunakan command default (command pertama)
curl -X POST http://localhost:8080/devops/deploy/{app_uuid} \
  -H "X-Deploy-Token: {token}"

# Deploy dengan command spesifik
curl -X POST http://localhost:8080/devops/deploy/{app_uuid} \
  -H "X-Deploy-Token: {token}" \
  -H "Content-Type: application/json" \
  -d '{"command_id": "command-uuid"}'
```

Response:
```json
{
  "message": "deployment started",
  "execution_id": "exec-uuid",
  "app_id": "app-uuid",
  "app_name": "my-webapp",
  "stream_url": "/devops/api/executions/exec-uuid/stream",
  "status_url": "/devops/api/executions/exec-uuid"
}
```

### Check Deployment Status

```bash
curl http://localhost:8080/devops/deploy/{app_uuid}/status/{execution_uuid} \
  -H "X-Deploy-Token: {token}"
```

Response:
```json
{
  "id": "exec-uuid",
  "command_id": "cmd-uuid",
  "status": "success",
  "output": "Already up to date.\nContainer started.\n",
  "exit_code": 0,
  "started_at": "2024-01-15T10:30:00Z",
  "finished_at": "2024-01-15T10:30:05Z"
}
```

### Regenerate Token

Jika token bocor, regenerate via Web UI atau API:

```bash
curl -X POST http://localhost:8080/devops/api/apps/{app_uuid}/regenerate-token \
  -b "session_id=YOUR_SESSION_ID"
```

---

## REST API Reference

### Authentication

```bash
# Login
curl -X POST http://localhost:8080/devops/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme"}'

# Logout
curl -X POST http://localhost:8080/devops/api/auth/logout \
  -b "session_id=YOUR_SESSION_ID"

# Get current user
curl http://localhost:8080/devops/api/auth/me \
  -b "session_id=YOUR_SESSION_ID"
```

### Apps

```bash
# List apps
curl http://localhost:8080/devops/api/apps \
  -b "session_id=YOUR_SESSION_ID"

# Create app
curl -X POST http://localhost:8080/devops/api/apps \
  -H "Content-Type: application/json" \
  -b "session_id=YOUR_SESSION_ID" \
  -d '{
    "name": "my-app",
    "description": "My Application",
    "working_dir": "/opt/apps/my-app"
  }'

# Get app detail
curl http://localhost:8080/devops/api/apps/{app_uuid} \
  -b "session_id=YOUR_SESSION_ID"

# Update app
curl -X PUT http://localhost:8080/devops/api/apps/{app_uuid} \
  -H "Content-Type: application/json" \
  -b "session_id=YOUR_SESSION_ID" \
  -d '{"description": "Updated description"}'

# Delete app
curl -X DELETE http://localhost:8080/devops/api/apps/{app_uuid} \
  -b "session_id=YOUR_SESSION_ID"

# Regenerate token
curl -X POST http://localhost:8080/devops/api/apps/{app_uuid}/regenerate-token \
  -b "session_id=YOUR_SESSION_ID"
```

### Commands

```bash
# List commands for app
curl http://localhost:8080/devops/api/apps/{app_uuid}/commands \
  -b "session_id=YOUR_SESSION_ID"

# Create command
curl -X POST http://localhost:8080/devops/api/apps/{app_uuid}/commands \
  -H "Content-Type: application/json" \
  -b "session_id=YOUR_SESSION_ID" \
  -d '{
    "name": "deploy",
    "description": "Pull and restart containers",
    "command": "git pull && docker-compose up -d --build",
    "timeout_seconds": 600
  }'

# Get command detail
curl http://localhost:8080/devops/api/commands/{command_uuid} \
  -b "session_id=YOUR_SESSION_ID"

# Update command
curl -X PUT http://localhost:8080/devops/api/commands/{command_uuid} \
  -H "Content-Type: application/json" \
  -b "session_id=YOUR_SESSION_ID" \
  -d '{"timeout_seconds": 900}'

# Delete command
curl -X DELETE http://localhost:8080/devops/api/commands/{command_uuid} \
  -b "session_id=YOUR_SESSION_ID"

# Execute command
curl -X POST http://localhost:8080/devops/api/commands/{command_uuid}/execute \
  -b "session_id=YOUR_SESSION_ID"
```

### Executions

```bash
# List executions
curl http://localhost:8080/devops/api/executions \
  -b "session_id=YOUR_SESSION_ID"

# Get execution detail
curl http://localhost:8080/devops/api/executions/{execution_uuid} \
  -b "session_id=YOUR_SESSION_ID"

# Stream execution output (SSE)
curl http://localhost:8080/devops/api/executions/{execution_uuid}/stream \
  -b "session_id=YOUR_SESSION_ID"
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger deployment
        run: |
          curl -X POST ${{ secrets.DEPLOY_URL }}/devops/deploy/${{ secrets.APP_ID }} \
            -H "X-Deploy-Token: ${{ secrets.DEPLOY_TOKEN }}"
```

### GitLab CI

```yaml
deploy:
  stage: deploy
  script:
    - |
      curl -X POST ${DEPLOY_URL}/devops/deploy/${APP_ID} \
        -H "X-Deploy-Token: ${DEPLOY_TOKEN}"
  only:
    - main
```

### Jenkins

```groovy
pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                sh '''
                    curl -X POST ${DEPLOY_URL}/devops/deploy/${APP_ID} \
                      -H "X-Deploy-Token: ${DEPLOY_TOKEN}"
                '''
            }
        }
    }
}
```

---

## Traefik Integration

### Traefik + HTTP Remote dalam Docker

Jika keduanya berjalan di Docker:

```yaml
http:
  routers:
    devops:
      rule: "Host(`app.example.com`) && PathPrefix(`/devops`)"
      service: devops-service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

  services:
    devops-service:
      loadBalancer:
        servers:
          - url: "http://http-remote:8080"
```

### Traefik di Docker + HTTP Remote di Host

Untuk skenario dimana Traefik berjalan di Docker dan HTTP Remote berjalan langsung di host (systemd service):

```
Internet → Traefik (Docker, port 80/443) → Host (port 8080) → http-remote
```

**1. Docker Compose untuk Traefik:**

```yaml
# docker-compose.yml
version: '3.8'

services:
  traefik:
    image: traefik:v3.0
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik.yml:/etc/traefik/traefik.yml:ro
      - ./dynamic:/etc/traefik/dynamic:ro
      - ./letsencrypt:/letsencrypt
    extra_hosts:
      - "host.docker.internal:host-gateway"  # Penting untuk akses host dari Docker
    restart: unless-stopped
```

**2. Traefik Static Config:**

```yaml
# traefik.yml
api:
  dashboard: true

entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"

certificatesResolvers:
  letsencrypt:
    acme:
      email: your-email@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web

providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
```

**3. Dynamic Config untuk HTTP Remote:**

```yaml
# dynamic/http-remote.yml
http:
  routers:
    http-remote:
      rule: "Host(`deploy.example.com`) && PathPrefix(`/devops`)"
      service: http-remote
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

  services:
    http-remote:
      loadBalancer:
        servers:
          - url: "http://host.docker.internal:8080"
```

**4. Pastikan HTTP Remote Listen di Semua Interface:**

```yaml
# /etc/http-remote/config.yaml
server:
  host: "0.0.0.0"  # Listen semua interface
  port: 8080
  path_prefix: "/devops"
```

**Alternatif: Menggunakan IP Docker Bridge**

Jika `host.docker.internal` tidak tersedia:

```bash
# Dapatkan IP docker bridge
ip addr show docker0
# Output: inet 172.17.0.1/16
```

```yaml
# dynamic/http-remote.yml
http:
  services:
    http-remote:
      loadBalancer:
        servers:
          - url: "http://172.17.0.1:8080"
```

**Verifikasi:**

```bash
# Test dari dalam container Traefik
docker exec -it traefik wget -qO- http://host.docker.internal:8080/devops/login

# Test dari luar via HTTPS
curl https://deploy.example.com/devops/
```

---

## Docker

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o http-remote ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/http-remote .
COPY --from=builder /app/config.yaml .

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./http-remote"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  http-remote:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml
    restart: unless-stopped
```

---

## Linux Systemd Service

Untuk menjalankan HTTP Remote sebagai service di Linux:

### 1. Build dan Install Binary

```bash
# Build (single binary dengan embedded assets)
go build -ldflags="-s -w" -o http-remote ./cmd/server

# Copy binary ke /usr/local/bin
sudo cp http-remote /usr/local/bin/
sudo chmod +x /usr/local/bin/http-remote

# Buat direktori untuk config dan data
sudo mkdir -p /etc/http-remote
sudo mkdir -p /var/lib/http-remote

# Copy config
sudo cp config.yaml /etc/http-remote/

# Set permission
sudo chown -R root:root /etc/http-remote
sudo chown -R root:root /var/lib/http-remote
```

> **Note**: Tidak perlu copy folder `web/` karena sudah embedded dalam binary.

### 2. Update Config Path

Edit `/etc/http-remote/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  path_prefix: "/devops"

database:
  path: "/var/lib/http-remote/deploy.db"

auth:
  session_duration: "24h"
  bcrypt_cost: 12

execution:
  default_timeout: 300
  max_timeout: 3600
  max_output_size: 10485760

admin:
  username: "admin"
  password: "your-secure-password"
```

### 3. Buat Systemd Service

Buat file `/etc/systemd/system/http-remote.service`:

```ini
[Unit]
Description=HTTP Remote - DevOps Deployment Tool
Documentation=https://github.com/pandeptwidyaop/http-remote
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/etc/http-remote
ExecStart=/usr/local/bin/http-remote -config /etc/http-remote/config.yaml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Environment
Environment=GIN_MODE=release

# Security hardening (optional)
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/http-remote

[Install]
WantedBy=multi-user.target
```

### 4. Enable dan Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service (auto-start on boot)
sudo systemctl enable http-remote

# Start service
sudo systemctl start http-remote

# Check status
sudo systemctl status http-remote

# View logs
sudo journalctl -u http-remote -f
```

### 5. Service Commands

```bash
# Stop service
sudo systemctl stop http-remote

# Restart service
sudo systemctl restart http-remote

# Disable auto-start
sudo systemctl disable http-remote

# View recent logs
sudo journalctl -u http-remote -n 100

# View logs since today
sudo journalctl -u http-remote --since today
```

### 6. Menjalankan dengan User Non-Root (Recommended)

Untuk keamanan lebih baik, jalankan service dengan user khusus:

```bash
# Buat user dan group
sudo useradd -r -s /bin/false http-remote

# Set ownership
sudo chown -R http-remote:http-remote /var/lib/http-remote
sudo chown -R http-remote:http-remote /etc/http-remote

# Update service file
sudo sed -i 's/User=root/User=http-remote/' /etc/systemd/system/http-remote.service
sudo sed -i 's/Group=root/Group=http-remote/' /etc/systemd/system/http-remote.service

# Reload dan restart
sudo systemctl daemon-reload
sudo systemctl restart http-remote
```

> **Note**: Jika menggunakan user non-root, pastikan user tersebut memiliki akses ke working directory aplikasi yang akan di-deploy.

### 7. Logrotate (Optional)

Jika ingin rotasi log manual, buat `/etc/logrotate.d/http-remote`:

```
/var/log/http-remote/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 http-remote http-remote
    postrotate
        systemctl reload http-remote > /dev/null 2>&1 || true
    endscript
}
```

---

## Security Notes

1. **Ganti password default** setelah instalasi
2. **Gunakan HTTPS** di production (via Traefik/Nginx)
3. **Simpan token dengan aman** - jangan commit ke repository
4. **Regenerate token** jika ada kebocoran
5. **Batasi akses** ke path `/devops` via firewall/reverse proxy
6. **Review command** sebelum menyimpan - hindari command berbahaya

---

## Project Structure

```
.
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── config/config.go         # Configuration loader
│   ├── database/
│   │   ├── database.go          # SQLite connection
│   │   └── migrations.go        # DB schema
│   ├── handlers/
│   │   ├── auth.go              # Login/logout
│   │   ├── apps.go              # App CRUD
│   │   ├── commands.go          # Command CRUD & execute
│   │   ├── deploy.go            # Token-based deploy API
│   │   ├── stream.go            # SSE streaming
│   │   └── web.go               # Web UI handlers
│   ├── middleware/
│   │   ├── auth.go              # Session auth middleware
│   │   └── logging.go           # Request logging
│   ├── models/                  # Data models
│   ├── router/router.go         # Route definitions
│   └── services/
│       ├── auth.go              # Auth & session service
│       ├── apps.go              # App & command service
│       └── executor.go          # Command executor
├── web/
│   ├── static/css/style.css     # Styles
│   ├── static/js/app.js         # Frontend JS
│   └── templates/               # HTML templates
├── config.yaml                  # Configuration
├── Dockerfile
├── go.mod
└── README.md
```

---

## API Endpoints Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/devops/deploy/:app_id` | Token | Trigger deployment |
| GET | `/devops/deploy/:app_id/status/:exec_id` | Token | Get deployment status |
| POST | `/devops/api/auth/login` | - | Login |
| POST | `/devops/api/auth/logout` | Session | Logout |
| GET | `/devops/api/auth/me` | Session | Get current user |
| GET | `/devops/api/apps` | Session | List apps |
| POST | `/devops/api/apps` | Session | Create app |
| GET | `/devops/api/apps/:id` | Session | Get app |
| PUT | `/devops/api/apps/:id` | Session | Update app |
| DELETE | `/devops/api/apps/:id` | Session | Delete app |
| POST | `/devops/api/apps/:id/regenerate-token` | Session | Regenerate token |
| GET | `/devops/api/apps/:id/commands` | Session | List commands |
| POST | `/devops/api/apps/:id/commands` | Session | Create command |
| GET | `/devops/api/commands/:id` | Session | Get command |
| PUT | `/devops/api/commands/:id` | Session | Update command |
| DELETE | `/devops/api/commands/:id` | Session | Delete command |
| POST | `/devops/api/commands/:id/execute` | Session | Execute command |
| GET | `/devops/api/executions` | Session | List executions |
| GET | `/devops/api/executions/:id` | Session | Get execution |
| GET | `/devops/api/executions/:id/stream` | Session | Stream output (SSE) |
| GET | `/devops/api/2fa/status` | Session | Get 2FA status |
| POST | `/devops/api/2fa/generate-secret` | Session | Generate TOTP secret |
| GET | `/devops/api/2fa/qrcode` | Session | Get QR code for TOTP setup |
| POST | `/devops/api/2fa/enable` | Session | Enable 2FA with verification |
| POST | `/devops/api/2fa/disable` | Session | Disable 2FA |
| GET | `/devops/api/terminal/ws` | Session | WebSocket terminal connection |
| GET | `/devops/api/audit-logs` | Session | List audit logs |

---

## Security Features

HTTP Remote implements multiple security layers to protect your deployment infrastructure:

### Authentication & Session Management

- **Bcrypt Password Hashing**: Cost factor 12 (configurable)
- **Secure Session Cookies**: HttpOnly flag enabled, Secure flag configurable
- **Session Regeneration**: Automatic session invalidation on login to prevent session fixation
- **Required Secure Password**: Application refuses to start with default password "changeme"
- **Timing-Safe Login**: Constant-time credential verification prevents username enumeration
- **Two-Factor Authentication (2FA/TOTP)**: Optional TOTP-based 2FA using authenticator apps (Google Authenticator, Authy, etc.)
- **Encrypted TOTP Secrets**: TOTP secrets encrypted at rest using AES-256-GCM (required encryption key)
- **Backup Codes**: Encrypted recovery codes for 2FA account recovery

### Rate Limiting

Built-in rate limiting to prevent brute-force attacks:

- **Login Endpoint**: 5 requests/minute
- **API Endpoints**: 60 requests/minute
- **Deploy Endpoint**: 30 requests/minute
- **2FA Endpoints**: 10 requests/minute (generate, enable, disable)

Rate limit headers included in responses:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining
- `X-RateLimit-Reset`: Unix timestamp when limit resets
- `Retry-After`: Seconds to wait before retry

### Token Security

- **Constant-time Comparison**: Prevents timing attacks on token validation
- **UUID v4 Tokens**: Cryptographically random, non-predictable tokens

### File System Security

- **Path Traversal Protection**: All file paths validated using `filepath.EvalSymlinks()` to prevent symlink attacks
- **System Path Protection**: Critical system directories (`/bin`, `/etc`, `/usr`, etc.) are protected from modification
- **Configurable Path Access**:
  - `files.allowed_paths`: Whitelist of allowed paths (if set, only these paths are accessible)
  - `files.blocked_paths`: Blacklist of blocked paths (in addition to system paths)
- **Filename Sanitization**: Uploaded file names are sanitized to remove dangerous characters and patterns
- **Command Output Limits**: Execution output is truncated at configurable limit (default 10MB) to prevent memory exhaustion

### Audit Logging

All sensitive operations are logged to `audit_logs` table:

- User login/logout events
- Command create/update/delete operations
- Command execution with user and IP tracking

### Production Deployment Recommendations

1. **Configure Required Security Settings**:
   ```yaml
   admin:
     password: "your-secure-password"  # REQUIRED: Cannot be "changeme"
   security:
     encryption_key: "your-64-char-hex-key"  # REQUIRED: Generate with openssl rand -hex 32
   ```

2. **Enable Secure Cookies**:
   ```yaml
   server:
     secure_cookie: true  # Requires HTTPS
   ```

3. **Use HTTPS**: Deploy behind reverse proxy (Traefik, Nginx, Caddy) with TLS

4. **Firewall Rules**: Limit access to trusted IPs if possible

5. **Regular Updates**: Use `http-remote upgrade` to keep up-to-date

### Security Considerations

⚠️ **Command Injection**: Users with Web UI access can execute arbitrary shell commands. Only grant access to trusted users.

⚠️ **Working Directory**: Ensure working directories have appropriate permissions. Service should run as non-root user with limited privileges.

⚠️ **Database Backup**: SQLite database contains tokens and audit logs. Backup securely.

---

## Code Coverage

[![codecov](https://codecov.io/gh/pandeptwidyaop/http-remote/branch/main/graph/badge.svg)](https://codecov.io/gh/pandeptwidyaop/http-remote)

Code coverage dijalankan otomatis pada setiap push ke branch `main` dan `develop`, serta pada setiap pull request.

### Menjalankan Coverage Secara Lokal

```bash
# Menjalankan test dengan coverage
make test-ci

# Atau dengan output HTML untuk detail coverage
make test-coverage
```

Hasil coverage akan disimpan di:
- `coverage.out` - Format Go coverage (untuk CI/Codecov)
- `coverage.html` - Format HTML untuk dilihat di browser

### Melihat Coverage Report

Setelah menjalankan `make test-coverage`, buka file `coverage.html` di browser:

```bash
# macOS
open coverage.html

# Linux
xdg-open coverage.html
```

---

## Screenshots

Screenshots tersedia di folder [docs/screenshots/](docs/screenshots/).

<!-- Uncomment dan sesuaikan setelah menambahkan screenshot
### Dashboard
![Dashboard](docs/screenshots/dashboard.png)

### Terminal
![Terminal](docs/screenshots/terminal.png)

### App Management
![App Management](docs/screenshots/app-management.png)
-->

---

## License

MIT
