"""通用工具: 时间戳、错误响应、字节流切分、上游请求头。"""
from datetime import datetime, timezone

from aiohttp import web


def now_iso():
    """UTC ISO8601 时间戳字符串。"""
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def json_err(status, msg):
    """构造 OpenAI 风格的 JSON 错误响应。"""
    return web.json_response(
        {"error": {"message": msg, "type": "forwarder_error", "code": status}},
        status=status,
    )


def up_headers(request, target, api_keys):
    """构造转发到上游的请求头: 优先用上游专属 Key, 否则透传客户端 Key。"""
    h = {"Content-Type": "application/json", "Accept": "*/*"}
    if target.get("auth"):
        h["Authorization"] = target["auth"]
    elif api_keys:
        a = request.headers.get("Authorization")
        if a:
            h["Authorization"] = a
    return h


class LineSplitter:
    """按分隔符切分字节流, 保留未结束尾巴缓冲, 逐单元产出。

    用于流式转发: 上游按 chunk 到达, 完整单元可能跨 chunk, 需缓冲拼装。
    """

    __slots__ = ("buf", "sep")

    def __init__(self, sep):
        self.buf = bytearray()
        self.sep = sep.encode() if isinstance(sep, str) else sep

    def feed(self, data):
        """喂入一段字节, 返回完整单元列表 (不含分隔符); 未结束部分留在缓冲。"""
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
        """返回缓冲中剩余的未结束单元并清空。"""
        r = bytes(self.buf)
        self.buf.clear()
        return r
