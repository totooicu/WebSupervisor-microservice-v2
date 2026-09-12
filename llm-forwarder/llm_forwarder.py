#!/usr/bin/env python3
"""大模型请求转发器 (LLM Forwarder).

对外同时提供 OpenAI (/v1/*) 与 Ollama (/api/*) 兼容接口。
每个请求按模型名路由到指定 GPU 服务器，SSH 检查该机 GPU 状态：
  空闲 -> 转发到该机本地 ollama
  繁忙 -> 转发到配置的网络上游 (任意 OpenAI/Ollama 兼容端点)

特性:
  - 双协议: OpenAI /v1/chat/completions、/v1/completions、/v1/models
            Ollama /api/chat、/api/generate、/api/tags
  - GPU 状态缓存 (TTL=check_interval) + 后台巡检，避免每请求都 SSH
  - 流式透传 (SSE / NDJSON)，并重写响应中的 model 字段为客户端请求的模型名
  - 跨协议转换: ollama 客户端繁忙时若仅配了 OpenAI 上游，自动 ollama<->openai 转换
  - SSH 支持密钥 / 密码两种认证
  - 健康检查 /health

依赖: aiohttp, asyncssh, pyyaml
配置: 见 config.yaml
"""
import argparse
import asyncio
import json
import logging
import os
import time
from datetime import datetime, timezone

import asyncssh
import yaml
from aiohttp import ClientSession, ClientTimeout, web

LOG = logging.getLogger("forwarder")

# ==================== 配置 ====================
class Config:
    def __init__(self, path):
        with open(path, "r", encoding="utf-8") as f:
            raw = yaml.safe_load(f) or {}
        self.raw = raw
        srv = raw.get("server", {}) or {}
        self.listen = srv.get("listen", "0.0.0.0:8080")
        self.api_keys = set(srv.get("api_keys") or [])
        g = raw.get("gpu", {}) or {}
        self.check_interval = float(g.get("check_interval", 10))
        self.ssh_timeout = float(g.get("ssh_timeout", 8))
        self.on_ssh_fail = g.get("on_ssh_fail", "busy")  # busy | idle | error
        self.request_timeout = float(raw.get("request_timeout", 600))
        self.servers = raw.get("servers", {}) or {}
        self.network = raw.get("network_upstream", {}) or {}
        self.models = raw.get("models", {}) or {}

    def resolve_model(self, name):
        """返回 (server_key, local_model, network_model) 或 None。"""
        m = self.models.get(name) or self.models.get("default")
        if not m:
            return None
        return m.get("server"), m.get("local_model", name), m.get("network_model", name)


# ==================== GPU 状态缓存 + SSH 检查 ====================
class GpuState:
    """每台服务器的 GPU 空闲状态缓存，按 check_interval 失效，带后台巡检。"""

    def __init__(self, cfg):
        self.cfg = cfg
        self._st = {s: {"idle": None, "ts": 0.0, "err": None} for s in cfg.servers}
        self._lk = {s: asyncio.Lock() for s in cfg.servers}

    async def get(self, server):
        st = self._st.get(server)
        if st is None:
            return {"idle": None, "ts": 0.0, "err": "unknown server"}
        if st["idle"] is not None and time.time() - st["ts"] < self.cfg.check_interval:
            return st
        # 缓存过期，加锁刷新 (避免惊群)
        async with self._lk[server]:
            st = self._st[server]
            if st["idle"] is not None and time.time() - st["ts"] < self.cfg.check_interval:
                return st
            await self._refresh(server)
            return self._st[server]

    async def _connect(self, ssh):
        host = ssh["host"]
        port = int(ssh.get("port", 22))
        user = ssh.get("user")
        kh = ssh.get("known_hosts", None)  # None => 不校验主机密钥
        base = dict(host=host, port=port, username=user, known_hosts=kh)
        if ssh.get("password"):
            return await asyncssh.connect(**base, password=ssh["password"])
        kf = ssh.get("key_file")
        if kf:
            kw = dict(client_keys=[os.path.expanduser(kf)])
            if ssh.get("passphrase"):
                kw["passphrase"] = ssh["passphrase"]
            return await asyncssh.connect(**base, **kw)
        return await asyncssh.connect(**base)

    async def _refresh(self, server):
        scfg = self.cfg.servers[server]
        ssh = scfg.get("ssh", {}) or {}
        cmd = scfg.get("check_command")
        keyword = str(scfg.get("idle_keyword", "true")).strip().lower()
        idle, err = None, None
        conn = None
        try:
            conn = await asyncio.wait_for(self._connect(ssh), timeout=self.cfg.ssh_timeout)
            res = await asyncio.wait_for(conn.run(cmd, check=False), timeout=self.cfg.ssh_timeout)
            out = (res.stdout or "").strip().lower()
            idle = (keyword in out) if out else False
            if not out:
                err = "empty stdout"
            if res.stderr:
                LOG.debug("[%s] check stderr: %s", server, res.stderr.strip()[:200])
        except Exception as e:
            err = f"{type(e).__name__}: {e}"
            pol = self.cfg.on_ssh_fail
            idle = False if pol == "busy" else (True if pol == "idle" else None)
            LOG.warning("[%s] GPU check failed (%s) -> policy=%s -> idle=%s", server, err, pol, idle)
        finally:
            if conn:
                try:
                    conn.close()
                except Exception:
                    pass
                try:
                    await asyncio.wait_for(conn.wait_closed(), timeout=2)
                except Exception:
                    pass
        self._st[server] = {"idle": idle, "ts": time.time(), "err": err}

    async def background_loop(self):
        """后台巡检，保持缓存新鲜，降低请求时延。"""
        while True:
            for s in list(self.cfg.servers):
                try:
                    async with self._lk[s]:
                        st = self._st[s]
                        if st["idle"] is None or time.time() - st["ts"] >= self.cfg.check_interval:
                            await self._refresh(s)
                except Exception as e:
                    LOG.warning("background refresh %s failed: %s", s, e)
            await asyncio.sleep(max(1.0, self.cfg.check_interval))


# ==================== 工具函数 ====================
def _now_iso():
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


class LineSplitter:
    """按分隔符切分字节流，保留未结束的尾巴缓冲，逐单元产出。"""
    __slots__ = ("buf", "sep")

    def __init__(self, sep):
        self.buf = bytearray()
        self.sep = sep.encode() if isinstance(sep, str) else sep

    def feed(self, data):
        self.buf.extend(data)
        out = []
        while True:
            i = self.buf.find(self.sep)
            if i < 0:
                break
            out.append(bytes(self.buf[:i]))
            del self.buf[:i + len(self.sep)]
        return out

    def flush(self):
        r = bytes(self.buf)
        self.buf.clear()
        return r


def _json_err(status, msg):
    return web.json_response(
        {"error": {"message": msg, "type": "forwarder_error", "code": status}},
        status=status,
    )


def _rewrite_sse_unit(unit, model):
    """重写一个 SSE 事件块 (bytes, 不含尾随 \\n\\n) 中 data: JSON 的 model 字段。"""
    lines = unit.split(b"\n")
    out = []
    for line in lines:
        if line.startswith(b"data:"):
            payload = line[5:].strip()
            if payload == b"[DONE]":
                out.append(b"data: [DONE]")
                continue
            try:
                obj = json.loads(payload)
                if isinstance(obj, dict) and "model" in obj:
                    obj["model"] = model
                out.append(b"data: " + json.dumps(obj, ensure_ascii=False).encode())
                continue
            except Exception:
                pass
        out.append(line)
    return b"\n".join(out)


def _rewrite_ndjson_line(line, model):
    """重写一行 NDJSON 中的 model 字段。"""
    if not line.strip():
        return line
    try:
        obj = json.loads(line)
        if isinstance(obj, dict) and "model" in obj:
            obj["model"] = model
        return json.dumps(obj, ensure_ascii=False).encode()
    except Exception:
        return line


# ---- 跨协议转换: ollama 客户端 <-> OpenAI 上游 ----
def _ollama_chat_to_openai(body, model):
    out = {"model": model, "messages": body.get("messages", []),
           "stream": bool(body.get("stream", False))}
    opt = body.get("options") or {}
    if "temperature" in opt:
        out["temperature"] = opt["temperature"]
    if "top_p" in opt:
        out["top_p"] = opt["top_p"]
    if "num_predict" in opt:
        out["max_tokens"] = opt["num_predict"]
    if "seed" in opt:
        out["seed"] = opt["seed"]
    if body.get("format") == "json":
        out["response_format"] = {"type": "json_object"}
    return out


def _ollama_generate_to_openai(body, model):
    out = {"model": model, "prompt": body.get("prompt", ""),
           "stream": bool(body.get("stream", False))}
    opt = body.get("options") or {}
    if "temperature" in opt:
        out["temperature"] = opt["temperature"]
    if "top_p" in opt:
        out["top_p"] = opt["top_p"]
    if "num_predict" in opt:
        out["max_tokens"] = opt["num_predict"]
    if "seed" in opt:
        out["seed"] = opt["seed"]
    return out


def _sse_unit_to_ollama_chat_lines(unit, model):
    """把一个 OpenAI SSE 事件块转成若干 ollama /api/chat NDJSON 行 (bytes)。返回 (lines, saw_done)。"""
    lines, saw_done = [], False
    for line in unit.split(b"\n"):
        if not line.startswith(b"data:"):
            continue
        payload = line[5:].strip()
        if payload == b"[DONE]":
            saw_done = True
            continue
        try:
            obj = json.loads(payload)
        except Exception:
            continue
        choices = obj.get("choices") or []
        ch = choices[0] if choices else {}
        finish = ch.get("finish_reason")
        if finish:
            saw_done = True
            lines.append(json.dumps({
                "model": model, "created_at": _now_iso(),
                "message": {"role": "assistant", "content": ""},
                "done": True, "done_reason": finish,
            }, ensure_ascii=False).encode())
            continue
        delta = ch.get("delta", {}) or {}
        content = delta.get("content", "")
        if content or delta.get("role"):
            lines.append(json.dumps({
                "model": model, "created_at": _now_iso(),
                "message": {"role": delta.get("role", "assistant"), "content": content},
                "done": False,
            }, ensure_ascii=False).encode())
    return lines, saw_done


def _sse_unit_to_ollama_generate_lines(unit, model):
    """把一个 OpenAI SSE 事件块 (/v1/completions) 转成若干 ollama /api/generate NDJSON 行。"""
    lines, saw_done = [], False
    for line in unit.split(b"\n"):
        if not line.startswith(b"data:"):
            continue
        payload = line[5:].strip()
        if payload == b"[DONE]":
            saw_done = True
            continue
        try:
            obj = json.loads(payload)
        except Exception:
            continue
        choices = obj.get("choices") or []
        ch = choices[0] if choices else {}
        finish = ch.get("finish_reason")
        if finish:
            saw_done = True
            lines.append(json.dumps({
                "model": model, "created_at": _now_iso(),
                "response": "", "done": True, "done_reason": finish,
            }, ensure_ascii=False).encode())
            continue
        text = ch.get("text", "")
        if text:
            lines.append(json.dumps({
                "model": model, "created_at": _now_iso(),
                "response": text, "done": False,
            }, ensure_ascii=False).encode())
    return lines, saw_done


# ==================== 转发器 ====================
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

    # ---- 鉴权 ----
    def _check_auth(self, request):
        if not self.cfg.api_keys:
            return True
        auth = request.headers.get("Authorization", "")
        if auth.startswith("Bearer "):
            return auth[7:].strip() in self.cfg.api_keys
        return False

    def _up_headers(self, request, target):
        h = {"Content-Type": "application/json", "Accept": "*/*"}
        if target.get("auth"):
            h["Authorization"] = target["auth"]
        elif self.cfg.api_keys:
            a = request.headers.get("Authorization")
            if a:
                h["Authorization"] = a
        return h

    # ---- 路由决策 ----
    async def route(self, req_model, incoming_proto):
        """返回 (target_dict, error)。target: {base, proto, model, auth, via, idle}。"""
        res = self.cfg.resolve_model(req_model)
        if not res:
            return None, f"model '{req_model}' not configured and no 'default' mapping"
        server_key, local_model, network_model = res
        if server_key not in self.cfg.servers:
            return None, f"model maps to unknown server '{server_key}'"
        state = await self.gpu.get(server_key)
        idle = state["idle"]
        scfg = self.cfg.servers[server_key]
        local_ep = (scfg.get("local_endpoint") or "").rstrip("/")
        if idle is True:
            base = (local_ep + "/v1") if incoming_proto == "openai" else local_ep
            proto = "openai" if incoming_proto == "openai" else "ollama"
            return {"base": base, "proto": proto, "model": local_model,
                    "auth": None, "via": f"local:{server_key}", "idle": True}, None
        if idle is False:
            return self._route_network(incoming_proto, network_model)
        return None, f"GPU state unavailable for '{server_key}' ({state.get('err')})"

    def _route_network(self, incoming_proto, network_model):
        net = self.cfg.network
        if incoming_proto == "openai":
            up = net.get("openai", {}) or {}
            base = (up.get("base_url") or "").rstrip("/")
            if not base:
                return None, "no network openai upstream configured"
            return {"base": base, "proto": "openai", "model": network_model,
                    "auth": ("Bearer " + up["api_key"]) if up.get("api_key") else None,
                    "via": "network:openai", "idle": False}, None
        # ollama 客户端繁忙
        up = net.get("ollama", {}) or {}
        if up.get("base_url"):
            return {"base": up["base_url"].rstrip("/"), "proto": "ollama", "model": network_model,
                    "auth": ("Bearer " + up["api_key"]) if up.get("api_key") else None,
                    "via": "network:ollama", "idle": False}, None
        # 仅有 OpenAI 上游 -> 跨协议转换
        up2 = net.get("openai", {}) or {}
        base = (up2.get("base_url") or "").rstrip("/")
        if base:
            return {"base": base, "proto": "openai", "model": network_model,
                    "auth": ("Bearer " + up2["api_key"]) if up2.get("api_key") else None,
                    "via": "network:openai(convert)", "idle": False}, None
        return None, "no network upstream configured"

    # ---- OpenAI 兼容 ----
    async def handle_openai_chat(self, request):
        return await self._handle_openai(request, "/chat/completions")

    async def handle_openai_completions(self, request):
        return await self._handle_openai(request, "/completions")

    async def _handle_openai(self, request, path_suffix):
        if not self._check_auth(request):
            return _json_err(401, "invalid api key")
        try:
            body = await request.json()
        except Exception:
            return _json_err(400, "invalid json body")
        req_model = body.get("model")
        target, err = await self.route(req_model, "openai")
        if not target:
            return _json_err(502, err)
        body["model"] = target["model"]
        url = target["base"] + path_suffix
        headers = self._up_headers(request, target)
        stream = bool(body.get("stream"))
        LOG.info("openai %s model=%s -> %s (%s)", path_suffix, req_model, target["via"],
                 "stream" if stream else "nostream")
        if stream:
            return await self._stream_openai(request, url, headers, body, req_model)
        return await self._proxy_openai_nonstream(request, url, headers, body, req_model)

    async def handle_openai_models(self, request):
        if not self._check_auth(request):
            return _json_err(401, "invalid api key")
        data = [{"id": m, "object": "model", "owned_by": "forwarder"}
                for m in self.cfg.models if m != "default"]
        return web.json_response({"object": "list", "data": data})

    async def _stream_openai(self, request, url, headers, body, req_model):
        data = json.dumps(body, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                if up.status != 200:
                    txt = await up.text()
                    return _json_err(up.status, f"upstream error: {txt[:500]}")
                resp = web.StreamResponse(status=200, headers={
                    "Content-Type": "text/event-stream",
                    "Cache-Control": "no-cache",
                    "Connection": "keep-alive",
                })
                await resp.prepare(request)
                sp = LineSplitter("\n\n")
                async for chunk in up.content.iter_any():
                    for unit in sp.feed(chunk):
                        await resp.write(_rewrite_sse_unit(unit, req_model) + b"\n\n")
                rest = sp.flush()
                if rest:
                    await resp.write(_rewrite_sse_unit(rest, req_model) + b"\n\n")
                await resp.write_eof()
                return resp
        except Exception as e:
            LOG.warning("stream openai failed: %s", e)
            return _json_err(502, f"upstream stream error: {e}")

    async def _proxy_openai_nonstream(self, request, url, headers, body, req_model):
        data = json.dumps(body, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                raw = await up.read()
                if up.status != 200:
                    return web.Response(status=up.status, body=raw, content_type="application/json")
                try:
                    obj = json.loads(raw)
                    if isinstance(obj, dict) and "model" in obj:
                        obj["model"] = req_model
                    raw = json.dumps(obj, ensure_ascii=False).encode()
                except Exception:
                    pass
                return web.Response(status=up.status, body=raw, content_type="application/json")
        except Exception as e:
            return _json_err(502, f"upstream error: {e}")

    # ---- Ollama 兼容 ----
    async def handle_ollama_chat(self, request):
        if not self._check_auth(request):
            return _json_err(401, "invalid api key")
        try:
            body = await request.json()
        except Exception:
            return _json_err(400, "invalid json body")
        req_model = body.get("model")
        target, err = await self.route(req_model, "ollama")
        if not target:
            return _json_err(502, err)
        stream = bool(body.get("stream"))
        headers = self._up_headers(request, target)
        if target["proto"] == "ollama":
            body["model"] = target["model"]
            url = target["base"] + "/api/chat"
            LOG.info("ollama /api/chat model=%s -> %s (%s)", req_model, target["via"],
                     "stream" if stream else "nostream")
            if stream:
                return await self._stream_ollama(request, url, headers, body, req_model)
            return await self._proxy_ollama_nonstream(request, url, headers, body, req_model)
        # 跨协议: ollama 客户端 -> OpenAI 上游
        LOG.info("ollama /api/chat model=%s -> %s (convert)", req_model, target["via"])
        return await self._proxy_ollama_chat_via_openai(request, body, target, stream, req_model)

    async def handle_ollama_generate(self, request):
        if not self._check_auth(request):
            return _json_err(401, "invalid api key")
        try:
            body = await request.json()
        except Exception:
            return _json_err(400, "invalid json body")
        req_model = body.get("model")
        target, err = await self.route(req_model, "ollama")
        if not target:
            return _json_err(502, err)
        stream = bool(body.get("stream"))
        headers = self._up_headers(request, target)
        if target["proto"] == "ollama":
            body["model"] = target["model"]
            url = target["base"] + "/api/generate"
            LOG.info("ollama /api/generate model=%s -> %s (%s)", req_model, target["via"],
                     "stream" if stream else "nostream")
            if stream:
                return await self._stream_ollama(request, url, headers, body, req_model)
            return await self._proxy_ollama_nonstream(request, url, headers, body, req_model)
        LOG.info("ollama /api/generate model=%s -> %s (convert)", req_model, target["via"])
        return await self._proxy_ollama_generate_via_openai(request, body, target, stream, req_model)

    async def handle_ollama_tags(self, request):
        if not self._check_auth(request):
            return _json_err(401, "invalid api key")
        models = []
        for m in self.cfg.models:
            if m == "default":
                continue
            models.append({"name": m, "model": m, "modified_at": _now_iso(),
                           "size": 0, "digest": "", "details": {}})
        return web.json_response({"models": models})

    async def _stream_ollama(self, request, url, headers, body, req_model):
        data = json.dumps(body, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                if up.status != 200:
                    txt = await up.text()
                    return _json_err(up.status, f"upstream error: {txt[:500]}")
                resp = web.StreamResponse(status=200, headers={"Content-Type": "application/x-ndjson"})
                await resp.prepare(request)
                sp = LineSplitter("\n")
                async for chunk in up.content.iter_any():
                    for line in sp.feed(chunk):
                        await resp.write(_rewrite_ndjson_line(line, req_model) + b"\n")
                rest = sp.flush()
                if rest:
                    await resp.write(_rewrite_ndjson_line(rest, req_model) + b"\n")
                await resp.write_eof()
                return resp
        except Exception as e:
            return _json_err(502, f"upstream stream error: {e}")

    async def _proxy_ollama_nonstream(self, request, url, headers, body, req_model):
        data = json.dumps(body, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                raw = await up.read()
                if up.status != 200:
                    return web.Response(status=up.status, body=raw, content_type="application/json")
                try:
                    obj = json.loads(raw)
                    if isinstance(obj, dict) and "model" in obj:
                        obj["model"] = req_model
                    raw = json.dumps(obj, ensure_ascii=False).encode()
                except Exception:
                    pass
                return web.Response(status=up.status, body=raw, content_type="application/json")
        except Exception as e:
            return _json_err(502, f"upstream error: {e}")

    # ---- 跨协议转换实现 ----
    async def _proxy_ollama_chat_via_openai(self, request, body, target, stream, req_model):
        oai = _ollama_chat_to_openai(body, target["model"])
        url = target["base"] + "/chat/completions"
        headers = self._up_headers(request, target)
        if stream:
            return await self._stream_openai_to_ollama_chat(request, url, headers, oai, req_model)
        return await self._nonstream_openai_to_ollama_chat(request, url, headers, oai, req_model)

    async def _proxy_ollama_generate_via_openai(self, request, body, target, stream, req_model):
        oai = _ollama_generate_to_openai(body, target["model"])
        url = target["base"] + "/completions"
        headers = self._up_headers(request, target)
        if stream:
            return await self._stream_openai_to_ollama_generate(request, url, headers, oai, req_model)
        return await self._nonstream_openai_to_ollama_generate(request, url, headers, oai, req_model)

    async def _nonstream_openai_to_ollama_chat(self, request, url, headers, oai, req_model):
        data = json.dumps(oai, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                raw = await up.read()
                if up.status != 200:
                    return web.Response(status=up.status, body=raw, content_type="application/json")
                obj = json.loads(raw)
                choice = (obj.get("choices") or [{}])[0]
                msg = choice.get("message", {}) or {}
                usage = obj.get("usage") or {}
                out = {
                    "model": req_model, "created_at": _now_iso(),
                    "message": {"role": msg.get("role", "assistant"), "content": msg.get("content", "")},
                    "done": True, "done_reason": choice.get("finish_reason", "stop"),
                    "eval_count": usage.get("completion_tokens"),
                    "prompt_eval_count": usage.get("prompt_tokens"),
                }
                return web.json_response(out)
        except Exception as e:
            return _json_err(502, f"upstream error: {e}")

    async def _stream_openai_to_ollama_chat(self, request, url, headers, oai, req_model):
        data = json.dumps(oai, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                if up.status != 200:
                    txt = await up.text()
                    return _json_err(up.status, f"upstream error: {txt[:500]}")
                resp = web.StreamResponse(status=200, headers={"Content-Type": "application/x-ndjson"})
                await resp.prepare(request)
                sp = LineSplitter("\n\n")
                saw_done = False
                async for chunk in up.content.iter_any():
                    for unit in sp.feed(chunk):
                        lines, d = _sse_unit_to_ollama_chat_lines(unit, req_model)
                        if d:
                            saw_done = True
                        for ln in lines:
                            await resp.write(ln + b"\n")
                rest = sp.flush()
                if rest:
                    lines, d = _sse_unit_to_ollama_chat_lines(rest, req_model)
                    if d:
                        saw_done = True
                    for ln in lines:
                        await resp.write(ln + b"\n")
                if not saw_done:
                    await resp.write(json.dumps({
                        "model": req_model, "created_at": _now_iso(),
                        "message": {"role": "assistant", "content": ""},
                        "done": True, "done_reason": "stop",
                    }, ensure_ascii=False).encode() + b"\n")
                await resp.write_eof()
                return resp
        except Exception as e:
            return _json_err(502, f"upstream stream error: {e}")

    async def _nonstream_openai_to_ollama_generate(self, request, url, headers, oai, req_model):
        data = json.dumps(oai, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                raw = await up.read()
                if up.status != 200:
                    return web.Response(status=up.status, body=raw, content_type="application/json")
                obj = json.loads(raw)
                ch = (obj.get("choices") or [{}])[0]
                out = {
                    "model": req_model, "created_at": _now_iso(),
                    "response": ch.get("text", ""), "done": True,
                    "done_reason": ch.get("finish_reason", "stop"), "context": [],
                }
                return web.json_response(out)
        except Exception as e:
            return _json_err(502, f"upstream error: {e}")

    async def _stream_openai_to_ollama_generate(self, request, url, headers, oai, req_model):
        data = json.dumps(oai, ensure_ascii=False).encode()
        try:
            async with self.session.post(url, headers=headers, data=data) as up:
                if up.status != 200:
                    txt = await up.text()
                    return _json_err(up.status, f"upstream error: {txt[:500]}")
                resp = web.StreamResponse(status=200, headers={"Content-Type": "application/x-ndjson"})
                await resp.prepare(request)
                sp = LineSplitter("\n\n")
                saw_done = False
                async for chunk in up.content.iter_any():
                    for unit in sp.feed(chunk):
                        lines, d = _sse_unit_to_ollama_generate_lines(unit, req_model)
                        if d:
                            saw_done = True
                        for ln in lines:
                            await resp.write(ln + b"\n")
                rest = sp.flush()
                if rest:
                    lines, d = _sse_unit_to_ollama_generate_lines(rest, req_model)
                    if d:
                        saw_done = True
                    for ln in lines:
                        await resp.write(ln + b"\n")
                if not saw_done:
                    await resp.write(json.dumps({
                        "model": req_model, "created_at": _now_iso(),
                        "response": "", "done": True, "done_reason": "stop",
                    }, ensure_ascii=False).encode() + b"\n")
                await resp.write_eof()
                return resp
        except Exception as e:
            return _json_err(502, f"upstream stream error: {e}")

    # ---- 健康检查 ----
    async def handle_health(self, request):
        states = {}
        for s in self.cfg.servers:
            st = self.gpu._st.get(s, {})
            states[s] = {"idle": st.get("idle"), "err": st.get("err"),
                         "age_s": round(time.time() - st.get("ts", 0), 1) if st.get("ts") else None}
        return web.json_response({"ok": True, "servers": states, "models": list(self.cfg.models)})


# ==================== 入口 ====================
def build_app(cfg):
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
    port = int(port) if port else 8080
    web.run_app(build_app(cfg), host=host or "0.0.0.0", port=port, print=None)


if __name__ == "__main__":
    main()
