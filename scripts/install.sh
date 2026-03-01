#!/bin/bash

# --- 1. Khai báo thông tin ---
REPO="acmavirus/vps-dashboard-1-install" # Thay bằng repo thực tế của bạn
BINARY_NAME="vps-dash"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="vps-dashboard"

# Màu sắc thông báo
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Đang tìm kiếm bản phát hành (Release) mới nhất trên GitHub...${NC}"

# Tự động lấy phiên bản mới nhất từ API GitHub
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo -e "${RED}Lỗi: Không tìm thấy Release nào trên repo $REPO!${NC}"
    exit 1
fi

# Xác định kiến trúc hệ thống (amd64 hoặc arm64)
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

echo -e "${GREEN}Đang tải xuống phiên bản $VERSION bản cho $FILE_ARCH...${NC}"
curl -L -o $INSTALL_DIR/$BINARY_NAME "$DOWNLOAD_URL"
chmod +x $INSTALL_DIR/$BINARY_NAME

# --- 2. Tạo Systemd Service để chạy ngầm và tự khởi động cùng VPS ---
echo -e "${GREEN}Đang cấu hình hệ thống (systemd)...${NC}"
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

# --- 3. Kích hoạt và Chạy ---
systemctl daemon-reload
systemctl enable $SERVICE_NAME
systemctl restart $SERVICE_NAME

echo -e "${GREEN}------------------------------------------------------------${NC}"
echo -e "${GREEN}CÀI ĐẶT THÀNH CÔNG!${NC}"
echo -e "${GREEN}Truy cập Dashboard tại: http://$(curl -s ifconfig.me):8900${NC}"
echo -e "${GREEN}------------------------------------------------------------${NC}"
