param(
    [string]$HostName = "15.235.199.163",
    [int]$Port = 2404,
    [string]$User = "root",
    [string]$SshKey = "",
    [string]$ServiceName = "vps-dashboard",
    [string]$RemoteBinary = "/usr/local/bin/vps-dash",
    [string]$LocalBinary = "vps-dash-linux-amd64",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

function Invoke-Remote {
    param([string]$Command)

    $sshArgs = @()
    if ($SshKey.Trim() -ne "") {
        $sshArgs += @("-i", $SshKey)
    }
    $sshArgs += @("-p", "$Port", "$User@$HostName", $Command)

    & ssh @sshArgs
    if ($LASTEXITCODE -ne 0) {
        throw "Remote command failed with exit code $LASTEXITCODE"
    }
}

function Copy-ToRemote {
    param(
        [string]$Source,
        [string]$Target
    )

    $scpArgs = @()
    if ($SshKey.Trim() -ne "") {
        $scpArgs += @("-i", $SshKey)
    }
    $scpArgs += @("-P", "$Port", $Source, "$User@$HostName`:$Target")

    & scp @scpArgs
    if ($LASTEXITCODE -ne 0) {
        throw "SCP failed with exit code $LASTEXITCODE"
    }
}

if (-not $SkipBuild) {
    Push-Location "frontend"
    try {
        npm run build
    } finally {
        Pop-Location
    }

    go test ./...

    $oldGoos = $env:GOOS
    $oldGoarch = $env:GOARCH
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    try {
        go build -ldflags "-s -w" -o $LocalBinary .
    } finally {
        $env:GOOS = $oldGoos
        $env:GOARCH = $oldGoarch
    }
}

if (-not (Test-Path $LocalBinary)) {
    throw "Local binary not found: $LocalBinary"
}

$remoteTemp = "/tmp/vps-dash.deploy.$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
Copy-ToRemote -Source $LocalBinary -Target $remoteTemp

$remoteScriptTemplate = @'
set -euo pipefail

SERVICE_NAME="__SERVICE_NAME__"
BINARY_PATH="__REMOTE_BINARY__"
DATA_DIR="/usr/local/bin/data"
BACKUP_PATH="${BINARY_PATH}.bak.$(date +%Y%m%d%H%M%S)"

install -d -m 755 /usr/local/bin
install -d -m 755 "$DATA_DIR"
install -d -m 755 /var/log/nginx

if [ -f "$BINARY_PATH" ]; then
    cp "$BINARY_PATH" "$BACKUP_PATH"
fi

install -m 755 "__REMOTE_TEMP__" "$BINARY_PATH"
rm -f "__REMOTE_TEMP__"

if [ ! -f "$DATA_DIR/security-settings.json" ]; then
    cat > "$DATA_DIR/security-settings.json" <<'JSON'
{
  "auto_ban_enabled": true,
  "ban_threshold": 1,
  "probe_patterns": [
    "/.env",
    ".env.",
    "/.git",
    "/.svn",
    "/.htaccess",
    "/wp-config.php",
    "/config.php",
    "/database.sql"
  ],
  "telegram_alerts": true
}
JSON
    chmod 600 "$DATA_DIR/security-settings.json"
fi

if command -v nginx > /dev/null 2>&1; then
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

cat > "/etc/systemd/system/$SERVICE_NAME.service" <<EOF
[Unit]
Description=Premium VPS Management Dashboard
After=network.target

[Service]
ExecStart=$BINARY_PATH
Restart=always
User=root
WorkingDirectory=/usr/local/bin

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null
systemctl restart "$SERVICE_NAME"
systemctl --no-pager --full status "$SERVICE_NAME"
'@

$remoteScript = $remoteScriptTemplate.
    Replace("__SERVICE_NAME__", $ServiceName).
    Replace("__REMOTE_BINARY__", $RemoteBinary).
    Replace("__REMOTE_TEMP__", $remoteTemp)

$encodedScript = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($remoteScript))
Invoke-Remote -Command "printf '%s' '$encodedScript' | base64 -d | bash"

Invoke-Remote -Command "ufw status numbered || true"
Invoke-Remote -Command "curl -fsS http://127.0.0.1:8900/ >/dev/null && echo 'Dashboard UI is responding locally'"
