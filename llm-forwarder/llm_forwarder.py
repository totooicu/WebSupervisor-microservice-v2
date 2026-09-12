#!/usr/bin/env python3
"""大模型请求转发器入口 (薄封装, 重导出子模块公共 API)。

子模块分工 (高内聚低耦合, 每个文件 <200 行):
  config.py     - 配置加载与模型路由解析
  gpu_state.py  - GPU 状态缓存 + SSH 检查 + 后台巡检
  util.py       - 通用工具 (时间戳/错误响应/字节流切分/请求头)
  rewriters.py  - 流式响应 model 字段重写 (SSE/NDJSON)
  converters.py - 跨协议转换 (Ollama <-> OpenAI)
  proxy.py      - 可复用上游代理原语 (非流式/流式透传/流式转换)
  router.py     - 路由决策 (模型名 + GPU 状态 -> 目标上游)
  forwarder.py  - 转发器: 生命周期 + 鉴权 + 端点处理
  app.py        - 应用构建与入口
  gpu_monitor.py- GPU 监督客户端 (部署在 GPU 服务器, SSH 调用)
"""
from app import build_app, main  # noqa: F401
from config import Config  # noqa: F401
from converters import (  # noqa: F401
    final_done_line, ollama_chat_to_openai, ollama_generate_to_openai,
    sse_to_ollama_lines, shape_chat_response, shape_generate_response)
from forwarder import Forwarder  # noqa: F401
from rewriters import rewrite_ndjson_model, rewrite_sse_model  # noqa: F401
from util import LineSplitter, json_err, now_iso, up_headers  # noqa: F401

if __name__ == "__main__":
    main()
