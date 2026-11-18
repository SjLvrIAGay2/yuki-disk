import { jsx as a, jsxs as s } from "react/jsx-runtime";
import { forwardRef as n, useId as u } from "react";
import * as l from "@radix-ui/react-radio-group";
import { c as f } from "./chunks/tokens-BqLZJxpY.js";
import './assets/radio-group.css';const k = n(function({ options: c, label: m, className: o, orientation: t = "vertical", ...i }, b) {
  const e = u();
  return /* @__PURE__ */ a(
    l.Root,
    {
      ...i,
      ref: b,
      orientation: t,
      "aria-label": i["aria-label"] ?? m,
      className: f("yk-radio-group", o),
      children: c.map((d, r) => /* @__PURE__ */ s(
        "div",
        {
          className: "yk-radio-field",
          "data-disabled": i.disabled || d.disabled,
          children: [
            /* @__PURE__ */ a(
              l.Item,
              {
                className: "yk-radio",
                id: `${e}-${r}`,
                value: d.value,
                disabled: d.disabled || i.disabled,
                "aria-describedby": d.description ? `${e}-${r}-description` : void 0,
                children: /* @__PURE__ */ a(l.Indicator, { className: "yk-radio__indicator" })
              }
            ),
            /* @__PURE__ */ s("div", { children: [
              /* @__PURE__ */ a("label", { htmlFor: `${e}-${r}`, children: d.label }),
              d.description && /* @__PURE__ */ a("p", { id: `${e}-${r}-description`, children: d.description })
            ] })
          ]
        },
        d.value
      ))
    }
  );
});
export {
  k as RadioGroup
};

