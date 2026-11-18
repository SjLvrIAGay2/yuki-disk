import { jsxs as c, jsx as p } from "react/jsx-runtime";
import { forwardRef as y } from "react";
import { c as d } from "./chunks/tokens-BqLZJxpY.js";
import './assets/button.css';const j = y(function({
  variant: o = "primary",
  size: r = "md",
  loading: t = !1,
  fullWidth: n,
  startIcon: s,
  endIcon: u,
  disabled: a,
  className: e,
  children: i,
  type: m = "button",
  ...b
}, f) {
  return /* @__PURE__ */ c(
    "button",
    {
      ...b,
      ref: f,
      type: m,
      disabled: a || t,
      "aria-busy": t || void 0,
      className: d(
        "yk-button",
        `yk-button--${o}`,
        `yk-button--${r}`,
        n && "yk-button--full",
        e
      ),
      children: [
        t ? /* @__PURE__ */ p("span", { className: "yk-button__spinner", "aria-hidden": "true" }) : s,
        i,
        u
      ]
    }
  );
});
export {
  j as Button
};

