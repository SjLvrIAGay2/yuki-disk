import { jsx as o, jsxs as D } from "react/jsx-runtime";
import { Component as M, lazy as U, useId as R, useState as E, useRef as B, useEffect as P, Suspense as O } from "react";
import { Markdown as $ } from "./markdown.js";
import { CodeBlock as q } from "./code-block.js";
import { Button as H } from "./button.js";
import { c as V } from "./chunks/tokens-BqLZJxpY.js";
import './assets/document-viewer.css';const W = new Set("txt text log csv tsv nfo asc rst adoc asciidoc srt vtt ass ssa lrc".split(" ")), X = new Set("md markdown mdown mkd".split(" ")), G = new Set("readme license licence notice copying authors changelog .gitignore .gitattributes .dockerignore".split(" ")), F = /* @__PURE__ */ new Map();
for (const [e, t] of Object.entries({
  javascript: "js jsx mjs cjs",
  typescript: "ts tsx mts cts",
  xml: "html htm xml svg vue svelte xsd xsl xslt",
  css: "css",
  scss: "scss",
  less: "less",
  json: "json jsonl ndjson",
  python: "py pyw pyi",
  bash: "sh bash zsh",
  sql: "sql",
  yaml: "yaml yml",
  lua: "lua luau",
  ini: "ini cfg toml env config",
  properties: "properties",
  go: "go",
  rust: "rs",
  java: "java",
  c: "c h",
  cpp: "cpp hpp cc cxx hxx",
  csharp: "cs csx",
  kotlin: "kt kts",
  swift: "swift",
  ruby: "rb rake gemspec",
  php: "php phtml",
  r: "r",
  perl: "pl pm",
  scala: "scala sc",
  groovy: "groovy gradle",
  dart: "dart",
  powershell: "ps1 psm1 psd1",
  dos: "bat cmd",
  dockerfile: "dockerfile",
  makefile: "mk makefile",
  cmake: "cmake",
  diff: "diff patch",
  graphql: "graphql gql",
  protobuf: "proto",
  latex: "tex sty cls",
  apache: "htaccess",
  text: "conf cnf gitignore gitattributes dockerignore"
}))
  for (const n of t.split(" ")) F.set(n, e);
const J = new Map(Object.entries({
  dockerfile: "dockerfile",
  containerfile: "dockerfile",
  makefile: "makefile",
  gnumakefile: "makefile",
  "cmakelists.txt": "cmake",
  ".bashrc": "bash",
  ".bash_profile": "bash",
  ".zshrc": "bash",
  ".profile": "bash",
  ".gitconfig": "ini",
  ".editorconfig": "ini",
  ".npmrc": "ini",
  ".yarnrc": "ini",
  "nginx.conf": "nginx",
  ".htaccess": "apache",
  gemfile: "ruby",
  rakefile: "ruby"
}));
function I(e) {
  const t = e.split(/[\\/]/).pop()?.toLowerCase() ?? "", n = J.get(t);
  if (n) return { format: "code", language: n };
  if (/^(dockerfile|containerfile)\..+$/.test(t)) return { format: "code", language: "dockerfile" };
  if (t === ".env" || t.startsWith(".env.")) return { format: "code", language: "ini" };
  if (G.has(t)) return { format: "text", language: "text" };
  const r = t.split(".").pop() ?? "";
  if (X.has(r)) return { format: "markdown", language: "markdown" };
  const i = F.get(r);
  if (i) return { format: "code", language: i };
  if (W.has(r)) return { format: "text", language: "text" };
}
function K(e) {
  if (e) {
    if (typeof e != "string") return "name" in e ? String(e.name) : void 0;
    try {
      return decodeURIComponent(
        new URL(e, "https://local.invalid").pathname.split("/").pop() || ""
      ) || void 0;
    } catch {
      return;
    }
  }
}
function Q(e = "", t = "") {
  const n = e.split(".").pop()?.toLowerCase() ?? "";
  if (n === "pdf") return "pdf";
  if (n === "docx") return "docx";
  const r = I(e);
  if (r) return r.format;
  if (n !== "doc") {
    if (t === "application/pdf") return "pdf";
    if (t.includes("wordprocessingml.document")) return "docx";
    if (t === "text/markdown") return "markdown";
    if (t === "text/plain") return "text";
  }
}
function Y(e = "") {
  return I(e)?.language ?? "text";
}
async function Z(e, t, n) {
  t.throwIfAborted();
  const r = Number.isFinite(n) && n > 0 ? n : 20 * 1024 * 1024, i = () => new Error(`文件超过 ${Math.round(r / 1024 / 1024 * 10) / 10} MB 的预览限制。`);
  if (typeof e != "string") {
    if (e.size > r) throw i();
    const l = await e.arrayBuffer();
    return t.throwIfAborted(), { data: l, mimeType: e.type };
  }
  const s = await fetch(e, { signal: t });
  if (!s.ok) throw new Error(`文档加载失败（HTTP ${s.status}）。`);
  if (Number(s.headers.get("content-length")) > r)
    throw await s.body?.cancel(), i();
  const y = s.headers.get("content-type")?.split(";")[0].trim() ?? "";
  if (!s.body) {
    const l = await s.arrayBuffer();
    if (l.byteLength > r) throw i();
    return { data: l, mimeType: y };
  }
  const h = s.body.getReader(), x = [];
  let m = 0;
  try {
    for (; ; ) {
      t.throwIfAborted();
      const { done: l, value: g } = await h.read();
      if (l) break;
      if (m += g.byteLength, m > r)
        throw await h.cancel(), i();
      x.push(g);
    }
  } finally {
    h.releaseLock();
  }
  const u = new Uint8Array(m);
  let b = 0;
  for (const l of x)
    u.set(l, b), b += l.byteLength;
  return { data: u.buffer, mimeType: y };
}
function ee(e, t = "", n = "") {
  const r = new Uint8Array(e, 0, Math.min(8, e.byteLength));
  if (r[0] === 37 && r[1] === 80 && r[2] === 68 && r[3] === 70)
    return "pdf";
  const i = Q(t, n);
  if (i) return i;
  if (r[0] === 80 && r[1] === 75) return "docx";
  throw new Error("暂不支持此文件格式，请选择 PDF、DOCX、Markdown、文本或代码文件。");
}
class te extends M {
  state = {};
  static getDerivedStateFromError(t) {
    return { error: t };
  }
  componentDidCatch(t) {
    this.props.onError?.(t);
  }
  render() {
    return this.state.error ? /* @__PURE__ */ o("p", { className: "yk-document-viewer__error", role: "alert", children: "暂时无法预览，请刷新页面后重试。" }) : this.props.children;
  }
}
const re = U(() => import("./chunks/pdf-preview-TqkJ_y-Z.js"));
function le({
  src: e,
  content: t,
  format: n = "auto",
  filename: r,
  language: i,
  maxFileSize: s = 20 * 1024 * 1024,
  showHeader: y = !0,
  pdfOptions: h,
  markdownOptions: x,
  onLoad: m,
  onError: u,
  className: b,
  ...l
}) {
  const g = R(), p = r ?? K(e), _ = Number.isFinite(s) && s > 0 ? s : 20 * 1024 * 1024, [c, C] = E(), [L, T] = E(""), [v, S] = E(0), N = B({ onLoad: m, onError: u });
  return N.current = { onLoad: m, onError: u }, P(() => {
    const f = new AbortController(), { signal: w } = f;
    return C(void 0), T(""), !e && t === void 0 ? () => f.abort() : ((async () => {
      const a = t !== void 0 ? {
        data: new TextEncoder().encode(t).buffer,
        mimeType: "text/markdown"
      } : await Z(e, w, _);
      if (a.data.byteLength > _) throw new Error("文档超过预览大小限制。");
      const d = n === "auto" ? ee(a.data, p, a.mimeType) : n;
      let k = { format: d, data: a.data }, A;
      if (d === "docx") {
        const { parseDocx: z } = await import("./chunks/parse-docx-Cw-A6E5a.js");
        w.throwIfAborted();
        const j = await z(a.data, `yuki-${g}-`);
        k = { ...k, text: j.text, html: j.html }, A = j.warnings;
      } else d !== "pdf" && (k.text = new TextDecoder("utf-8", { fatal: !0 }).decode(a.data));
      w.throwIfAborted(), C(k), d !== "pdf" && N.current.onLoad?.({
        format: d,
        filename: p,
        size: a.data.byteLength,
        text: k.text,
        warnings: A
      });
    })().catch((a) => {
      if (w.aborted) return;
      const d = new Error(a instanceof Error ? a.message : "文档解析失败。");
      T("暂时无法预览此文档，可下载后查看。"), N.current.onError?.(d);
    }), () => f.abort());
  }, [e, t, n, p, _, v, g]), /* @__PURE__ */ D("div", { ...l, className: V("yk-document-viewer", b), children: [
    y && /* @__PURE__ */ D("div", { className: "yk-document-viewer__header", children: [
      /* @__PURE__ */ o("span", { children: p ?? "文档预览" }),
      /* @__PURE__ */ o("span", { children: c?.format.toUpperCase() ?? (n === "auto" ? "AUTO" : n.toUpperCase()) })
    ] }),
    L ? /* @__PURE__ */ D("div", { className: "yk-document-viewer__empty", children: [
      /* @__PURE__ */ o("p", { role: "alert", className: "yk-document-viewer__error", children: L }),
      /* @__PURE__ */ o(H, { size: "sm", variant: "outline", onClick: () => S(v + 1), children: "重新加载文档" })
    ] }) : !e && t === void 0 ? /* @__PURE__ */ o("p", { className: "yk-document-viewer__loading", children: "选择文档开始预览" }) : c ? c.format === "pdf" ? /* @__PURE__ */ o(te, { onError: u, children: /* @__PURE__ */ o(
      O,
      {
        fallback: /* @__PURE__ */ o("p", { className: "yk-document-viewer__loading", role: "status", children: "正在打开 PDF…" }),
        children: /* @__PURE__ */ o(
          re,
          {
            data: c.data,
            filename: p,
            options: h,
            onLoad: m,
            onError: u
          },
          v
        )
      }
    ) }, v) : c.format === "markdown" ? /* @__PURE__ */ o("div", { className: "yk-document-viewer__body", children: /* @__PURE__ */ o($, { ...x, children: c.text ?? "" }) }) : c.format === "docx" ? /* @__PURE__ */ o("div", { className: "yk-document-viewer__body", children: /* @__PURE__ */ o(
      "article",
      {
        className: "yk-markdown yk-document-viewer__word",
        "aria-label": "Word 文档内容",
        onClick: (f) => {
          const a = (f.target instanceof Element ? f.target.closest('a[href^="#"]') : null)?.getAttribute("href");
          if (!a) return;
          const d = f.currentTarget.ownerDocument.getElementById(a.slice(1));
          d && f.currentTarget.contains(d) && (f.preventDefault(), d.scrollIntoView({ block: "nearest" }));
        },
        dangerouslySetInnerHTML: { __html: c.html ?? "" }
      }
    ) }) : /* @__PURE__ */ o(
      q,
      {
        code: c.text ?? "",
        language: c.format === "code" ? i ?? Y(p) : "text",
        showLineNumbers: c.format === "code",
        wrap: c.format === "text"
      }
    ) : /* @__PURE__ */ o("p", { className: "yk-document-viewer__loading", role: "status", children: "正在读取文档…" })
  ] });
}
export {
  le as DocumentViewer,
  Q as inferDocumentFormat
};

