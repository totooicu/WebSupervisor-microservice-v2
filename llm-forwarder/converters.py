"""跨协议转换: Ollama 客户端 <-> OpenAI 上游。

ollama 客户端繁忙且仅配了 OpenAI 上游时, 把 ollama 请求转成 OpenAI 请求,
再把 OpenAI 响应(含 SSE 流)转回 ollama 格式。
"""
import json

from util import now_iso


def _copy_opts(opt, out):
    """把 ollama options 中的通用字段映射到 OpenAI 请求参数。"""
    if "temperature" in opt:
        out["temperature"] = opt["temperature"]
    if "top_p" in opt:
        out["top_p"] = opt["top_p"]
    if "num_predict" in opt:
        out["max_tokens"] = opt["num_predict"]
    if "seed" in opt:
        out["seed"] = opt["seed"]


def ollama_chat_to_openai(body, model):
    """ollama /api/chat 请求体 -> OpenAI /v1/chat/completions 请求体。"""
    out = {"model": model, "messages": body.get("messages", []),
           "stream": bool(body.get("stream", False))}
    _copy_opts(body.get("options") or {}, out)
    if body.get("format") == "json":
        out["response_format"] = {"type": "json_object"}
    return out


def ollama_generate_to_openai(body, model):
    """ollama /api/generate 请求体 -> OpenAI /v1/completions 请求体。"""
    out = {"model": model, "prompt": body.get("prompt", ""),
           "stream": bool(body.get("stream", False))}
    _copy_opts(body.get("options") or {}, out)
    return out


def sse_to_ollama_lines(unit, model, kind):
    """OpenAI SSE 事件块 -> ollama NDJSON 行列表。

    kind='chat' 输出 message 结构; kind='generate' 输出 response 结构。
    返回 (lines, saw_done)。
    """
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
        ch = (obj.get("choices") or [{}])[0]
        finish = ch.get("finish_reason")
        if finish:
            saw_done = True
            lines.append(_done_row(model, kind, finish))
            continue
        if kind == "chat":
            delta = ch.get("delta", {}) or {}
            content = delta.get("content", "")
            if content or delta.get("role"):
                lines.append(json.dumps({"model": model, "created_at": now_iso(),
                    "message": {"role": delta.get("role", "assistant"), "content": content},
                    "done": False}, ensure_ascii=False).encode())
        else:
            text = ch.get("text", "")
            if text:
                lines.append(json.dumps({"model": model, "created_at": now_iso(),
                    "response": text, "done": False}, ensure_ascii=False).encode())
    return lines, saw_done


def _done_row(model, kind, reason):
    """构造一个 ollama 结束行 (上游发了 finish_reason 时)。"""
    row = {"model": model, "created_at": now_iso(), "done": True, "done_reason": reason}
    row["message" if kind == "chat" else "response"] = (
        {"role": "assistant", "content": ""} if kind == "chat" else "")
    return json.dumps(row, ensure_ascii=False).encode()


def final_done_line(model, kind):
    """构造兜底结束行 (上游未发 finish_reason / [DONE] 时保证流正常关闭)。"""
    return _done_row(model, kind, "stop")


def shape_chat_response(oai_obj, model):
    """OpenAI /v1/chat/completions 响应 -> ollama /api/chat 响应。"""
    choice = (oai_obj.get("choices") or [{}])[0]
    msg = choice.get("message", {}) or {}
    usage = oai_obj.get("usage") or {}
    return {"model": model, "created_at": now_iso(),
            "message": {"role": msg.get("role", "assistant"), "content": msg.get("content", "")},
            "done": True, "done_reason": choice.get("finish_reason", "stop"),
            "eval_count": usage.get("completion_tokens"),
            "prompt_eval_count": usage.get("prompt_tokens")}


def shape_generate_response(oai_obj, model):
    """OpenAI /v1/completions 响应 -> ollama /api/generate 响应。"""
    ch = (oai_obj.get("choices") or [{}])[0]
    return {"model": model, "created_at": now_iso(),
            "response": ch.get("text", ""), "done": True,
            "done_reason": ch.get("finish_reason", "stop"), "context": []}
