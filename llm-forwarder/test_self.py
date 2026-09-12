#!/usr/bin/env python3
"""自测脚本: 验证转发器纯函数 + 真实 HTTP 转发(双协议/空闲繁忙/流式/跨协议转换).
不依赖真实 SSH/GPU; 用 mock 上游 + 直接注入 GPU 状态.
运行: python3 test_self.py
"""
import asyncio
import json
import socket
import sys

import llm_forwarder as F
from aiohttp import ClientSession, web


def free_port():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(("127.0.0.1", 0))
    p = s.getsockname()[1]
    s.close()
    return p


async def run_app(app, port):
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "127.0.0.1", port)
    await site.start()
    return runner


# ---------- 纯函数 ----------
def test_pure():
    sp = F.LineSplitter("\n\n")
    out = sp.feed(b'data: {"model":"x","choices":[{"delta":{"content":"hi"}}]}\n\n')
    assert out == [b'data: {"model":"x","choices":[{"delta":{"content":"hi"}}]}'], out
    r = F._rewrite_sse_unit(out[0], "want")
    assert json.loads(r[6:])["model"] == "want", r

    sp2 = F.LineSplitter("\n")
    out2 = sp2.feed(b'{"model":"x","message":{"content":"a"}}\n{"model":"x","done":true}\n')
    assert len(out2) == 2, out2
    assert json.loads(F._rewrite_ndjson_line(out2[0], "want"))["model"] == "want"

    oai = F._ollama_chat_to_openai(
        {"model": "m", "messages": [{"role": "user", "content": "hi"}], "stream": True,
         "options": {"temperature": 0.5, "num_predict": 10}}, "gpt4")
    assert oai["model"] == "gpt4" and oai["max_tokens"] == 10 and oai["temperature"] == 0.5, oai

    unit = (b'data: {"choices":[{"delta":{"content":"he"}}]}\n\n'
            b'data: {"choices":[{"finish_reason":"stop"}]}\n\n'
            b'data: [DONE]\n\n')
    sp3 = F.LineSplitter("\n\n")
    parts = sp3.feed(unit)
    assert len(parts) == 3, parts
    lines, saw = [], False
    for p in parts:
        ls, d = F._sse_unit_to_ollama_chat_lines(p, "m")
        if d:
            saw = True
        lines.extend(ls)
    assert saw
    contents = [json.loads(l)["message"]["content"] for l in lines]
    assert "he" in contents, contents
    print("  [pure] OK")


# ---------- 真实转发 ----------
async def test_http():
    async def oai_chat(req):
        body = await req.json()
        if body.get("stream"):
            resp = web.StreamResponse(status=200, headers={"Content-Type": "text/event-stream"})
            await resp.prepare(req)
            for tok in ["Hel", "lo"]:
                await resp.write(b'data: ' + json.dumps(
                    {"model": body["model"], "choices": [{"delta": {"content": tok}}]}).encode() + b'\n\n')
            await resp.write(b'data: ' + json.dumps(
                {"model": body["model"], "choices": [{"finish_reason": "stop"}]}).encode() + b'\n\n')
            await resp.write(b'data: [DONE]\n\n')
            await resp.write_eof()
            return resp
        return web.json_response({"model": body["model"],
                                  "choices": [{"message": {"role": "assistant", "content": "hi"},
                                               "finish_reason": "stop"}],
                                  "usage": {"prompt_tokens": 1, "completion_tokens": 2}})

    async def oai_completions(req):
        body = await req.json()
        if body.get("stream"):
            resp = web.StreamResponse(status=200, headers={"Content-Type": "text/event-stream"})
            await resp.prepare(req)
            for tok in ["A", "B"]:
                await resp.write(b'data: ' + json.dumps(
                    {"model": body["model"], "choices": [{"text": tok}]}).encode() + b'\n\n')
            await resp.write(b'data: ' + json.dumps(
                {"model": body["model"], "choices": [{"finish_reason": "stop"}]}).encode() + b'\n\n')
            await resp.write(b'data: [DONE]\n\n')
            await resp.write_eof()
            return resp
        return web.json_response({"model": body["model"], "choices": [{"text": "hi", "finish_reason": "stop"}]})

    async def ollama_chat(req):
        body = await req.json()
        if body.get("stream"):
            resp = web.StreamResponse(status=200, headers={"Content-Type": "application/x-ndjson"})
            await resp.prepare(req)
            for tok in ["x", "y"]:
                await resp.write(json.dumps({"model": body["model"],
                              "message": {"role": "assistant", "content": tok}, "done": False}).encode() + b'\n')
            await resp.write(json.dumps({"model": body["model"],
                          "message": {"role": "assistant", "content": ""}, "done": True,
                          "done_reason": "stop"}).encode() + b'\n')
            await resp.write_eof()
            return resp
        return web.json_response({"model": body["model"], "message": {"role": "assistant", "content": "hi"}, "done": True})

    up = web.Application()
    up.router.add_post("/v1/chat/completions", oai_chat)
    up.router.add_post("/v1/completions", oai_completions)
    up.router.add_post("/api/chat", ollama_chat)
    up_port = free_port()
    up_runner = await run_app(up, up_port)
    base = f"http://127.0.0.1:{up_port}"

    # 构造配置: local_endpoint / network_upstream 都指向 mock
    cfg = F.Config.__new__(F.Config)
    cfg.raw = {}
    cfg.listen = "127.0.0.1:0"
    cfg.api_keys = set()
    cfg.check_interval = 10
    cfg.ssh_timeout = 8
    cfg.on_ssh_fail = "busy"
    cfg.request_timeout = 30
    cfg.servers = {"s1": {"local_endpoint": base}}
    cfg.network = {"openai": {"base_url": base + "/v1", "api_key": ""},
                   "ollama": {"base_url": "", "api_key": ""}}
    cfg.models = {"m": {"server": "s1", "local_model": "mlocal", "network_model": "gpt4"},
                  "default": {"server": "s1", "local_model": "mlocal", "network_model": "gpt4"}}

    app = F.build_app(cfg)
    fwd = app["fwd"]

    state = {"idle": True}

    async def fake_get(server):
        return {"idle": state["idle"], "ts": 0.0, "err": None}
    fwd.gpu.get = fake_get  # 注入 GPU 状态, 跳过 SSH

    app_port = free_port()
    app_runner = await run_app(app, app_port)
    url = f"http://127.0.0.1:{app_port}"

    async with ClientSession() as cli:
        async def post(path, **kw):
            return await cli.post(url + path, **kw)

        # 1. idle openai nostream -> local; model rewritten to "m"
        state["idle"] = True
        r = await post("/v1/chat/completions", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m", obj
        assert obj["choices"][0]["message"]["content"] == "hi", obj
        print("  [idle openai nostream] OK")

        # 2. idle openai stream -> local SSE; model rewritten
        r = await post("/v1/chat/completions", json={"model": "m", "stream": True, "messages": []})
        text = await r.text()
        assert "Hel" in text and "data: [DONE]" in text, text
        assert ('"model": "m"' in text) or ('"model":"m"' in text), text
        print("  [idle openai stream] OK")

        # 3. idle ollama nostream native -> local /api/chat
        r = await post("/api/chat", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m" and obj["message"]["content"] == "hi" and obj["done"], obj
        print("  [idle ollama nostream] OK")

        # 4. idle ollama stream native -> local NDJSON
        r = await post("/api/chat", json={"model": "m", "stream": True, "messages": []})
        text = await r.text()
        lines = [json.loads(l) for l in text.strip().split("\n") if l.strip()]
        assert any(l["message"]["content"] == "x" for l in lines), lines
        assert lines[-1]["done"], lines
        assert all(l["model"] == "m" for l in lines), lines
        print("  [idle ollama stream] OK")

        # 5. busy openai nostream -> network openai
        state["idle"] = False
        r = await post("/v1/chat/completions", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m" and obj["choices"][0]["message"]["content"] == "hi", obj
        print("  [busy openai nostream] OK")

        # 6. busy ollama nostream convert -> openai -> ollama
        r = await post("/api/chat", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m" and obj["message"]["content"] == "hi" and obj["done"], obj
        print("  [busy ollama nostream convert] OK")

        # 7. busy ollama stream convert -> openai SSE -> ollama NDJSON
        r = await post("/api/chat", json={"model": "m", "stream": True, "messages": []})
        text = await r.text()
        lines = [json.loads(l) for l in text.strip().split("\n") if l.strip()]
        contents = "".join(l["message"]["content"] for l in lines if not l["done"])
        assert "Hel" in contents and "lo" in contents, (contents, lines)
        assert lines[-1]["done"], lines
        assert all(l["model"] == "m" for l in lines), lines
        print("  [busy ollama stream convert] OK")

        # 8. busy ollama generate stream convert -> openai completions SSE -> ollama generate NDJSON
        r = await post("/api/generate", json={"model": "m", "stream": True, "prompt": "p"})
        text = await r.text()
        lines = [json.loads(l) for l in text.strip().split("\n") if l.strip()]
        resp = "".join(l.get("response", "") for l in lines if not l["done"])
        assert "A" in resp and "B" in resp, (resp, lines)
        assert lines[-1]["done"], lines
        print("  [busy ollama generate stream convert] OK")

        # 9. health
        r = await cli.get(url + "/health")
        obj = await r.json()
        assert obj["ok"], obj
        print("  [health] OK")

        # 10. /v1/models
        r = await cli.get(url + "/v1/models")
        obj = await r.json()
        assert any(d["id"] == "m" for d in obj["data"]), obj
        print("  [v1/models] OK")

    await app_runner.cleanup()
    await up_runner.cleanup()
    print("  [http] OK")


async def main():
    print("== pure function tests ==")
    test_pure()
    print("== http forwarding tests ==")
    await test_http()
    print("\nALL TESTS PASSED")


if __name__ == "__main__":
    asyncio.run(main())
