import { jsxs as s, jsx as d } from "react/jsx-runtime";
import { forwardRef as f, useId as u } from "react";
import { c as y } from "./chunks/tokens-BqLZJxpY.js";
import './assets/textarea.css';const v = f(function({ label: i, description: l, error: a, id: n, className: c, "aria-describedby": o, ...r }, m) {
  const x = u(), e = n ?? x, t = a || l;
  return /* @__PURE__ */ s("div", { className: y("yk-textarea-field", c), children: [
    i && /* @__PURE__ */ s("label", { htmlFor: e, className: "yk-textarea-label", children: [
      i,
      r.required && " *"
    ] }),
    /* @__PURE__ */ d(
      "textarea",
      {
        rows: 4,
        ...r,
        id: e,
        ref: m,
        className: "yk-textarea",
        "aria-invalid": a ? !0 : r["aria-invalid"],
        "aria-describedby": [o, t ? `${e}-message` : null].filter(Boolean).join(" ") || void 0
      }
    ),
    t && /* @__PURE__ */ d("p", { id: `${e}-message`, className: "yk-textarea-message", "data-error": !!a, children: t })
  ] });
});
export {
  v as Textarea
};

