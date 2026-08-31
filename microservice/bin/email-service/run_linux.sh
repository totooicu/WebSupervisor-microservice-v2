#!/bin/bash
# Run email-service. Auto-detects amd64/arm64 binary.
cd "$(dirname "$0")" || exit 1
arch=$(uname -m)
case "$arch" in
    x86_64|amd64)   bin="email-service_linux_amd64" ;;
    aarch64|arm64)  bin="email-service_linux_arm64" ;;
    *) echo "Unsupported architecture: $arch"; exit 1 ;;
esac
chmod +x "$bin" 2>/dev/null
echo "[email-service] starting ($bin)..."
exec ./"$bin" -config=./config.json
