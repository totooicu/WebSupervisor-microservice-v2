"""应用构建与入口: 注册路由, 启动 aiohttp 服务。"""
import argparse
import logging

from aiohttp import web

from config import Config
from forwarder import Forwarder


def build_app(cfg):
    """构建 aiohttp Application, 注册双协议路由。"""
    app = web.Application()
    fwd = Forwarder(cfg)
    app["fwd"] = fwd
    app.on_startup.append(fwd.start)
    app.on_cleanup.append(fwd.stop)
    app.router.add_get("/health", fwd.handle_health)
    app.router.add_post("/v1/chat/completions", fwd.handle_openai_chat)
    app.router.add_post("/v1/completions", fwd.handle_openai_completions)
    app.router.add_get("/v1/models", fwd.handle_openai_models)
    app.router.add_post("/api/chat", fwd.handle_ollama_chat)
    app.router.add_post("/api/generate", fwd.handle_ollama_generate)
    app.router.add_get("/api/tags", fwd.handle_ollama_tags)
    return app


def main():
    ap = argparse.ArgumentParser(description="大模型请求转发器")
    ap.add_argument("-c", "--config", default="config.yaml", help="配置文件路径 (默认 config.yaml)")
    ap.add_argument("--log-level", default="INFO", help="日志级别 (默认 INFO)")
    args = ap.parse_args()
    logging.basicConfig(
        level=args.log_level.upper(),
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    )
    cfg = Config(args.config)
    host, _, port = cfg.listen.partition(":")
    web.run_app(build_app(cfg), host=host or "0.0.0.0", port=int(port) if port else 8080, print=None)


if __name__ == "__main__":
    main()
