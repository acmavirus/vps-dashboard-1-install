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

if [ ! -f "$DATA_DIR/security-logs.json" ]; then
    printf '[]\n' > "$DATA_DIR/security-logs.json"
    chmod 600 "$DATA_DIR/security-logs.json"
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
