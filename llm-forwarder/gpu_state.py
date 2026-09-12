"""GPU 状态缓存: SSH 检查 + 按 TTL 缓存 + 后台巡检。"""
import asyncio
import logging
import os
import time

import asyncssh

LOG = logging.getLogger("gpu_state")


class GpuState:
    """每台 GPU 服务器的空闲状态缓存, 按 check_interval 失效, 带后台巡检。"""

    def __init__(self, cfg):
        self.cfg = cfg
        self._st = {s: {"idle": None, "ts": 0.0, "err": None} for s in cfg.servers}
        self._lk = {s: asyncio.Lock() for s in cfg.servers}

    async def get(self, server):
        """返回服务器状态 dict {idle, ts, err}; 缓存过期则刷新。"""
        st = self._st.get(server)
        if st is None:
            return {"idle": None, "ts": 0.0, "err": "unknown server"}
        if st["idle"] is not None and time.time() - st["ts"] < self.cfg.check_interval:
            return st
        async with self._lk[server]:  # 加锁防惊群
            st = self._st[server]
            if st["idle"] is not None and time.time() - st["ts"] < self.cfg.check_interval:
                return st
            await self._refresh(server)
            return self._st[server]

    async def _connect(self, ssh):
        host = ssh["host"]
        base = dict(host=host, port=int(ssh.get("port", 22)),
                    username=ssh.get("user"), known_hosts=ssh.get("known_hosts"))
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
            idle = self._on_fail_idle()
            LOG.warning("[%s] GPU check failed (%s) -> idle=%s", server, err, idle)
        finally:
            if conn:
                await self._close(conn)
        self._st[server] = {"idle": idle, "ts": time.time(), "err": err}

    def _on_fail_idle(self):
        """SSH 失败时按配置策略决定 idle 值。"""
        pol = self.cfg.on_ssh_fail
        if pol == "idle":
            return True
        if pol == "error":
            return None
        return False  # busy

    async def _close(self, conn):
        try:
            conn.close()
        except Exception:
            pass
        try:
            await asyncio.wait_for(conn.wait_closed(), timeout=2)
        except Exception:
            pass

    async def background_loop(self):
        """后台巡检, 保持缓存新鲜, 降低请求时延。"""
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
