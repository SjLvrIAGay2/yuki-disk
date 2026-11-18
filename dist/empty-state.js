import { jsxs as i, jsx as e } from "react/jsx-runtime";
import { c as n, I as l } from "./chunks/tokens-BqLZJxpY.js";
import './assets/empty-state.css';function o({
  title: c,
  description: s,
  icon: m = /* @__PURE__ */ e(l, { name: "search" }),
  action: a,
  className: t,
  ...r
}) {
  return /* @__PURE__ */ i("div", { ...r, className: n("yk-empty", t), children: [
    /* @__PURE__ */ e("span", { className: "yk-empty__icon", children: m }),
    /* @__PURE__ */ e("h3", { className: "yk-empty__title", children: c }),
    s && /* @__PURE__ */ e("p", { className: "yk-empty__description", children: s }),
    a && /* @__PURE__ */ e("div", { className: "yk-empty__action", children: a })
  ] });
}
export {
  o as EmptyState
};

