"""可复用的上游代理原语: 非流式 / 流式透传 / 跨协议流式转换。

将重复的 POST-读-返回 / 流式切分-重写-写回逻辑收敛到此, 由 forwarder 复用。
"""
import json

from aiohttp import web

from util import LineSplitter, json_err


async def post_json(session, url, headers, body):
    """POST JSON, 返回 (status, raw_bytes, error_str)。"""
    data = json.dumps(body, ensure_ascii=False).encode()
    try:
        async with session.post(url, headers=headers, data=data) as up:
            raw = await up.read()
            return up.status, raw, None
    except Exception as e:
        return 0, b"", f"upstream error: {e}"


async def proxy_json(session, request, url, headers, body, model):
    """非流式 JSON 代理: POST, 重写响应 model 字段, 返回 Response。

    OpenAI 与 ollama 的非流式响应都是单段 JSON 且含 model 字段, 共用此函数。
    """
    status, raw, err = await post_json(session, url, headers, body)
    if err:
        return json_err(502, err)
    if status != 200:
        return web.Response(status=status, body=raw, content_type="application/json")
    try:
        obj = json.loads(raw)
        if isinstance(obj, dict) and "model" in obj:
            obj["model"] = model
        raw = json.dumps(obj, ensure_ascii=False).encode()
    except Exception:
        pass
    return web.Response(status=status, body=raw, content_type="application/json")


async def stream_passthrough(session, request, url, headers, body,
                             sep, unit_rewriter, content_type):
    """流式透传: 按 sep 切分上游字节流, unit_rewriter 重写每个单元后写回客户端。

    OpenAI(SSE, sep=\\n\\n) 与 ollama(NDJSON, sep=\\n) 的同协议流式转发共用此函数。
    """
    data = json.dumps(body, ensure_ascii=False).encode()
    sep_b = sep.encode() if isinstance(sep, str) else sep
    try:
        async with session.post(url, headers=headers, data=data) as up:
            if up.status != 200:
                txt = await up.text()
                return json_err(up.status, f"upstream error: {txt[:500]}")
            resp = web.StreamResponse(status=200, headers={"Content-Type": content_type})
            await resp.prepare(request)
            sp = LineSplitter(sep)
            async for chunk in up.content.iter_any():
                for unit in sp.feed(chunk):
                    await resp.write(unit_rewriter(unit) + sep_b)
            rest = sp.flush()
            if rest:
                await resp.write(unit_rewriter(rest) + sep_b)
            await resp.write_eof()
            return resp
    except Exception as e:
        return json_err(502, f"upstream stream error: {e}")


async def stream_openai_to_ollama(session, request, url, headers, oai_body,
                                  req_model, unit_converter, final_line):
    """OpenAI SSE 流 -> ollama NDJSON 流。

    unit_converter(unit, model) -> (lines, saw_done); final_line 为上游未发结束时的兜底行。
    chat 与 generate 的转换共用此函数, 仅 unit_converter/final_line 不同。
    """
    data = json.dumps(oai_body, ensure_ascii=False).encode()
    try:
        async with session.post(url, headers=headers, data=data) as up:
            if up.status != 200:
                txt = await up.text()
                return json_err(up.status, f"upstream error: {txt[:500]}")
            resp = web.StreamResponse(status=200, headers={"Content-Type": "application/x-ndjson"})
            await resp.prepare(request)
            sp = LineSplitter("\n\n")
            saw_done = False
            async for chunk in up.content.iter_any():
                for unit in sp.feed(chunk):
                    lines, d = unit_converter(unit, req_model)
                    if d:
                        saw_done = True
                    for ln in lines:
                        await resp.write(ln + b"\n")
            rest = sp.flush()
            if rest:
                lines, d = unit_converter(rest, req_model)
                if d:
                    saw_done = True
                for ln in lines:
                    await resp.write(ln + b"\n")
            if not saw_done:
                await resp.write(final_line + b"\n")
            await resp.write_eof()
            return resp
    except Exception as e:
        return json_err(502, f"upstream stream error: {e}")


async def nonstream_openai_to_ollama(session, url, headers, oai_body, req_model, shaper):
    """非流式 OpenAI 响应 -> ollama 响应。shaper(oai_obj, model) -> ollama_dict。"""
    status, raw, err = await post_json(session, url, headers, oai_body)
    if err:
        return json_err(502, err)
    if status != 200:
        return web.Response(status=status, body=raw, content_type="application/json")
    try:
        return web.json_response(shaper(json.loads(raw), req_model))
    except Exception as e:
        return json_err(502, f"convert error: {e}")
