# Memos + Nginx HTTPS

This folder provides an Nginx reverse-proxy template and a self-signed certificate generator.

If you're on macOS (local machine) and want to use a real domain like `www.yingshun.xin`, you have two common setups:

1) Local-only HTTPS (recommended): use `mkcert` + `/etc/hosts` mapping.
2) Public HTTPS: use Let's Encrypt (requires the domain to resolve to your Mac's public IP and port 80/443 reachable).

If you only need **LAN access** (devices inside your home/office network), you can still use `www.yingshun.xin` as long as it resolves to your Mac's **LAN IP** (e.g. `192.168.1.x`).

## 1) Run Memos on localhost only

Run memos on a local interface and let Nginx handle TLS:

- Example:
  - `./memos --addr 127.0.0.1 --port 8081 --instance-url https://www.yingshun.xin`

You can also set env var:
- `MEMOS_INSTANCE_URL=https://memos.example.com`

LAN note:
- Ensure `www.yingshun.xin` resolves to your Mac's LAN IP from other devices.
- If you want `https://yingshun.xin` (apex) to work too, ensure `yingshun.xin` also resolves to the same LAN IP.
- On a client device, validate:
  - `nslookup www.yingshun.xin`
  - or `ping www.yingshun.xin`

## 2) Generate a certificate

### Option A: Self-signed (quick)

- macOS (Homebrew nginx, Intel): `sudo ./scripts/nginx/gen-self-signed-cert.sh memos.example.com /usr/local/etc/nginx/certs`
- macOS (Homebrew nginx, Apple Silicon): `sudo ./scripts/nginx/gen-self-signed-cert.sh memos.example.com /opt/homebrew/etc/nginx/certs`
- Linux (common): `sudo ./scripts/nginx/gen-self-signed-cert.sh memos.example.com /etc/nginx/certs`

Browsers will warn unless you trust the cert.

This is acceptable for quick LAN testing, but you'll likely prefer `mkcert` for a clean browser experience.

### Option B: Local-dev on macOS (recommended): mkcert

If you only need HTTPS locally (no public CA), mkcert is much nicer than a raw self-signed cert.

- `brew install mkcert nss`
- `mkcert -install`
- Intel: `sudo mkdir -p /usr/local/etc/nginx/certs`
- Apple Silicon: `sudo mkdir -p /opt/homebrew/etc/nginx/certs`

For `www.yingshun.xin` on your Mac (local-only), map it to localhost:

- Add to `/etc/hosts`:
  - `127.0.0.1 www.yingshun.xin`

Then generate a trusted cert for the domain:

- Intel:
  - `mkcert -key-file /usr/local/etc/nginx/certs/www.yingshun.xin.key -cert-file /usr/local/etc/nginx/certs/www.yingshun.xin.crt www.yingshun.xin`
- Apple Silicon:
  - `mkcert -key-file /opt/homebrew/etc/nginx/certs/www.yingshun.xin.key -cert-file /opt/homebrew/etc/nginx/certs/www.yingshun.xin.crt www.yingshun.xin`

If you're doing LAN access and also want to include the LAN IP in the cert (optional):
- `mkcert -key-file /usr/local/etc/nginx/certs/www.yingshun.xin.key -cert-file /usr/local/etc/nginx/certs/www.yingshun.xin.crt yingshun.xin www.yingshun.xin 192.168.5.8`

To make other LAN devices trust the cert, install the mkcert local CA on those devices:
- Find the CA location on your Mac:
  - `mkcert -CAROOT`
- The root CA is typically `rootCA.pem` under that directory.
- Import/trust it on the device (OS-specific):
  - iOS/iPadOS: AirDrop/email the `rootCA.pem`, install profile, then enable full trust in Settings.
  - Android: install as a CA certificate (note: Android has tighter rules for user CAs in some apps).
  - Windows: import into "Trusted Root Certification Authorities".

Then set `server_name localhost;` and point Nginx to these files.

### Option C: Public domain (recommended): Let's Encrypt

Use certbot on your server (Linux):
- Install certbot for your distro
- Obtain certs:
  - `certbot certonly --nginx -d memos.example.com`

Then update Nginx `ssl_certificate` and `ssl_certificate_key` to the paths under `/etc/letsencrypt/live/...`.

If you're doing Let's Encrypt directly on macOS:
- HTTP-01 requires inbound `:80` reachable from the internet.
- If your Mac is behind NAT, you must configure router port-forwarding.
- If you can't expose `:80`, use DNS-01 (e.g. `acme.sh` with your DNS provider).

## 3) Configure Nginx

macOS (Homebrew nginx):
- Put the vhost config under `.../etc/nginx/servers/` and ensure it's included from `nginx.conf`.

Linux (common):
- Put the vhost config under `/etc/nginx/conf.d/` (or distro equivalent).

For `www.yingshun.xin` there's also a ready-to-edit template:
- `./scripts/nginx/memos.www.yingshun.xin.conf.example`

Validate and reload:
- `sudo nginx -t`
- `sudo systemctl reload nginx` (Linux)

macOS (Homebrew nginx) tips:
- Default config root:
  - Apple Silicon: `/opt/homebrew/etc/nginx/`
  - Intel: `/usr/local/etc/nginx/`
- Common include dir:
  - Apple Silicon: `/opt/homebrew/etc/nginx/servers/`
  - Intel: `/usr/local/etc/nginx/servers/`
- Start/reload:
  - `sudo nginx -t`
  - `sudo nginx -s reload` (or `sudo nginx` to start)

Note: binding to `:80`/`:443` usually requires root privileges.

macOS firewall:
- If clients can't reach your Mac, allow inbound connections for `nginx` (System Settings → Network → Firewall).

## 4) Common gotchas

- If you terminate TLS at Nginx, you usually want:
  - `--instance-url https://<your-domain>`
- Ensure headers are forwarded:
  - `X-Forwarded-Proto`, `X-Forwarded-For`, `Host`
- Increase upload limit if you attach big files:
  - `client_max_body_size`

## 5) iOS 长音频播放（>90s）稳定性建议

如果 iOS 客户端播放长音频（常见约 90s 以上）出现 `NSURLErrorDomain Code=-1005`，优先检查反向代理链路超时和 buffering。

推荐仅对下载路由单独配置（见示例 vhost 已包含）：

- `location ^~ /file/attachments/`
- `proxy_connect_timeout 300s`
- `proxy_read_timeout 300s`
- `proxy_send_timeout 300s`
- `send_timeout 300s`
- `keepalive_timeout 300s`
- `proxy_buffering off`
- `proxy_request_buffering off`
- `proxy_max_temp_file_size 0`
- 保留 `Range` / `If-Range` 头

验证 nginx 配置并重载：

```bash
sudo nginx -t
sudo nginx -s reload
```

验证 Range 与长时下载：

```bash
# 1) 检查响应头
curl -I "https://<domain>/file/attachments/<uid>/memo-audio.m4a"

# 2) 检查 206 Partial Content
curl -H "Range: bytes=0-999" -v "https://<domain>/file/attachments/<uid>/memo-audio.m4a" -o /dev/null

# 3) 人为限速，拉长下载时长到 >90s
curl --limit-rate 16k -w "status:%{http_code} time_total:%{time_total}s size:%{size_download}\n" \
  -o /dev/null "https://<domain>/file/attachments/<uid>/memo-audio.m4a"
```

日志排查（修复前后对比）：

```bash
# 最近附件下载请求
grep "/file/attachments/" /var/log/nginx/access.log | tail -n 200

# 重点错误关键词
grep -E "upstream timed out|connection reset by peer|client timed out| 499 | 504 " /var/log/nginx/error.log
```

如果前置了 Cloudflare/CDN/LB，请确认是否存在固定请求时长限制（常见 100s）；若存在，需要在 CDN/LB 层调整策略或绕过该层下载附件。
