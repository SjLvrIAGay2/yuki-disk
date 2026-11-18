import { jsxs as n, jsx as e } from "react/jsx-runtime";
import { forwardRef as y, useState as N } from "react";
import * as a from "@radix-ui/react-slider";
import { c as g } from "./chunks/tokens-BqLZJxpY.js";
import './assets/slider.css';const x = y(function({
  label: s,
  showValue: c,
  formatValue: m = String,
  thumbLabels: d,
  value: o,
  defaultValue: h = [50],
  onValueChange: _,
  className: f,
  ...t
}, k) {
  const [p, u] = N(h), r = o ?? p;
  return /* @__PURE__ */ n("div", { className: g("yk-slider-field", f), children: [
    (s || c) && /* @__PURE__ */ n("div", { className: "yk-slider__label", children: [
      /* @__PURE__ */ e("span", { children: s }),
      c && /* @__PURE__ */ e("span", { children: r.map(m).join(" – ") })
    ] }),
    /* @__PURE__ */ n(
      a.Root,
      {
        ...t,
        ref: k,
        className: "yk-slider",
        value: r,
        onValueChange: (i) => {
          u(i), _?.(i);
        },
        children: [
          /* @__PURE__ */ e(a.Track, { className: "yk-slider__track", children: /* @__PURE__ */ e(a.Range, { className: "yk-slider__range" }) }),
          r.map((i, l) => /* @__PURE__ */ e(
            a.Thumb,
            {
              className: "yk-slider__thumb",
              "aria-label": d?.[l] ?? `${s ?? t["aria-label"] ?? "数值"}${r.length > 1 ? ` ${l + 1}` : ""}`
            },
            l
          ))
        ]
      }
    )
  ] });
});
export {
  x as Slider
};

