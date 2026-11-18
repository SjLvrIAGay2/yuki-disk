import { jsxs as d, jsx as c } from "react/jsx-runtime";
import { forwardRef as b, useId as f } from "react";
import * as r from "@radix-ui/react-switch";
import { c as a } from "./chunks/tokens-BqLZJxpY.js";
import './assets/switch.css';const N = b(function({ label: t, description: i, size: l = "md", id: o, className: m, ...s }, h) {
  const n = f(), e = o ?? n;
  return /* @__PURE__ */ d("div", { className: a("yk-switch-field", m), "data-disabled": s.disabled, children: [
    /* @__PURE__ */ c(
      r.Root,
      {
        ...s,
        id: e,
        ref: h,
        className: a("yk-switch", `yk-switch--${l}`),
        "aria-describedby": [s["aria-describedby"], i ? `${e}-description` : null].filter(Boolean).join(" ") || void 0,
        children: /* @__PURE__ */ c(r.Thumb, { className: "yk-switch__thumb" })
      }
    ),
    (t || i) && /* @__PURE__ */ d("div", { children: [
      /* @__PURE__ */ c("label", { htmlFor: e, className: "yk-switch-label", children: t }),
      i && /* @__PURE__ */ c("p", { id: `${e}-description`, className: "yk-switch-description", children: i })
    ] })
  ] });
});
export {
  N as Switch
};

