# 🚀 AcmaDash - Premium VPS Management System

**AcmaDash** is a lightweight, high-performance, and secure VPS management dashboard. The system is designed with a **Go (Gin)** backend that directly embeds a **React (Vite + TypeScript + TailwindCSS + Shadcn/UI)** frontend into a single executable binary.

---

## ✨ Key Features

### 1. 📊 Real-Time System Monitoring
- Updates hardware metrics continuously using **SSE (Server-Sent Events)**: CPU, RAM, Disk, Network I/O, and active TCP connections.
- Displays system Uptime, Hostname, OS, Platform, and Kernel version.
- **Telegram Alert Integration**: Automatically sends alert notifications to a designated Telegram chat when CPU usage exceeds 90% or TCP connections exceed 2000 (indicating a potential DDoS attack).

### 2. 📝 Unified Log Viewer
- Views system logs (`/var/log/syslog`) in real time.
- Automatically scans and displays HTTP access (`access.log`) and error (`error.log`) logs for each Nginx site configured in `/etc/nginx/sites-enabled`.

### 3. ⚙️ Process Manager
- Lists the top 10 running processes consuming the most system resources (CPU/RAM).

### 4. 🐳 Docker Container Management
- Lists active Docker containers showing their real-time resource utilization (CPU, RAM, Image, and Status).
- Includes a simulation mode for developers running on Windows.

### 5. 🟢 PM2 Process Manager
- Lists Node.js applications managed by PM2.
- Supports remote control actions (Start, Stop, Restart, Delete) directly from the dashboard.

### 6. 🌐 Domain & Webspace Management
- Scans and checks the HTTP status of virtual hosts configured under Nginx sites.
- Allows managing quick annotations/notes for each domain.
- **Safe Domain Deletion**:
  - Gathers database and source path details.
  - Automatically drops associated MySQL databases by parsing Laravel `.env` files.
  - Deletes site configuration files and related Nginx logs.
  - Deletes the source directories safely (strictly limited to designated paths: `/var/www`, `/home`, `/srv/www`, `/opt`).
  - Reloads Nginx automatically (`nginx -s reload`).

---

## 🛠️ Tech Stack

### Backend
- **Language**: Go (Golang)
- **Web Framework**: [Gin Gonic](https://github.com/gin-gonic/gin)
- **System Metrics**: [gopsutil](https://github.com/shirou/gopsutil)
- **Asset Embedding**: Native Go `embed` package
- **OS Support**: Linux (production environment) and Windows (simulation/development).

### Frontend
- **Framework**: React 18 + Vite + TypeScript
- **CSS / UI**: Tailwind CSS + Shadcn/UI + Lucide Icons
- **Charts**: Recharts

---

## 📂 Project Directory Structure

```text
├── .agent/               # Agent Workspace (Instructions, Persona, Memory, Workflows)
├── frontend/             # React Frontend source code (Vite, TS, Tailwind CSS)
│   ├── src/              # React components & UI logic
│   ├── dist/             # Built assets (embedded into the Go binary)
│   └── package.json
├── scripts/
│   └── install.sh        # Shell script for automated installation/updates
├── .env.example          # Environment template file
├── go.mod                # Go module dependencies
├── go.sum
├── main.go               # Backend entry point
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

*What the installer script does:*
1. Identifies the system architecture (amd64 / arm64).
2. Downloads the latest release package directly from GitHub.
3. Installs the binary to `/usr/local/bin/vps-dash`.
4. Configures and registers a systemd service: `vps-dashboard.service`.
5. Starts the dashboard on port `8900`.

---

### Method 2: Building from Source

#### Prerequisites:
- **Go**: Version 1.21 or higher
- **Node.js & npm**: Required to build frontend assets

#### Steps:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/acmavirus/vps-dashboard-1-install.git
   cd vps-dashboard-1-install
   ```

2. **Build the project using Makefile**:
   ```bash
   make build
   ```
   *Note: This command installs frontend dependencies, builds static files, runs `go mod tidy`, and compiles the Go backend binary.*

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

> [!WARNING]
> If `ADMIN_USER`, `ADMIN_PASS`, and `AUTH_TOKEN` are not defined in the `.env` file, the dashboard falls back to the default credentials shown above. Please change them before deploying to a production server.

---

## 🔒 Security & Scope Restrictions

1. **Token Authentication**: All API endpoints require authorization using the token passed via the `Authorization` header or `token` query parameter (for SSE streams).
2. **Action Safety Bounds**: Administrative actions (such as service restarts) are restricted to standard system services (`nginx`, `php`, `mysql`). Folder and database deletion targets are sanitized and constrained to safe paths to prevent accidental or malicious data loss.

---

## 📄 License

Developed by **AcmaVirus**. All rights reserved to the repository owner.
Please do not use for unauthorized commercial purposes.
