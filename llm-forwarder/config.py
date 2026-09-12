"""配置加载与模型路由解析。"""
import yaml


class Config:
    """从 YAML 加载配置, 提供模型路由解析。"""

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
        """返回 (server_key, local_model, network_model) 或 None (无匹配且无 default)。"""
        m = self.models.get(name) or self.models.get("default")
        if not m:
            return None
        return m.get("server"), m.get("local_model", name), m.get("network_model", name)
