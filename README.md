# 🚀 AcmaDash v3.0.0 - Premium VPS Management System

**AcmaDash** is a lightweight, high-performance, and secure VPS management dashboard. The system is designed with a **Go (Gin)** backend that directly embeds a **Svelte 4 (Vite + TypeScript + TailwindCSS)** frontend into a single executable binary.

---

## ✨ Key Features (v3.0.0 Updates)

### 1. 📊 Real-Time System Monitoring
- Updates hardware metrics continuously using **SSE (Server-Sent Events)**: CPU, RAM, Disk, Network I/O, and active TCP connections.
- **SSE & Disk I/O Optimization**: Real-time stream is optimized to transmit lightweight resource metrics only, completely eliminating disk log reads during streaming.
- Displays system Uptime, Hostname, OS, Platform, and Kernel version.
- **Telegram Alert Integration**: Automatically sends alert notifications to a designated Telegram chat when CPU usage exceeds 90% or TCP connections exceed 2000 (indicating a potential DDoS attack).

### 2. 🌐 Domain & Webspace Management
- Scans and checks the HTTP status of virtual hosts configured under Nginx sites.
- Allows managing quick annotations/notes for each domain.
- **Safe Domain Deletion**:
  - Gathers database and source path details.
  - Automatically drops associated MySQL databases by parsing Laravel `.env` files.
  - Deletes site configuration files and related Nginx logs.
  - Deletes the source directories safely (strictly limited to designated paths: `/var/www`, `/home`, `/srv/www`, `/opt`).
  - Reloads Nginx automatically (`nginx -s reload`).

### 3. 🛡️ SSL Manager Tab (Let's Encrypt Integration)
- Automatically scans `/etc/letsencrypt/live/` configurations.
- Extracts and parses the `fullchain.pem` using Go's native `x509` parser to retrieve the Issuer, expiration date, and remaining validity days without calling external tools.
- Supports manual/automatic **SSL Renewal** (triggers `certbot renew --cert-name <domain>`) directly from the UI.

### 4. ⚙️ Process Manager (Enhanced)
- Lists the **Top 15** running processes consuming the most system resources (CPU/RAM).
- Supports real-time process search, filtering, and terminates target processes via `Kill PID` (system critical processes and the dashboard process are safe-blocked from being killed).
- Fully integrated with a premium verification `ConfirmModal`.

### 5. 🐳 Docker Container Management
- Displays active Docker containers in a premium grid layout with real-time CPU/RAM progress bars.
- Supports lifecycle control actions (**Start**, **Stop**, **Restart**, and **Remove**) directly from the UI.
- Integrated database drop and proxy cleanups when containerized applications are uninstalled.

### 6. 🗄️ Database Explorer (SQL Query Runner)
- Inspects tables metadata (Rows, Data Size, Engine, Collation, and Comments).
- Visualizes table structures (field types, nullability, keys, default values, and extras).
- Features an interactive **SQL Query Editor** to execute arbitrary queries, rendering dynamic SELECT tables or transaction reports (affected rows, insert IDs) with ease.

### 7. 🟢 PM2 Process Manager & Cronjobs
- Lists and manages Node.js applications powered by PM2.
- Lists, adds, deletes, toggles, and views logs for standard system/custom Cron jobs.

### 8. 📂 Web File Manager
- A full-featured web-based file manager (Browse, Read, Write/Save, Create, Delete, and Rename files or directories).
- Restricted deletion rules for critical system paths (`/`, `/etc`, `/root`, `/usr`, `/var`, `/bin`) to enforce administration safety.

### 9. 🎨 Design System & UX Upgrades
- Custom **Toast Notification System** (`success`, `error`, `warning`, `info`) and glassmorphism **ConfirmModal** replacing native browser popups.
- Dynamic color themes and beautiful animations.
- Smart **In-Memory Cache Layer** for `/api/software` (60s), `/api/domains` (30s), and `/api/processes` (5s) endpoints.

---

## 🛠️ Tech Stack

### Backend
- **Language**: Go 1.24 (Golang)
- **Web Framework**: [Gin Gonic](https://github.com/gin-gonic/gin)
- **System Metrics**: [gopsutil v4](https://github.com/shirou/gopsutil)
- **Asset Embedding**: Native Go `embed` package
- **Architecture**: Modular structure (Refactored `main.go` monolith into dedicated handler packages).

### Frontend
- **Framework**: [Svelte 4](https://svelte.dev/) + Vite + TypeScript
- **CSS / UI**: Tailwind CSS + Lucide Icons
- **Transitions**: Native Svelte spring/fly motion

---

## 📂 Project Directory Structure

```text
├── .agent/               # Agent Workspace (Instructions, Persona, Memory, Workflows)
├── frontend/             # Svelte 4 Frontend source code
│   ├── src/              # Svelte components & UI state logic
│   ├── dist/             # Built assets (embedded into the Go binary)
│   └── package.json
├── scripts/
│   ├── install.sh        # Shell script for automated installation/updates on VPS
│   └── deploy-vps.ps1    # PowerShell script for build & remote SCP deploy
├── .env.example          # Environment template file
├── go.mod                # Go module dependencies
├── main.go               # Application bootstrapper and routing coordinator
├── types.go              # Shared Struct definitions
├── helpers.go            # General utilities (SQL commands, Telegram sending)
├── handlers_auth.go      # Middleware token auth & login routes
├── handlers_system.go    # Stats, PM2, process monitor, settings, and SSE stream
├── handlers_docker.go    # Docker container lifecycle API
├── handlers_domains.go   # Nginx site manager & SSL scan/renew API
├── handlers_database.go  # Database explorer & custom query executor
├── handlers_security.go  # Firewall (UFW) configurations & IPS Auto-Ban engine
├── handlers_files.go     # Web file manager actions
├── handlers_apps.go      # One-click Docker application installer
└── handlers_cron.go      # System Cronjob manager
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
- **Go**: Version 1.24 or higher
- **Node.js & npm**: Required to build frontend assets

#### Steps:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/acmavirus/vps-dashboard-1-install.git
   cd vps-dashboard-1-install
   ```

2. **Build the project**:
   - First build the frontend assets:
     ```bash
     cd frontend
     npm install
     npm run build
     cd ..
     ```
   - Compile the Go backend binary:
     ```bash
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
