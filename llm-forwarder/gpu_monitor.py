#!/usr/bin/env python3
"""GPU 监控客户端 (GPU Monitor Client).

部署在每台 GPU 服务器上。由代理转发器通过 SSH 执行，也可在本地持续巡检。

判断逻辑：所有 GPU 的 utilization.gpu 与显存使用率均低于阈值 => 空闲。
空闲输出 "true"，繁忙输出 "false"；退出码 0=空闲, 1=繁忙, 2=出错。

用法:
  一次性检查（供 SSH 调用，转发器默认使用此模式）:
    python3 gpu_monitor.py [--util 30] [--mem 30]
  持续巡检（写状态 JSON 到 stdout）:
    python3 gpu_monitor.py --watch --interval 5

退出码:
  0  空闲 (stdout: true)
  1  繁忙 (stdout: false)
  2  出错 (stdout: false, stderr: 错误信息)
"""
import argparse
import json
import subprocess
import sys
import time

NVIDIA_SMI = "nvidia-smi"
# 查询: index, 利用率%, 已用显存, 总显存(MiB)
QUERY_ARGS = [
    "--query-gpu=index,utilization.gpu,memory.used,memory.total",
    "--format=csv,noheader,nounits",
]


def query_gpus():
    """返回 (gpus, error)。gpus 为 list[dict]，error 为错误字符串或 None。"""
    try:
        proc = subprocess.run(
            [NVIDIA_SMI, *QUERY_ARGS],
            capture_output=True, text=True, timeout=10,
        )
    except FileNotFoundError:
        return None, "nvidia-smi not found (no NVIDIA driver?)"
    except subprocess.TimeoutExpired:
        return None, "nvidia-smi timeout"
    if proc.returncode != 0:
        return None, (proc.stderr.strip() or f"nvidia-smi exit {proc.returncode}")

    gpus = []
    for line in proc.stdout.strip().splitlines():
        line = line.strip()
        if not line:
            continue
        parts = [p.strip() for p in line.split(",")]
        if len(parts) < 4:
            continue
        try:
            gpus.append({
                "index": int(parts[0]),
                "util": float(parts[1]),
                "mem_used": float(parts[2]),
                "mem_total": float(parts[3]),
            })
        except ValueError:
            continue
    return gpus, None


def is_idle(gpus, util_thresh, mem_thresh):
    """所有 GPU 都低于阈值才算空闲。"""
    if not gpus:
        return False  # 查不到 GPU => 无法跑本地模型 => 视为繁忙
    for g in gpus:
        mem_pct = (g["mem_used"] / g["mem_total"] * 100.0) if g["mem_total"] > 0 else 100.0
        if g["util"] >= util_thresh or mem_pct >= mem_thresh:
            return False
    return True


def check_once(util_thresh, mem_thresh):
    """一次性检查，向 stdout 输出 true/false。"""
    gpus, err = query_gpus()
    if err:
        print("false")
        print(f"[gpu_monitor] error: {err}", file=sys.stderr)
        return 2
    idle = is_idle(gpus, util_thresh, mem_thresh)
    print("true" if idle else "false")
    return 0 if idle else 1


def watch(util_thresh, mem_thresh, interval):
    """持续巡检，每 interval 秒输出一行状态 JSON。"""
    while True:
        gpus, err = query_gpus()
        ts = time.strftime("%Y-%m-%d %H:%M:%S")
        if err:
            print(json.dumps({"ts": ts, "idle": False, "error": err}), flush=True)
        else:
            idle = is_idle(gpus, util_thresh, mem_thresh)
            print(json.dumps({"ts": ts, "idle": idle, "gpus": gpus}), flush=True)
        time.sleep(interval)


def main():
    ap = argparse.ArgumentParser(description="GPU 监控客户端")
    ap.add_argument("--util", type=float, default=30.0,
                    help="GPU 利用率阈值%%，低于视为空闲 (默认 30)")
    ap.add_argument("--mem", type=float, default=30.0,
                    help="显存使用率阈值%%，低于视为空闲 (默认 30)")
    ap.add_argument("--watch", action="store_true", help="持续巡检模式")
    ap.add_argument("--interval", type=float, default=5.0,
                    help="巡检间隔秒 (默认 5，仅 --watch 生效)")
    args = ap.parse_args()

    if args.watch:
        try:
            watch(args.util, args.mem, args.interval)
        except KeyboardInterrupt:
            pass
    else:
        sys.exit(check_once(args.util, args.mem))


if __name__ == "__main__":
    main()
