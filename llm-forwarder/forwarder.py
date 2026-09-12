"""大模型请求转发器: 生命周期 + 鉴权 + 端点处理。

端点处理均为薄层: 解析请求 -> router 决策目标 -> 委托 proxy 原语转发。
OpenAI(/v1/*) 与 ollama(/api/*) 同协议转发直接复用 proxy; 繁忙时 ollama->openai 走转换。
"""
import asyncio
import logging
import time

from aiohttp import ClientSession, ClientTimeout, web

import converters as cv
import proxy
import rewriters as rw
import router
from gpu_state import GpuState
from util import json_err, now_iso, up_headers

LOG = logging.getLogger("forwarder")


class Forwarder:
    def __init__(self, cfg):
        self.cfg = cfg
        self.gpu = GpuState(cfg)
        self.session = None

    async def start(self, app):
        self.session = ClientSession(timeout=ClientTimeout(total=self.cfg.request_timeout))
        app["bg"] = asyncio.create_task(self.gpu.background_loop())
        LOG.info("forwarder ready; listen=%s servers=%s", self.cfg.listen, list(self.cfg.servers))

    async def stop(self, app):
        bg = app.get("bg")
        if bg:
            bg.cancel()
            try:
                await bg
            except (asyncio.CancelledError, Exception):
                pass
        if self.session:
            await self.session.close()

    def _auth(self, request):
        if not self.cfg.api_keys:
            return True
        auth = request.headers.get("Authorization", "")
        return auth.startswith("Bearer ") and auth[7:].strip() in self.cfg.api_keys

    async def _read_body(self, request):
        """读取+鉴权+校验 JSON, 返回 (body, error_response)。"""
        if not self._auth(request):
            return None, json_err(401, "invalid api key")
        try:
            return await request.json(), None
        except Exception:
            return None, json_err(400, "invalid json body")

    # ---- OpenAI 端点 (/v1/*) ----
    async def handle_openai_chat(self, request):
        return await self._openai(request, "/chat/completions")

    async def handle_openai_completions(self, request):
        return await self._openai(request, "/completions")

    async def _openai(self, request, path_suffix):
        body, err = await self._read_body(request)
        if err:
            return err
        req_model = body.get("model")
        target, err = await router.route(self.cfg, self.gpu, req_model, "openai")
        if not target:
            return json_err(502, err)
        body["model"] = target["model"]
        url = target["base"] + path_suffix
        headers = up_headers(request, target, self.cfg.api_keys)
        stream = bool(body.get("stream"))
        LOG.info("openai %s model=%s -> %s (%s)", path_suffix, req_model, target["via"],
                 "stream" if stream else "nostream")
        if stream:
            return await proxy.stream_passthrough(
                self.session, request, url, headers, body, "\n\n",
                lambda u: rw.rewrite_sse_model(u, req_model), "text/event-stream")
        return await proxy.proxy_json(self.session, request, url, headers, body, req_model)

    async def handle_openai_models(self, request):
        if not self._auth(request):
            return json_err(401, "invalid api key")
        data = [{"id": m, "object": "model", "owned_by": "forwarder"}
                for m in self.cfg.models if m != "default"]
        return web.json_response({"object": "list", "data": data})

    # ---- Ollama 端点 (/api/*) ----
    async def handle_ollama_chat(self, request):
        return await self._ollama(request, "/api/chat", "chat")

    async def handle_ollama_generate(self, request):
        return await self._ollama(request, "/api/generate", "generate")

    async def _ollama(self, request, ollama_path, kind):
        body, err = await self._read_body(request)
        if err:
            return err
        req_model = body.get("model")
        target, err = await router.route(self.cfg, self.gpu, req_model, "ollama")
        if not target:
            return json_err(502, err)
        stream = bool(body.get("stream"))
        headers = up_headers(request, target, self.cfg.api_keys)
        if target["proto"] == "ollama":  # 同协议直转
            body["model"] = target["model"]
            url = target["base"] + ollama_path
            LOG.info("ollama %s model=%s -> %s (%s)", ollama_path, req_model, target["via"],
                     "stream" if stream else "nostream")
            if stream:
                return await proxy.stream_passthrough(
                    self.session, request, url, headers, body, "\n",
                    lambda u: rw.rewrite_ndjson_model(u, req_model), "application/x-ndjson")
            return await proxy.proxy_json(self.session, request, url, headers, body, req_model)
        LOG.info("ollama %s model=%s -> %s (convert)", ollama_path, req_model, target["via"])
        return await self._convert(request, body, target, stream, kind, req_model)

    async def _convert(self, request, body, target, stream, kind, req_model):
        """ollama 客户端 -> OpenAI 上游的跨协议转换 (chat/generate 共用)。"""
        if kind == "chat":
            oai = cv.ollama_chat_to_openai(body, target["model"])
            url = target["base"] + "/chat/completions"
            shaper = cv.shape_chat_response
        else:
            oai = cv.ollama_generate_to_openai(body, target["model"])
            url = target["base"] + "/completions"
            shaper = cv.shape_generate_response
        headers = up_headers(request, target, self.cfg.api_keys)
        if stream:
            return await proxy.stream_openai_to_ollama(
                self.session, request, url, headers, oai, req_model,
                lambda u, m: cv.sse_to_ollama_lines(u, m, kind), cv.final_done_line(req_model, kind))
        return await proxy.nonstream_openai_to_ollama(self.session, url, headers, oai, req_model, shaper)

    async def handle_ollama_tags(self, request):
        if not self._auth(request):
            return json_err(401, "invalid api key")
        models = [{"name": m, "model": m, "modified_at": now_iso(),
                   "size": 0, "digest": "", "details": {}}
                  for m in self.cfg.models if m != "default"]
        return web.json_response({"models": models})

    async def handle_health(self, request):
        states = {}
        for s in self.cfg.servers:
            st = self.gpu._st.get(s, {})
            states[s] = {"idle": st.get("idle"), "err": st.get("err"),
                         "age_s": round(time.time() - st.get("ts", 0), 1) if st.get("ts") else None}
        return web.json_response({"ok": True, "servers": states, "models": list(self.cfg.models)})
