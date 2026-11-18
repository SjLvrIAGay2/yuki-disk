import { useState as h, useEffect as b } from "react";
function L(t) {
  const [e, n] = h();
  return b(() => {
    if (typeof t == "string" || !t) return;
    const a = URL.createObjectURL(t);
    return n({ source: t, url: a }), () => URL.revokeObjectURL(a);
  }, [t]), typeof t == "string" ? t : e?.source === t ? e?.url : void 0;
}
function v(t, e) {
  return {
    duration: Number.isFinite(t.duration) ? t.duration : null,
    ..."videoWidth" in t ? {
      width: t.videoWidth,
      height: t.videoHeight
    } : {},
    src: t.currentSrc || t.src,
    ...e && typeof e != "string" ? {
      mimeType: e.type,
      size: e.size,
      filename: "name" in e ? String(e.name) : void 0
    } : {}
  };
}
function M(t) {
  const e = {
    1: "媒体加载已中止。",
    2: "无法加载媒体，请检查网络或文件地址。",
    3: "媒体解码失败，文件可能已损坏。",
    4: "浏览器不支持此媒体格式，或文件不可访问。"
  };
  return {
    code: t?.code ?? 0,
    message: e[t?.code ?? 0] ?? "媒体加载失败，请尝试其他文件。"
  };
}
function y(t) {
  if (t === null || !Number.isFinite(t)) return "--:--";
  const e = Math.max(0, Math.floor(t)), n = Math.floor(e / 3600);
  return `${n ? `${n}:` : ""}${String(Math.floor(e / 60) % 60).padStart(2, "0")}:${String(e % 60).padStart(2, "0")}`;
}
async function E(t, {
  kind: e,
  signal: n,
  timeout: a = 15e3
}) {
  if (typeof document > "u") throw new Error("媒体元数据解析需要浏览器环境。");
  n?.throwIfAborted();
  const r = document.createElement(e), d = typeof t == "string" ? t : URL.createObjectURL(t);
  return new Promise((u, l) => {
    let s = !1;
    const c = () => {
      clearTimeout(m), n?.removeEventListener("abort", f), r.onloadedmetadata = null, r.onerror = null, r.removeAttribute("src"), r.load(), typeof t != "string" && URL.revokeObjectURL(d);
    }, o = (i) => {
      if (s) return;
      s = !0;
      const p = i ? void 0 : v(r, t);
      c(), i ? l(i) : u(p);
    }, f = () => o(new DOMException("解析已取消", "AbortError")), m = setTimeout(() => o(new Error("媒体元数据读取超时。")), Math.max(1, a));
    r.preload = "metadata", r.onloadedmetadata = () => o(), r.onerror = () => o(new Error(M(r.error).message)), n?.addEventListener("abort", f, { once: !0 }), r.src = d, r.load();
  });
}
export {
  y as f,
  M as m,
  E as p,
  v as r,
  L as u
};

