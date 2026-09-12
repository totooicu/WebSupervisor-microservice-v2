"""路由决策: 请求模型名 + GPU 状态 -> 目标上游。"""


async def route(cfg, gpu, req_model, incoming_proto):
    """返回 (target_dict, error_str)。

    target: {base, proto, model, auth, via, idle}。
    incoming_proto: 'openai' 或 'ollama'。
    """
    res = cfg.resolve_model(req_model)
    if not res:
        return None, f"model '{req_model}' not configured and no 'default' mapping"
    server_key, local_model, network_model = res
    if server_key not in cfg.servers:
        return None, f"model maps to unknown server '{server_key}'"
    state = await gpu.get(server_key)
    idle = state["idle"]
    if idle is True:
        return _local_target(cfg, server_key, local_model, incoming_proto), None
    if idle is False:
        return route_network(cfg, incoming_proto, network_model)
    return None, f"GPU state unavailable for '{server_key}' ({state.get('err')})"


def _local_target(cfg, server_key, local_model, incoming_proto):
    """空闲: 转发到该机本地 ollama (openai 客户端用其 /v1 兼容口)。"""
    scfg = cfg.servers[server_key]
    local_ep = (scfg.get("local_endpoint") or "").rstrip("/")
    if incoming_proto == "openai":
        return {"base": local_ep + "/v1", "proto": "openai", "model": local_model,
                "auth": None, "via": f"local:{server_key}", "idle": True}
    return {"base": local_ep, "proto": "ollama", "model": local_model,
            "auth": None, "via": f"local:{server_key}", "idle": True}


def route_network(cfg, incoming_proto, network_model):
    """GPU 繁忙时的网络上游路由。"""
    if incoming_proto == "openai":
        return _net_openai(cfg, network_model)
    net = cfg.network
    up = net.get("ollama", {}) or {}
    if up.get("base_url"):
        return {"base": up["base_url"].rstrip("/"), "proto": "ollama", "model": network_model,
                "auth": ("Bearer " + up["api_key"]) if up.get("api_key") else None,
                "via": "network:ollama", "idle": False}, None
    target, err = _net_openai(cfg, network_model)
    if target:
        target["via"] = "network:openai(convert)"  # 需跨协议转换
    return target, err


def _net_openai(cfg, network_model):
    """OpenAI 兼容网络上游。"""
    up = cfg.network.get("openai", {}) or {}
    base = (up.get("base_url") or "").rstrip("/")
    if not base:
        return None, "no network openai upstream configured"
    return {"base": base, "proto": "openai", "model": network_model,
            "auth": ("Bearer " + up["api_key"]) if up.get("api_key") else None,
            "via": "network:openai", "idle": False}, None
