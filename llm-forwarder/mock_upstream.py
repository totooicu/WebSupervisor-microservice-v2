"""测试用 mock 上游: 同时模拟 OpenAI 与 Ollama 端点, 供转发器转发。"""
import json

from aiohttp import web


async def _stream_sse(req, model, tokens, choice_builder):
    """复用: 生成 SSE 流, 每个 token 用 choice_builder 构造 choices 块。"""
    resp = web.StreamResponse(status=200, headers={"Content-Type": "text/event-stream"})
    await resp.prepare(req)
    for tok in tokens:
        await resp.write(b'data: ' + json.dumps(
            {"model": model, **choice_builder(tok)}).encode() + b'\n\n')
    await resp.write(b'data: ' + json.dumps(
        {"model": model, "choices": [{"finish_reason": "stop"}]}).encode() + b'\n\n')
    await resp.write(b'data: [DONE]\n\n')
    await resp.write_eof()
    return resp


async def oai_chat(req):
    body = await req.json()
    if body.get("stream"):
        return await _stream_sse(req, body["model"], ["Hel", "lo"],
                                 lambda t: {"choices": [{"delta": {"content": t}}]})
    return web.json_response({"model": body["model"],
        "choices": [{"message": {"role": "assistant", "content": "hi"}, "finish_reason": "stop"}],
        "usage": {"prompt_tokens": 1, "completion_tokens": 2}})


async def oai_completions(req):
    body = await req.json()
    if body.get("stream"):
        return await _stream_sse(req, body["model"], ["A", "B"],
                                 lambda t: {"choices": [{"text": t}]})
    return web.json_response({"model": body["model"],
        "choices": [{"text": "hi", "finish_reason": "stop"}]})


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
    return web.json_response({"model": body["model"],
        "message": {"role": "assistant", "content": "hi"}, "done": True})


def build_mock_app():
    app = web.Application()
    app.router.add_post("/v1/chat/completions", oai_chat)
    app.router.add_post("/v1/completions", oai_completions)
    app.router.add_post("/api/chat", ollama_chat)
    return app
