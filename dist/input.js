import { jsxs as d, jsx as e } from "react/jsx-runtime";
import { forwardRef as f, useId as b } from "react";
import { c as r } from "./chunks/tokens-BqLZJxpY.js";
import './assets/input.css';const x = f(function({
  label: t,
  description: u,
  error: a,
  startIcon: l,
  endIcon: c,
  size: m = "md",
  id: p,
  className: o,
  "aria-describedby": y,
  ...i
}, h) {
  const k = b(), n = p ?? k, s = a || u;
  return /* @__PURE__ */ d("div", { className: r("yk-input-field", o), hidden: i.hidden, children: [
    t && /* @__PURE__ */ d("label", { className: "yk-input-label", htmlFor: n, children: [
      t,
      i.required && /* @__PURE__ */ e("span", { "aria-hidden": "true", children: " *" })
    ] }),
    /* @__PURE__ */ d(
      "div",
      {
        className: r(
          "yk-input-wrap",
          `yk-input-wrap--${m}`,
          !!a && "yk-input-wrap--error",
          i.disabled && "yk-input-wrap--disabled"
        ),
        children: [
          l && /* @__PURE__ */ e("span", { className: "yk-input-icon", children: l }),
          /* @__PURE__ */ e(
            "input",
            {
              ...i,
              ref: h,
              id: n,
              className: "yk-input",
              "aria-invalid": a ? !0 : i["aria-invalid"],
              "aria-describedby": [y, s ? `${n}-message` : null].filter(Boolean).join(" ") || void 0
            }
          ),
          c && /* @__PURE__ */ e("span", { className: "yk-input-icon", children: c })
        ]
      }
    ),
    s && /* @__PURE__ */ e(
      "p",
      {
        id: `${n}-message`,
        className: r("yk-input-message", !!a && "yk-input-message--error"),
        children: s
      }
    )
  ] });
});
export {
  x as Input
};

