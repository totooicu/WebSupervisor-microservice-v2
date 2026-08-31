#!/bin/bash
# Run web_supervisor-manager. Auto-detects amd64/arm64 binary.
# Reads config.json (jobs_path=./jobs.json) by default.
cd "$(dirname "$0")" || exit 1
arch=$(uname -m)
case "$arch" in
    x86_64|amd64)   bin="web_supervisor-manager_linux_amd64" ;;
    aarch64|arm64)  bin="web_supervisor-manager_linux_arm64" ;;
    *) echo "Unsupported architecture: $arch"; exit 1 ;;
esac
chmod +x "$bin" 2>/dev/null
echo "[web_supervisor-manager] starting ($bin) | config: config.json | jobs: jobs.json"
exec ./"$bin" -config=./config.json
