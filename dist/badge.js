import { jsxs as n, jsx as t } from "react/jsx-runtime";
import { c as o } from "./chunks/tokens-BqLZJxpY.js";
import './assets/badge.css';function m({ variant: a = "neutral", dot: e, className: r, children: s, ...d }) {
  return /* @__PURE__ */ n("span", { ...d, className: o("yk-badge", `yk-badge--${a}`, r), children: [
    e && /* @__PURE__ */ t("span", { className: "yk-badge__dot", "aria-hidden": "true" }),
    s
  ] });
}
export {
  m as Badge
};

