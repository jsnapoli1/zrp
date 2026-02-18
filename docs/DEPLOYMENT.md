# ZRP Deployment Guide

## Building

```bash
cd zrp
go build -o zrp .
```

The binary is fully self-contained. Copy it plus the `static/` directory to your target machine.

## Quick Deploy

```bash
./zrp -pmDir /path/to/parts/database -port 9000 -db /var/data/zrp.db
```

## systemd Service (Linux)

Create `/etc/systemd/system/zrp.service`:

```ini
[Unit]
Description=ZRP - Resource Planning
After=network.target

[Service]
Type=simple
User=zrp
Group=zrp
WorkingDirectory=/opt/zrp
ExecStart=/opt/zrp/zrp -pmDir /opt/zrp/parts -port 9000 -db /var/data/zrp/zrp.db
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now zrp
sudo systemctl status zrp
```

## launchd Service (macOS)

Create `~/Library/LaunchAgents/com.zrp.server.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.zrp.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/zrp</string>
        <string>-pmDir</string>
        <string>/Users/you/parts/database</string>
        <string>-port</string>
        <string>9000</string>
        <string>-db</string>
        <string>/Users/you/.zrp/zrp.db</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/zrp.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/zrp.err</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.zrp.server.plist
```

## Reverse Proxy

ZRP binds to all interfaces by default. In production, bind to localhost and put a reverse proxy in front.

### Caddy

```
zrp.example.com {
    reverse_proxy localhost:9000
}
```

### nginx

```nginx
server {
    listen 443 ssl;
    server_name zrp.example.com;

    ssl_certificate /etc/ssl/certs/zrp.crt;
    ssl_certificate_key /etc/ssl/private/zrp.key;

    location / {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Database Backup

The SQLite database is a single file. Back it up with:

```bash
# While ZRP is running (safe — WAL mode handles concurrent access)
sqlite3 /var/data/zrp/zrp.db ".backup /backups/zrp-$(date +%Y%m%d).db"
```

Or simply copy the file when the server is stopped:

```bash
cp /var/data/zrp/zrp.db /backups/zrp-$(date +%Y%m%d).db
```

Automate with cron:

```bash
0 2 * * * sqlite3 /var/data/zrp/zrp.db ".backup /backups/zrp-$(date +\%Y\%m\%d).db"
```

## Security Considerations

- **No authentication built in.** ZRP has no login system or API keys. Use a reverse proxy with HTTP basic auth, OAuth proxy, or VPN to restrict access.
- **CORS is permissive.** The server sets `Access-Control-Allow-Origin: *`. Restrict this in your reverse proxy if needed.
- **Bind to localhost** in production and expose through a reverse proxy with TLS.
- **File permissions:** Ensure the SQLite database file is only readable by the ZRP process user.
- **Parts directory:** The `-pmDir` path grants read access to all CSV files in that directory tree.

## Upgrading

1. Stop the server
2. Back up the database
3. Replace the binary and `static/` directory
4. Start the server

Migrations run automatically on startup — new tables are created with `CREATE TABLE IF NOT EXISTS`. Existing data is preserved.
