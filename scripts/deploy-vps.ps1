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
    <title>Tên miền chưa cấu hình</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background-color: #0f172a;
            color: #f8fafc;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: 20px;
        }
        .card {
            background-color: #1e293b;
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 40px;
            max-width: 480px;
            width: 100%;
            text-align: center;
            box-shadow: 0 10px 25px rgba(0,0,0,0.3);
        }
        .icon {
            font-size: 48px;
            margin-bottom: 16px;
        }
        h1 {
            font-size: 20px;
            font-weight: 600;
            color: #38bdf8;
            margin-bottom: 12px;
        }
        p {
            color: #94a3b8;
            font-size: 14px;
            line-height: 1.6;
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">🌐</div>
        <h1>Tên Miền Chưa Cấu Hình</h1>
        <p>Tên miền này đã trỏ thành công về IP Server, nhưng chưa được thiết lập Virtual Host trên Nginx.</p>
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
