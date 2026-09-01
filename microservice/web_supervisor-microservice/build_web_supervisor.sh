#!/usr/bin/env bash

# 遇到错误立即退出
set -euo pipefail

# 切换到脚本所在目录
cd "$(dirname "$0")"

# 服务列表
services=(
    crawler-service
    email-service
    parser-service
    redis_cache-service
    web_supervisor-manager
)

# 默认目标平台和架构（全平台）
default_targets=(
    linux/arm64
    linux/amd64
    windows/amd64
)

# 如果脚本带参数，则使用传入的目标列表；否则使用默认全平台
if [ $# -gt 0 ]; then
    targets=("$@")
else
    targets=("${default_targets[@]}")
fi

# 创建根 bin 目录
mkdir -p bin

for svc in "${services[@]}"; do
    echo "=== Building $svc ==="
    cd "$svc"

    # 整理依赖
    go mod tidy

    # 创建该服务的输出子目录
    service_dir="../bin/$svc"
    mkdir -p "$service_dir"

    for target in "${targets[@]}"; do
        # 检查目标格式是否为 os/arch
        if [[ "$target" != *"/"* ]]; then
            echo "Error: Invalid target format '$target'. Expected format: os/arch (e.g., linux/amd64)"
            exit 1
        fi

        # 分割 platform/arch
        IFS='/' read -r GOOS GOARCH <<< "$target"
        export GOOS
        export GOARCH
        export CGO_ENABLED=0

        # 构建输出文件名
        outfile="$service_dir/${svc}_${GOOS}_${GOARCH}"
        if [ "$GOOS" = "windows" ]; then
            outfile="${outfile}.exe"
        fi

        echo "  [${GOOS}/${GOARCH}] Building..."
        go build -o "$outfile" .
    done

    cd ..
done

echo "All builds completed."