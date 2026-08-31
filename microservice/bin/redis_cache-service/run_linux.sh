#!/bin/bash
# Run redis_cache-service. Auto-detects amd64/arm64 binary.
cd "$(dirname "$0")" || exit 1
arch=$(uname -m)
case "$arch" in
    x86_64|amd64)   bin="redis_cache-service_linux_amd64" ;;
    aarch64|arm64)  bin="redis_cache-service_linux_arm64" ;;
    *) echo "Unsupported architecture: $arch"; exit 1 ;;
esac
chmod +x "$bin" 2>/dev/null
echo "[redis_cache-service] starting ($bin)..."
exec ./"$bin" -config=./config.json
