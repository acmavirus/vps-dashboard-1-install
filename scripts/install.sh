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
