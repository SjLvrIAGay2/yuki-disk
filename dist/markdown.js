import { jsx as a } from "react/jsx-runtime";
import { useMemo as u } from "react";
import h from "react-markdown";
import k from "remark-gfm";
import { CodeBlock as x } from "./code-block.js";
import { c as y } from "./chunks/tokens-BqLZJxpY.js";
import './assets/markdown.css';function g() {
  return (p) => {
    const l = (t) => t.type === "text" ? t.value : t.children.map(
      (r) => r.type === "element" || r.type === "text" ? l(r) : ""
    ).join("");
    function c(t) {
      if (t.type === "element" && t.tagName === "li") {
        const r = (s) => {
          for (const n of s.children)
            if (n.type === "element") {
              if (n.tagName === "input" && n.properties.type === "checkbox") return n;
              if (n.tagName === "p") {
                const i = r(n);
                if (i) return i;
              }
            }
        }, m = r(t);
        m && (m.properties.ariaLabel = l(t).trim() || "任务状态");
      }
      for (const r of t.children) r.type === "element" && c(r);
    }
    c(p);
  };
}
function M({
  children: p,
  components: l,
  openLinksInNewTab: c = !0,
  codeLineNumbers: t = !1,
  allowImages: r = !0,
  className: m,
  ...s
}) {
  const n = u(
    () => ({
      pre({ node: i }) {
        const e = i?.children.find(
          (o) => o.type === "element" && o.tagName === "code"
        );
        if (!e || e.type !== "element") return null;
        const d = String(e.properties.className ?? "").match(/language-([\w+-]+)/)?.[1] ?? "text", f = e.children.map((o) => o.type === "text" ? o.value : "").join("").replace(/\n$/, "");
        return /* @__PURE__ */ a(x, { code: f, language: d, showLineNumbers: t });
      },
      a({ node: i, href: e, children: d, ...f }) {
        const o = !!e && /^(https?:)?\/\//i.test(e);
        return /* @__PURE__ */ a(
          "a",
          {
            ...f,
            href: e,
            target: o && c ? "_blank" : void 0,
            rel: o ? "noopener noreferrer" : void 0,
            children: d
          }
        );
      },
      img({ node: i, ...e }) {
        return r ? /* @__PURE__ */ a("img", { ...e, alt: e.alt ?? "", loading: "lazy" }) : /* @__PURE__ */ a("span", { children: e.alt || "图片" });
      },
      table({ node: i, ...e }) {
        return /* @__PURE__ */ a(
          "div",
          {
            className: "yk-markdown__table-scroll",
            tabIndex: 0,
            role: "region",
            "aria-label": "文档表格",
            children: /* @__PURE__ */ a("table", { ...e })
          }
        );
      },
      ...l
    }),
    [l, c, t, r]
  );
  return /* @__PURE__ */ a("div", { ...s, className: y("yk-markdown", m), children: /* @__PURE__ */ a(
    h,
    {
      remarkPlugins: [k],
      rehypePlugins: [g],
      skipHtml: !0,
      components: n,
      children: p
    }
  ) });
}
export {
  M as Markdown
};

