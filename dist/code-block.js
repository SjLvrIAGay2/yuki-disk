import { jsxs as n, jsx as m } from "react/jsx-runtime";
import { useState as q, useRef as E, useEffect as I, useMemo as M } from "react";
import { createLowlight as S } from "lowlight";
import z from "highlight.js/lib/languages/javascript";
import F from "highlight.js/lib/languages/typescript";
import L from "highlight.js/lib/languages/xml";
import O from "highlight.js/lib/languages/css";
import $ from "highlight.js/lib/languages/json";
import B from "highlight.js/lib/languages/python";
import R from "highlight.js/lib/languages/bash";
import A from "highlight.js/lib/languages/sql";
import D from "highlight.js/lib/languages/yaml";
import G from "highlight.js/lib/languages/markdown";
import H from "highlight.js/lib/languages/lua";
import J from "highlight.js/lib/languages/ini";
import K from "highlight.js/lib/languages/go";
import P from "highlight.js/lib/languages/rust";
import Q from "highlight.js/lib/languages/java";
import U from "highlight.js/lib/languages/c";
import V from "highlight.js/lib/languages/cpp";
import W from "highlight.js/lib/languages/csharp";
import X from "highlight.js/lib/languages/kotlin";
import Y from "highlight.js/lib/languages/swift";
import Z from "highlight.js/lib/languages/ruby";
import rr from "highlight.js/lib/languages/php";
import or from "highlight.js/lib/languages/r";
import tr from "highlight.js/lib/languages/perl";
import er from "highlight.js/lib/languages/scala";
import ir from "highlight.js/lib/languages/groovy";
import mr from "highlight.js/lib/languages/dart";
import sr from "highlight.js/lib/languages/powershell";
import cr from "highlight.js/lib/languages/dos";
import pr from "highlight.js/lib/languages/dockerfile";
import ar from "highlight.js/lib/languages/makefile";
import lr from "highlight.js/lib/languages/cmake";
import nr from "highlight.js/lib/languages/diff";
import fr from "highlight.js/lib/languages/graphql";
import hr from "highlight.js/lib/languages/protobuf";
import dr from "highlight.js/lib/languages/latex";
import ur from "highlight.js/lib/languages/scss";
import yr from "highlight.js/lib/languages/less";
import kr from "highlight.js/lib/languages/properties";
import br from "highlight.js/lib/languages/nginx";
import xr from "highlight.js/lib/languages/apache";
import { I as vr, c as gr } from "./chunks/tokens-BqLZJxpY.js";
import './assets/code-block.css';const b = S({
  javascript: z,
  typescript: F,
  xml: L,
  css: O,
  json: $,
  python: B,
  bash: R,
  sql: A,
  yaml: D,
  markdown: G,
  lua: H,
  ini: J,
  go: K,
  rust: P,
  java: Q,
  c: U,
  cpp: V,
  csharp: W,
  kotlin: X,
  swift: Y,
  ruby: Z,
  php: rr,
  r: or,
  perl: tr,
  scala: er,
  groovy: ir,
  dart: mr,
  powershell: sr,
  dos: cr,
  dockerfile: pr,
  makefile: ar,
  cmake: lr,
  diff: nr,
  graphql: fr,
  protobuf: hr,
  latex: dr,
  scss: ur,
  less: yr,
  properties: kr,
  nginx: br,
  apache: xr
}), x = {
  js: "javascript",
  jsx: "javascript",
  ts: "typescript",
  tsx: "typescript",
  html: "xml",
  svg: "xml",
  vue: "xml",
  sh: "bash",
  shell: "bash",
  yml: "yaml",
  md: "markdown",
  py: "python",
  toml: "ini",
  cfg: "ini",
  env: "ini",
  luau: "lua",
  cs: "csharp",
  ps1: "powershell",
  bat: "dos",
  cmd: "dos",
  docker: "dockerfile",
  make: "makefile",
  proto: "protobuf",
  tex: "latex",
  kt: "kotlin",
  rb: "ruby",
  rs: "rust",
  gql: "graphql",
  patch: "diff",
  zsh: "bash",
  shellscript: "bash"
};
function Nr(i, r) {
  const o = r.toLowerCase(), t = Object.hasOwn(x, o) ? x[o] : o;
  return i.length <= 1e5 && b.registered(t) ? b.highlight(t, i) : void 0;
}
function wr(i) {
  const r = [[]];
  function o(t, c = []) {
    if (t.type === "text")
      t.value.split(`
`).forEach((p, s) => {
        s && r.push([]), p && r[r.length - 1].push({ value: p, className: c.join(" ") });
      });
    else {
      const p = t.type === "element" ? [...c, ...t.properties.className ?? []] : c;
      for (const s of t.children)
        (s.type === "element" || s.type === "text") && o(s, p);
    }
  }
  return o(i), r;
}
function yo({
  code: i,
  language: r = "text",
  filename: o,
  showLineNumbers: t = !0,
  startLine: c = 1,
  highlightLines: p = [],
  wrap: s = !1,
  maxHeight: v = 480,
  copyable: g = !0,
  onCopy: N,
  onCopyError: w,
  className: _,
  ...j
}) {
  const [l, f] = q("idle"), h = E(void 0);
  I(() => () => clearTimeout(h.current), []);
  const d = i.replace(/\r\n?/g, `
`), u = M(() => {
    try {
      const e = Nr(d, r);
      if (e) return wr(e);
    } catch {
    }
    return d.split(`
`).map((e) => [{ value: e, className: "" }]);
  }, [d, r]), y = Number.isFinite(c) ? Math.max(1, Math.floor(c)) : 1, C = async () => {
    try {
      await navigator.clipboard.writeText(i), f("copied");
    } catch (e) {
      const a = e instanceof Error ? e : new Error("复制失败，请手动选择代码。");
      f("error"), w?.(a);
      return;
    } finally {
      clearTimeout(h.current), h.current = setTimeout(() => f("idle"), 2500);
    }
    N?.(i);
  };
  return /* @__PURE__ */ n("div", { ...j, className: gr("yk-code-block", s && "yk-code-block--wrap", _), children: [
    /* @__PURE__ */ n("div", { className: "yk-code-block__toolbar", children: [
      /* @__PURE__ */ m("span", { children: o ?? r }),
      o && /* @__PURE__ */ m("span", { className: "yk-code-block__language", children: r }),
      g && /* @__PURE__ */ n("button", { type: "button", onClick: C, "aria-label": "复制代码", children: [
        l === "copied" && /* @__PURE__ */ m(vr, { name: "check" }),
        l === "copied" ? "已复制" : l === "error" ? "复制失败，请手动复制" : "复制代码"
      ] })
    ] }),
    /* @__PURE__ */ m(
      "pre",
      {
        className: "yk-code-block__pre",
        tabIndex: 0,
        "aria-label": o ? `${o} 代码` : `${r} 代码`,
        style: { maxHeight: v },
        children: /* @__PURE__ */ m("code", { children: u.map((e, a) => /* @__PURE__ */ n(
          "span",
          {
            className: "yk-code-block__line",
            "data-highlighted": p.includes(y + a) || void 0,
            children: [
              t && /* @__PURE__ */ m("span", { className: "yk-code-block__number", "aria-hidden": "true", children: y + a }),
              /* @__PURE__ */ n("span", { className: "yk-code-block__text", children: [
                e.map((k, T) => /* @__PURE__ */ m("span", { className: k.className || void 0, children: k.value }, T)),
                a < u.length - 1 ? `
` : ""
              ] })
            ]
          },
          a
        )) })
      }
    ),
    /* @__PURE__ */ m("span", { role: "status", className: "yk-sr-only", children: l === "copied" ? "代码已复制" : l === "error" ? "无法访问剪贴板，请手动复制代码" : "" })
  ] });
}
export {
  yo as CodeBlock
};

