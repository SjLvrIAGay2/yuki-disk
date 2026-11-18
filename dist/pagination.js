import { jsxs as m, jsx as i } from "react/jsx-runtime";
import { I as p, c as h } from "./chunks/tokens-BqLZJxpY.js";
import './assets/pagination.css';function y({
  page: l,
  totalPages: s,
  onPageChange: n,
  disabled: o,
  className: u,
  ...b
}) {
  const r = Number.isFinite(s) ? Math.max(1, Math.floor(s)) : 1, a = Number.isFinite(l) ? Math.min(r, Math.max(1, Math.floor(l))) : 1, c = [.../* @__PURE__ */ new Set([1, a - 1, a, a + 1, r])].filter((t) => t > 0 && t <= r).sort((t, e) => t - e);
  return /* @__PURE__ */ m("nav", { "aria-label": "分页", ...b, className: h("yk-pagination", u), children: [
    /* @__PURE__ */ i(
      "button",
      {
        type: "button",
        "aria-label": "上一页",
        disabled: o || a === 1,
        onClick: () => n(a - 1),
        children: /* @__PURE__ */ i(p, { name: "chevron", style: { transform: "rotate(90deg)" } })
      }
    ),
    c.map((t, e) => /* @__PURE__ */ m("span", { className: "yk-pagination__item", children: [
      e > 0 && t - c[e - 1] > 1 && /* @__PURE__ */ i("span", { className: "yk-pagination__ellipsis", "aria-hidden": "true", children: "…" }),
      /* @__PURE__ */ i(
        "button",
        {
          type: "button",
          disabled: o,
          "aria-label": `第 ${t} 页`,
          "aria-current": t === a ? "page" : void 0,
          onClick: () => n(t),
          children: t
        }
      )
    ] }, t)),
    /* @__PURE__ */ i(
      "button",
      {
        type: "button",
        "aria-label": "下一页",
        disabled: o || a === r,
        onClick: () => n(a + 1),
        children: /* @__PURE__ */ i(p, { name: "chevron", style: { transform: "rotate(-90deg)" } })
      }
    )
  ] });
}
export {
  y as Pagination
};

