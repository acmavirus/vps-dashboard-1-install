# 🚀 Kế hoạch Nâng cấp Toàn diện — AcmaDash v3.0

## 📋 Sơ lược Dự án Hiện tại

**AcmaDash** là VPS Management Dashboard mã nguồn mở, được viết bằng **Go (Gin)** backend + **Svelte 4 + TailwindCSS** frontend, đóng gói thành một **binary duy nhất** qua `go:embed`.

### Stack kỹ thuật
| Lớp | Công nghệ |
|---|---|
| Backend | Go 1.24, Gin v1.10, gopsutil v4, go-sql-driver/mysql |
| Frontend | Svelte 4, Vite 5, TypeScript, TailwindCSS 3, Lucide-Svelte |
| Embedding | `go:embed all:frontend/dist` |
| Target OS | Linux (prod) + Windows (dev/sim) |

### Tính năng hiện có (v2.2.5)
- ✅ **Overview Tab**: 3 dial gauge (Load/CPU/RAM), network chart SVG, Disk, System Info, Service control
- ✅ **Domains Tab**: Quản lý website Nginx, tạo site (Static/PHP/Proxy), Delete domain + DB, Star/Note
- ✅ **Files Tab**: Web file manager, text editor modal, path shortcuts
- ✅ **Databases Tab**: MySQL/MariaDB management, backup via mysqldump
- ✅ **Security Tab**: UFW Firewall + IPS Auto-Ban engine
- ✅ **App Store Tab**: Docker app one-click install (WordPress, Joomla, Drupal, Ghost, PrestaShop, Redis, PostgreSQL, MongoDB)
- ✅ **FTP Tab**, **Cron Tab**, **Logs Tab**, **Nodes (PM2) Tab**, **Monitor Tab**, **Docker Tab**, **Settings Tab**
- ✅ Multi-theme (6 themes: aaPanel Green, Slate Onyx, Aurora Violet, Nordic Forest, Oceanic Abyss, Sunset Amber)
- ✅ SSE real-time streaming, Token Auth, Telegram Alerts

---

## 🔍 Phân tích Vấn đề & Cơ hội Cải thiện

### ❌ Vấn đề UX/Design hiện tại
1. **Software card cứng (hardcoded)**: Nginx 1.22.1, PHP 8.3, PHP 7.4, MariaDB — không lấy từ API thực, version tĩnh
2. **`alert()` / `confirm()` native browser**: Toàn bộ feedback sử dụng `window.alert()` và `window.confirm()` — không đẹp, chặn UI
3. **Copyright footer**: "aaPanel design emulation" — nên là AcmaDash branding
4. **DockerTab stub**: `DockerTab.svelte` chỉ 1.4KB — hiển thị containers thô, thiếu actions
5. **ProcessesTab stub**: `ProcessesTab.svelte` chỉ 1.3KB — chỉ hiển thị list đơn giản
6. **DomainDetailTab**: Có nhưng chưa tích hợp đầy đủ với DomainInfo mới
7. **DatabaseExplorer.svelte**: 28KB — component riêng nhưng chưa được tích hợp hoàn chỉnh
8. **Poll mọi 3s**: `poll()` gọi cùng lúc 6 API endpoint — gây overhead
9. **Backend monolith**: `main.go` 4885 dòng — cần tách module
10. **Không có toast notification system** — UX cần cải thiện

### 🆕 Tính năng thiếu theo roadmap
- SSL Manager (Let's Encrypt tích hợp)
- Backup tự động lên Cloud
- 2FA Authentication
- Database Explorer (SQL query UI)
- Terminal/SSH in browser (xterm.js)
- WebSocket thay SSE (optional)

---

## ✅ Phạm vi Nâng cấp v3.0

> [!IMPORTANT]
> Kế hoạch được chia 3 giai đoạn. Mỗi giai đoạn là một PR độc lập, có thể deploy riêng lẻ.

---

## 📐 Giai đoạn 1: Toast System + UX Polish (Ưu tiên cao nhất)

### Mục tiêu
Thay thế toàn bộ `alert()` / `confirm()` bằng toast notification system đẹp, hiện đại.

### Proposed Changes

---

#### Frontend

##### [NEW] `frontend/src/lib/toast.ts`
- Store Svelte cho toast queue
- Types: `success | error | warning | info`
- Auto-dismiss sau 4s, manual dismiss

##### [NEW] `frontend/src/components/Toast.svelte`
- Toast container fixed bottom-right
- Slide-in animation (svelte `fly` transition)
- Icon theo type, màu theo theme
- Max 5 toasts stack

##### [MODIFY] `frontend/src/App.svelte`
- Import `Toast.svelte`, mount global
- Thay `alert()` / `confirm()` bằng toast + confirm modal đẹp
- Confirm modal: glassmorphism card, animated

##### [MODIFY] Tất cả Tab components
- Thay `alert(...)` → `toast.success(...)` / `toast.error(...)`
- Thay `confirm(...)` → `showConfirm(message, callback)`

---

## 📐 Giai đoạn 2: Feature Upgrade — Docker, Process, SSL, Software Detection

### 2A. Docker Tab Upgrade

##### [MODIFY] `frontend/src/components/dashboard/DockerTab.svelte`
- Hiển thị containers dạng card grid đẹp (có icon, CPU/RAM bar)
- Actions: Start / Stop / Restart / Remove cho từng container
- Link sang App Store nếu container thuộc managed app

##### Backend: `main.go`
- `POST /api/docker/control` — start/stop/restart container theo name

### 2B. Process Monitor Upgrade

##### [MODIFY] `frontend/src/components/dashboard/ProcessesTab.svelte`
- Top 15 processes thay vì 10
- Bar chart mini cho CPU/RAM
- Kill process action (với confirm)
- Search/filter process name

##### Backend: `main.go`
- `POST /api/processes/kill` — kill PID (chỉ cho non-system processes)

### 2C. Software Detection (OverviewTab)

##### [MODIFY] `frontend/src/components/dashboard/OverviewTab.svelte`
- Software card đọc từ API thực (version thực từ `nginx -v`, `php --version`)
- Không hard-code version string

##### Backend: `main.go`
- `GET /api/software` — detect Nginx, PHP versions, MariaDB version, Redis status
- Cache 60s

### 2D. SSL Manager

##### [NEW] `frontend/src/components/dashboard/SSLTab.svelte`
- Danh sách SSL certificates theo domain
- Status: valid/expiring/expired với countdown
- Actions: Renew (gọi certbot), View cert details

##### Backend: `main.go`
- `GET /api/ssl` — scan `/etc/letsencrypt/live/` + parse expiry dates
- `POST /api/ssl/renew` — chạy `certbot renew --cert-name <domain>`

##### [MODIFY] `frontend/src/App.svelte`
- Thêm SSL tab vào `appTabsExtended`

---

## 📐 Giai đoạn 3: Performance + Architecture + Backend Refactor

### 3A. Backend Module Split

##### Tách `main.go` thành các file riêng:
- `handlers_auth.go` — login, logout, token management
- `handlers_system.go` — stats, stream SSE, processes
- `handlers_domains.go` — domain CRUD, nginx config
- `handlers_docker.go` — docker management
- `handlers_database.go` — MySQL operations
- `handlers_security.go` — firewall, IPS
- `handlers_files.go` — file manager
- `handlers_apps.go` — app store
- `handlers_cron.go` — cron management
- `helpers.go` — shared utilities
- `types.go` — all struct definitions

> [!WARNING]
> Đây là refactor lớn nhất. Cần test toàn diện trước khi deploy production.

### 3B. SSE Optimization

##### [MODIFY] `main.go` — stream endpoint
- Tách SSE thành lightweight (chỉ push stats thực sự thay đổi)
- Thêm delta compression: chỉ gửi field thay đổi
- Giảm poll interval frontend từ 3s → 5s cho non-critical data

### 3C. Caching Layer

##### [MODIFY] `main.go`
- Add in-memory cache struct với TTL
- Cache: `/api/software` (60s), `/api/domains` (30s khi không scan), `/api/processes` (5s)

### 3D. DatabaseExplorer Integration

##### [MODIFY] `frontend/src/components/dashboard/DatabasesTab.svelte`
- Tích hợp `DatabaseExplorer.svelte` đã có nhưng chưa hook đầy đủ
- SQL query runner UI: textarea → execute → result table
- Table browser per database

---

## 🎨 Design System Upgrades (Cross-cutting)

### Typography & Spacing
- Upgrade font từ `Inter` → `Inter + JetBrains Mono` (monospace cho code, terminal)
- Cải thiện spacing consistency (đồng bộ padding trên tất cả cards)

### Animation Upgrades
- Thêm `animate-pulse` cho live indicators
- Smooth số counter animation (animatable numbers cho CPU/RAM %)
- Page transition khi switch tab

### Micro-interactions
- Hover state nhất quán trên tất cả buttons
- Loading skeleton cho tất cả fetch operations (thay spinner)
- Empty state illustrations (SVG inline)

---

## 🔒 Security Fixes

### Input Validation
- Validate tất cả API inputs với strict regex
- Rate limiting cho login endpoint (chống brute-force)
- CSRF token cho POST requests

### Auth Improvement
- Token expiry (hiện tại token không hết hạn)
- Refresh token mechanism
- Session invalidation khi đổi password

---

## 📋 Open Questions

> [!IMPORTANT]
> Cần xác nhận trước khi bắt đầu:

1. **Giai đoạn nào ưu tiên triển khai trước?** (Đề xuất: Giai đoạn 1 trước — Toast System là nền tảng cho tất cả)
2. **Backend refactor có cần không?** Nếu `main.go` quá lớn ảnh hưởng đến DX thì tách, nhưng đây là rủi ro lớn nhất
3. **SSL Tab có cần không?** Nếu dùng certbot manual thì không cần UI
4. **Terminal/SSH trong browser** (xterm.js) — có muốn thêm vào scope không?
5. **Database Explorer** — SQL query runner có trong scope v3.0 không?

---

## 🧪 Verification Plan

### Automated Tests
```powershell
cd d:\laragon\www_thuc\vps-dashboard-1-install
go test ./...
```

### Frontend Build Test
```powershell
cd frontend
npm run build
```

### Manual Verification
- Đăng nhập, kiểm tra toast notification thay alert
- Kiểm tra mỗi tab hoạt động đúng sau upgrade
- Kiểm tra theme switching
- Kiểm tra responsive mobile
- Test trên VPS Linux thực (port 8900)

---

## 📊 Ước tính Bundle Size

| Phase | Frontend Bundle | Delta |
|---|---|---|
| Hiện tại v2.2.5 | ~142-230 KB (varies) | baseline |
| + Toast System | +8 KB | nhỏ |
| + Docker/Process Upgrade | +5 KB | nhỏ |
| + SSL Tab | +12 KB | nhỏ |
| v3.0 target | < 280 KB | acceptable |
