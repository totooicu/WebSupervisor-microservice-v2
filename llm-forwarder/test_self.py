#!/usr/bin/env python3
"""自测: 纯函数 + 真实 HTTP 转发 (双协议/空闲繁忙/流式/跨协议转换)。

不依赖真实 SSH/GPU; 用 mock 上游 + 直接注入 GPU 状态。运行: python3 test_self.py
"""
import asyncio
import json
import socket

import converters as cv
import llm_forwarder as F  # noqa: 验证入口薄封装可导入全部公共 API
import rewriters as rw
import util
from aiohttp import ClientSession, web
from mock_upstream import build_mock_app


def free_port():
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    p = s.getsockname()[1]
    s.close()
    return p


async def run_app(app, port):
    runner = web.AppRunner(app)
    await runner.setup()
    await web.TCPSite(runner, "127.0.0.1", port).start()
    return runner


def test_pure():
    # LineSplitter + SSE 重写
    sp = util.LineSplitter("\n\n")
    out = sp.feed(b'data: {"model":"x","choices":[{"delta":{"content":"hi"}}]}\n\n')
    assert out == [b'data: {"model":"x","choices":[{"delta":{"content":"hi"}}]}']
    assert json.loads(rw.rewrite_sse_model(out[0], "want")[6:])["model"] == "want"

    # NDJSON 重写
    sp2 = util.LineSplitter("\n")
    out2 = sp2.feed(b'{"model":"x","message":{"content":"a"}}\n{"model":"x","done":true}\n')
    assert len(out2) == 2
    assert json.loads(rw.rewrite_ndjson_model(out2[0], "want"))["model"] == "want"

    # ollama->openai 请求转换 (options 映射)
    oai = cv.ollama_chat_to_openai(
        {"model": "m", "messages": [{"role": "user", "content": "hi"}], "stream": True,
         "options": {"temperature": 0.5, "num_predict": 10}}, "gpt4")
    assert oai["model"] == "gpt4" and oai["max_tokens"] == 10 and oai["temperature"] == 0.5

    # SSE -> ollama NDJSON 转换 (含 finish + [DONE])
    unit = (b'data: {"choices":[{"delta":{"content":"he"}}]}\n\n'
            b'data: {"choices":[{"finish_reason":"stop"}]}\n\n'
            b'data: [DONE]\n\n')
    sp3 = util.LineSplitter("\n\n")
    lines, saw = [], False
    for p in sp3.feed(unit):
        ls, d = cv.sse_to_ollama_lines(p, "m", "chat")
        if d:
            saw = True
        lines.extend(ls)
    assert saw and "he" in [json.loads(l)["message"]["content"] for l in lines]
    print("  [pure] OK")


async def test_http():
    up_runner = await run_app(build_mock_app(), free_port())
    base = f"http://127.0.0.1:{up_runner.addresses[0][1]}"

    cfg = F.Config.__new__(F.Config)  # 直接构造, 跳过文件加载
    for k, v in dict(raw={}, listen="127.0.0.1:0", api_keys=set(), check_interval=10,
                      ssh_timeout=8, on_ssh_fail="busy", request_timeout=30,
                      servers={"s1": {"local_endpoint": base}},
                      network={"openai": {"base_url": base + "/v1", "api_key": ""},
                               "ollama": {"base_url": "", "api_key": ""}},
                      models={"m": {"server": "s1", "local_model": "mlocal", "network_model": "gpt4"},
                              "default": {"server": "s1", "local_model": "mlocal", "network_model": "gpt4"}}
                      ).items():
        setattr(cfg, k, v)

    app = F.build_app(cfg)
    fwd = app["fwd"]
    state = {"idle": True}
    fwd.gpu.get = lambda server: _idle(state)  # 注入 GPU 状态, 跳过 SSH

    app_runner = await run_app(app, free_port())
    url = f"http://127.0.0.1:{app_runner.addresses[0][1]}"

    async with ClientSession() as cli:
        async def post(path, **kw):
            return await cli.post(url + path, **kw)

        # 1-2 idle openai (nostream/stream) -> local, model 重写为 "m"
        state["idle"] = True
        r = await post("/v1/chat/completions", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m" and obj["choices"][0]["message"]["content"] == "hi"
        r = await post("/v1/chat/completions", json={"model": "m", "stream": True, "messages": []})
        text = await r.text()
        assert "Hel" in text and "data: [DONE]" in text and ('"model": "m"' in text or '"model":"m"' in text)
        print("  [idle openai] OK")

        # 3-4 idle ollama 原生 (nostream/stream) -> local /api/chat
        r = await post("/api/chat", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m" and obj["message"]["content"] == "hi" and obj["done"]
        r = await post("/api/chat", json={"model": "m", "stream": True, "messages": []})
        lines = [json.loads(l) for l in (await r.text()).strip().split("\n") if l.strip()]
        assert any(l["message"]["content"] == "x" for l in lines) and lines[-1]["done"]
        assert all(l["model"] == "m" for l in lines)
        print("  [idle ollama] OK")

        # 5 busy openai nostream -> network openai
        state["idle"] = False
        r = await post("/v1/chat/completions", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m" and obj["choices"][0]["message"]["content"] == "hi"
        print("  [busy openai] OK")

        # 6 busy ollama nostream 跨协议转换 -> openai -> ollama
        r = await post("/api/chat", json={"model": "m", "messages": [{"role": "user", "content": "hi"}]})
        obj = await r.json()
        assert obj["model"] == "m" and obj["message"]["content"] == "hi" and obj["done"]
        print("  [busy ollama nostream convert] OK")

        # 7 busy ollama chat 流式跨协议转换 -> openai SSE -> ollama NDJSON
        r = await post("/api/chat", json={"model": "m", "stream": True, "messages": []})
        lines = [json.loads(l) for l in (await r.text()).strip().split("\n") if l.strip()]
        content = "".join(l["message"]["content"] for l in lines if not l["done"])
        assert "Hel" in content and "lo" in content and lines[-1]["done"] and all(l["model"] == "m" for l in lines)
        print("  [busy ollama chat stream convert] OK")

        # 8 busy ollama generate 流式跨协议转换 -> openai completions -> ollama generate
        r = await post("/api/generate", json={"model": "m", "stream": True, "prompt": "p"})
        lines = [json.loads(l) for l in (await r.text()).strip().split("\n") if l.strip()]
        resp = "".join(l.get("response", "") for l in lines if not l["done"])
        assert "A" in resp and "B" in resp and lines[-1]["done"]
        print("  [busy ollama generate stream convert] OK")

        # 9 health + v1/models + api/tags
        assert (await (await cli.get(url + "/health")).json())["ok"]
        assert any(d["id"] == "m" for d in (await (await cli.get(url + "/v1/models")).json())["data"])
        assert any(m["name"] == "m" for m in (await (await cli.get(url + "/api/tags")).json())["models"])
        print("  [health + models + tags] OK")

    await app_runner.cleanup()
    await up_runner.cleanup()
    print("  [http] OK")


async def _idle(state):
    return {"idle": state["idle"], "ts": 0.0, "err": None}


async def main():
    print("== pure function tests ==")
    test_pure()
    print("== http forwarding tests ==")
    await test_http()
    print("\nALL TESTS PASSED")


if __name__ == "__main__":
    asyncio.run(main())
