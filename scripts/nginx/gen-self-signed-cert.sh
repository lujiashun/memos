#!/usr/bin/env bash
set -euo pipefail

# Generate a self-signed TLS certificate (with SAN) for Nginx.
#
# Usage:
#   ./gen-self-signed-cert.sh memos.example.com /etc/nginx/certs
#   ./gen-self-signed-cert.sh localhost ./certs
#
# Output:
#   <outDir>/<domain>.crt
#   <outDir>/<domain>.key

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <domain> <outDir>" >&2
  exit 2
fi

domain="$1"
out_dir="$2"

mkdir -p "$out_dir"

crt="$out_dir/$domain.crt"
key="$out_dir/$domain.key"

# Build SAN list.
# - Always include DNS:domain
# - If domain == localhost, also include common localhost IPs
san="DNS:${domain}"
if [[ "$domain" == "localhost" ]]; then
  san="${san},DNS:localhost,IP:127.0.0.1,IP:::1"
fi

# macOS OpenSSL supports -addext on modern versions; keep it simple.
openssl req -x509 -newkey rsa:2048 -sha256 -nodes \
  -days 825 \
  -subj "/CN=${domain}" \
  -addext "subjectAltName=${san}" \
  -keyout "$key" \
  -out "$crt"

chmod 600 "$key"

echo "Generated: $crt"
echo "Generated: $key"
