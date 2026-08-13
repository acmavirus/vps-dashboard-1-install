#!/bin/bash

# --- 1. Khai báo thông tin ---
REPO="acmavirus/vps-dashboard-1-install"
BINARY_NAME="vps-dash"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="vps-dashboard"

# Màu sắc thông báo
GREEN='\033[0;32m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${CYAN}------------------------------------------------------------${NC}"
echo -e "${CYAN}AcmaDash - Premium VPS Management System - Unified Installer${NC}"
echo -e "${CYAN}------------------------------------------------------------${NC}"

# Kiểm tra trạng thái hiện tại (Install hay Update)
IS_UPDATE=0
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    IS_UPDATE=1
    CURRENT_VERSION=$($INSTALL_DIR/$BINARY_NAME -v | awk '{print $NF}')
    echo -e "${GREEN}Hệ thống đã tồn tại (Phiên bản: $CURRENT_VERSION). Chế độ: UPDATE.${NC}"
else
    echo -e "${GREEN}Môi trường sạch. Chế độ: NEW INSTALL.${NC}"
fi

echo -e "${GREEN}Đang lấy thông tin bản phát hành mới nhất từ GitHub...${NC}"
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo -e "${RED}Lỗi: Không tìm thấy Release nào trên GitHub!${NC}"
    exit 1
fi

if [ "$IS_UPDATE" -eq 1 ] && [ "$VERSION" == "$CURRENT_VERSION" ]; then
    echo -e "${CYAN}Bạn đã ở phiên bản mới nhất ($VERSION). Vẫn tiếp tục cài đặt đè...${NC}"
fi

# Xác định kiến trúc hệ thống
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    FILE_ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    FILE_ARCH="arm64"
else
    echo -e "${RED}Lỗi: Kiến trúc $ARCH chưa được hỗ trợ.${NC}"
    exit 1
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}-linux-${FILE_ARCH}"

# Dừng service nếu đang chạy (đặc biệt quan trọng khi update)
if [ "$IS_UPDATE" -eq 1 ]; then
    echo -e "${GREEN}Đang tạm dừng service hiện tại để cập nhật...${NC}"
    systemctl stop $SERVICE_NAME > /dev/null 2>&1
fi

echo -e "${GREEN}Đang tải xuống phiên bản $VERSION cho $FILE_ARCH...${NC}"
curl -L -o $INSTALL_DIR/$BINARY_NAME "$DOWNLOAD_URL"
chmod +x $INSTALL_DIR/$BINARY_NAME

# Kiểm tra nếu chưa có systemd service thì tạo mới
if [ ! -f "/etc/systemd/system/$SERVICE_NAME.service" ]; then
    echo -e "${GREEN}Đang cấu hình Systemd Service...${NC}"
    cat <<EOF > /etc/systemd/system/$SERVICE_NAME.service
[Unit]
Description=Premium VPS Management Dashboard
After=network.target

[Service]
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=always
User=root
WorkingDirectory=/usr/local/bin

[Install]
WantedBy=multi-user.target
EOF
fi

# Cấu hình Nginx Default Catch-All Page (xử lý domain chưa gán vhost cho cả HTTP & HTTPS)
if command -v nginx > /dev/null 2>&1; then
    echo -e "${GREEN}Đang cấu hình Nginx Default Catch-All Page (HTTP & HTTPS)...${NC}"
    mkdir -p /var/www/default
    mkdir -p /etc/nginx/ssl

    if [ ! -f "/etc/nginx/ssl/default.crt" ]; then
        openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
            -keyout /etc/nginx/ssl/default.key \
            -out /etc/nginx/ssl/default.crt \
            -subj "/CN=default" > /dev/null 2>&1 || true
    fi

    if [ ! -f "/var/www/default/index.html" ]; then
        cat <<'EOF' > /var/www/default/index.html
<!DOCTYPE html>
<html lang="vi">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hệ Thống Đang Bảo Trì & Sắp Ra Mắt</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background-color: #0f172a;
            color: #f8fafc;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: 24px;
            overflow: hidden;
            position: relative;
        }
        .bg-glow {
            position: absolute;
            width: 400px;
            height: 400px;
            background: radial-gradient(circle, rgba(56, 189, 248, 0.15) 0%, rgba(99, 102, 241, 0.05) 50%, transparent 70%);
            border-radius: 50%;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            animation: pulse 6s ease-in-out infinite alternate;
            pointer-events: none;
        }
        @keyframes pulse {
            0% { transform: translate(-50%, -50%) scale(0.8); opacity: 0.5; }
            100% { transform: translate(-50%, -50%) scale(1.2); opacity: 1; }
        }
        .card {
            position: relative;
            background: rgba(30, 41, 59, 0.7);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 20px;
            padding: 48px 36px;
            max-width: 520px;
            width: 100%;
            text-align: center;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
            z-index: 10;
        }
        .badge {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            background: rgba(56, 189, 248, 0.1);
            border: 1px solid rgba(56, 189, 248, 0.3);
            color: #38bdf8;
            font-size: 13px;
            font-weight: 600;
            padding: 6px 14px;
            border-radius: 9999px;
            margin-bottom: 24px;
            letter-spacing: 0.5px;
            text-transform: uppercase;
        }
        .badge-dot {
            width: 8px;
            height: 8px;
            background-color: #38bdf8;
            border-radius: 50%;
            animation: blink 1.5s infinite;
        }
        @keyframes blink {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.3; }
        }
        .icon {
            font-size: 56px;
            margin-bottom: 20px;
            display: inline-block;
            animation: float 3s ease-in-out infinite;
        }
        @keyframes float {
            0%, 100% { transform: translateY(0); }
            50% { transform: translateY(-8px); }
        }
        h1 {
            font-size: 26px;
            font-weight: 700;
            background: linear-gradient(135deg, #ffffff 0%, #cbd5e1 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 14px;
            line-height: 1.3;
        }
        p {
            color: #94a3b8;
            font-size: 15px;
            line-height: 1.7;
            margin-bottom: 28px;
        }
        .footer-text {
            font-size: 13px;
            color: #64748b;
            border-top: 1px solid rgba(255, 255, 255, 0.06);
            padding-top: 20px;
        }
    </style>
</head>
<body>
    <div class="bg-glow"></div>
    <div class="card">
        <div class="badge">
            <span class="badge-dot"></span>
            Hệ Thống Đang Nâng Cấp
        </div>
        <div class="icon">🚀</div>
        <h1>Website Đang Bảo Trì & Sắp Ra Mắt</h1>
        <p>Hệ thống hiện đang được bảo trì định kỳ và nâng cấp tính năng mới. Chúng tôi sẽ trở lại trong thời gian sớm nhất!</p>
        <div class="footer-text">
            Cảm ơn sự kiên nhẫn của bạn. Vui lòng quay lại sau ít phút.
        </div>
    </div>
</body>
</html>
EOF
    fi

    cat <<'EOF' > /etc/nginx/sites-available/00-default.conf
server {
    listen 80 default_server;
    listen [::]:80 default_server;

    server_name _;
    charset utf-8;

    root /var/www/default;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    access_log /var/log/nginx/default_access.log;
    error_log /var/log/nginx/default_error.log;
}

server {
    listen 443 ssl default_server;
    listen [::]:443 ssl default_server;

    server_name _;
    charset utf-8;

    ssl_certificate /etc/nginx/ssl/default.crt;
    ssl_certificate_key /etc/nginx/ssl/default.key;

    root /var/www/default;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    access_log /var/log/nginx/default_access.log;
    error_log /var/log/nginx/default_error.log;
}
EOF

    rm -f /etc/nginx/sites-enabled/default
    ln -sf /etc/nginx/sites-available/00-default.conf /etc/nginx/sites-enabled/00-default.conf
    nginx -t > /dev/null 2>&1 && systemctl reload nginx > /dev/null 2>&1 || true
fi

# Kích hoạt và Khởi động lại
systemctl daemon-reload
systemctl enable $SERVICE_NAME > /dev/null 2>&1
systemctl restart $SERVICE_NAME

echo -e "${GREEN}------------------------------------------------------------${NC}"
if [ "$IS_UPDATE" -eq 1 ]; then
    echo -e "${GREEN}CẬP NHẬT THÀNH CÔNG LÊN PHIÊN BẢN $VERSION!${NC}"
else
    echo -e "${GREEN}CÀI ĐẶT THÀNH CÔNG PHIÊN BẢN $VERSION!${NC}"
fi
echo -e "${GREEN}Truy cập Dashboard tại: http://$(curl -s ifconfig.me):8900${NC}"
echo -e "${GREEN}------------------------------------------------------------${NC}"
