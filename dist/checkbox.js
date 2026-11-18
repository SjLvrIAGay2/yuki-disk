import { jsxs as a, jsx as e } from "react/jsx-runtime";
import { forwardRef as m, useId as h } from "react";
import * as d from "@radix-ui/react-checkbox";
import { I as b, c as k } from "./chunks/tokens-BqLZJxpY.js";
import './assets/checkbox.css';const N = m(function({ label: r, description: c, id: s, className: l, ...o }, n) {
  const t = h(), i = s ?? t;
  return /* @__PURE__ */ a("div", { className: k("yk-checkbox-field", l), "data-disabled": o.disabled, children: [
    /* @__PURE__ */ e(
      d.Root,
      {
        ...o,
        ref: n,
        id: i,
        className: "yk-checkbox",
        "aria-describedby": [o["aria-describedby"], c ? `${i}-description` : null].filter(Boolean).join(" ") || void 0,
        children: /* @__PURE__ */ a(d.Indicator, { className: "yk-checkbox__indicator", children: [
          /* @__PURE__ */ e(b, { name: "check" }),
          /* @__PURE__ */ e("span", {})
        ] })
      }
    ),
    (r || c) && /* @__PURE__ */ a("div", { children: [
      /* @__PURE__ */ e("label", { htmlFor: i, className: "yk-checkbox-label", children: r }),
      c && /* @__PURE__ */ e("p", { className: "yk-checkbox-description", id: `${i}-description`, children: c })
    ] })
  ] });
});
export {
  N as Checkbox
};

