# 🚀 AcmaDash - Premium VPS Management System (v3.0.0)

**AcmaDash** is a lightweight, high-performance, and secure VPS management dashboard. The system is designed with a modular **Go (Gin)** backend that directly embeds a **Svelte 4 (Vite 5 + TypeScript + TailwindCSS)** frontend into a single executable binary.

---

## ✨ Key Features

### 1. 📊 Real-Time System Monitoring (Optimized SSE)
- Updates hardware metrics continuously using an optimized **SSE (Server-Sent Events)** loop (2s interval, lightweight stats-only payload to minimize disk I/O).
- Displays system Uptime, Hostname, OS, Platform, Kernel version, CPU, RAM, Disk, and Network I/O.
- **Telegram Alert Integration**: Automatically sends alert notifications to a designated Telegram chat when CPU usage exceeds 90% or TCP connections exceed 2000 (indicating a potential DDoS attack).

### 2. 📝 Unified Log Viewer
- Views system logs (`/var/log/syslog`) in real time.
- Automatically scans and displays HTTP access and error logs for each Nginx virtual host.

### 3. ⚙️ Process Manager (Top 15 & Control)
- Lists the top 15 running processes consuming the most system resources (CPU/RAM).
- Supports terminating (killing) processes directly from the UI (with safe exclusions for critical system services).

### 4. 🐳 Docker Container Management (Full Control)
- Lists active Docker containers showing their real-time resource utilization (CPU, RAM, Image, and Status).
- Allows starting, stopping, restarting, and removing containers directly from the UI.
- Automatically handles database, volume, and Nginx proxy cleanup when uninstalling managed apps.

### 5. 🟢 PM2 Process Manager
- Lists Node.js applications managed by PM2.
- Supports PM2 status monitoring directly from the dashboard.

### 6. 🌐 Domain & Webspace Management (SSL Integrated)
- Scans and checks the HTTP status of virtual hosts configured under Nginx sites.
- Allows managing quick annotations/notes and starring domains.
- **SSL Manager**: Scans Let's Encrypt certificates, shows days remaining, and supports one-click renewal (`certbot renew`) from the UI.
- **Safe Domain Deletion**: Drops associated MySQL databases by parsing Laravel `.env` files, deletes Nginx configs, site directories, and reloads Nginx.

### 7. 🗄️ Database Explorer & SQL Runner (New)
- Explores MySQL/MariaDB database list and table schemas (Engine, collation, rows count, data size).
- Inspects table column definitions (types, primary keys, defaults).
- **SQL Editor**: Executes raw custom SQL queries (SELECT, INSERT, UPDATE, DELETE) directly from the browser with dynamic results table.

---

## 🛠️ Tech Stack

### Backend
- **Language**: Go 1.24
- **Web Framework**: Gin Gonic v1.10
- **System Metrics**: gopsutil v4
- **Architecture**: Modular layout (split handlers, types, helpers)
- **Caching Layer**: In-memory cache with custom TTLs (Software: 60s, Domains: 30s, Processes: 5s)
- **Asset Embedding**: Native Go `embed` package

### Frontend
- **Framework**: Svelte 4 + Vite 5 + TypeScript
- **CSS / UI**: Tailwind CSS 3 + Lucide Icons
- **Branding**: AcmaDash v3.0

---

## 📂 Project Directory Structure

```text
├── .agent/               # Agent Workspace (Instructions, Persona, Memory, Workflows)
├── frontend/             # Svelte Frontend source code
│   ├── src/              # Svelte components & UI logic
│   ├── dist/             # Built assets (embedded into the Go binary)
│   └── package.json
├── scripts/
│   ├── deploy-vps.ps1    # Automated deployment script for Windows to Linux VPS
│   └── install.sh        # Shell script for automated installation/updates
├── .env.example          # Environment template file
├── go.mod                # Go module dependencies
├── go.sum
├── main.go               # Backend entry point (bootstrapper & routing)
├── types.go              # Shared structure definitions
├── helpers.go             # Global helper and utility functions
├── handlers_*.go         # Modular domain handlers (auth, system, docker, domains, database, files, apps, cron, security)
├── Makefile              # Automation tool for building and cleaning the project
└── README.md
```

---

## 🚀 Installation & Setup

### Method 1: Quick Installer (Recommended for Linux VPS)

Run the following command to download the latest release binary, configure the systemd service, and launch the dashboard:

```bash
curl -sSL https://raw.githubusercontent.com/acmavirus/vps-dashboard-1-install/main/scripts/install.sh | sudo bash
```

---

### Method 2: Building from Source

#### Prerequisites:
- **Go**: Version 1.24 or higher
- **Node.js & npm**: Required to build frontend assets

#### Steps:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/acmavirus/vps-dashboard-1-install.git
   cd vps-dashboard-1-install
   ```

2. **Build the project**:
   ```bash
   # Build frontend static files first
   cd frontend && npm install && npm run build && cd ..
   # Build Go binary
   go build -o vps-dash .
   ```

3. **Run the dashboard**:
   ```bash
   ./vps-dash
   ```

---

## ⚙️ Environment Configuration (.env)

Configure your `.env` file in the binary's root directory:

```env
# Server Port (Default: 8900)
PORT=8900

# Gin Framework Mode (release / debug)
GIN_MODE=release

# Admin Credentials
ADMIN_USER=admin
ADMIN_PASS=h5jH7Gv|5m+0

# API Authorization Token (Change this to secure your endpoints)
AUTH_TOKEN=acmadash_secret_token_2024

# Telegram Alerts Configuration (Optional)
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
TELEGRAM_CHAT_ID=your_telegram_chat_id
```

---

## 📄 License

Developed by **AcmaVirus**. All rights reserved to the repository owner.
Please do not use for unauthorized commercial purposes.
