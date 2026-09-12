"""流式响应 model 字段重写: SSE 与 NDJSON。

转发到上游时 model 名被替换成上游实际模型名, 返回客户端前需改回客户端请求的名字。
"""
import json


def rewrite_sse_model(unit, model):
    """重写一个 SSE 事件块(bytes, 不含尾随分隔符) 中 data: JSON 的 model 字段。"""
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


def rewrite_ndjson_model(line, model):
    """重写一行 NDJSON(bytes) 中的 model 字段; 非 JSON 行原样返回。"""
    if not line.strip():
        return line
    try:
        obj = json.loads(line)
        if isinstance(obj, dict) and "model" in obj:
            obj["model"] = model
        return json.dumps(obj, ensure_ascii=False).encode()
    except Exception:
        return line
