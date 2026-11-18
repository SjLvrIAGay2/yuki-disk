import { jsxs as c, jsx as e } from "react/jsx-runtime";
import { useId as v } from "react";
import { c as m } from "./chunks/tokens-BqLZJxpY.js";
import './assets/progress.css';function b({
  value: s = 0,
  max: l = 100,
  label: r,
  showValue: o,
  size: u = "md",
  className: p,
  ...n
}) {
  const t = v(), d = Number.isFinite(l) && l > 0 ? l : 100, i = s === null ? null : Math.min(d, Math.max(0, Number.isFinite(s) ? s : 0)), a = i === null ? null : Math.round(i / d * 100);
  return /* @__PURE__ */ c("div", { className: m("yk-progress-field", p), children: [
    (r || o) && /* @__PURE__ */ c("div", { className: "yk-progress__label", children: [
      r && /* @__PURE__ */ e("span", { id: t, children: r }),
      o && /* @__PURE__ */ e("span", { children: a === null ? "…" : `${a}%` })
    ] }),
    /* @__PURE__ */ e(
      "div",
      {
        ...n,
        className: m("yk-progress", `yk-progress--${u}`),
        role: "progressbar",
        "aria-label": n["aria-label"] ?? (r ? void 0 : "进度"),
        "aria-labelledby": n["aria-labelledby"] ?? (r ? t : void 0),
        "aria-valuemin": 0,
        "aria-valuemax": d,
        "aria-valuenow": i ?? void 0,
        "aria-busy": i === null || void 0,
        children: /* @__PURE__ */ e(
          "div",
          {
            className: "yk-progress__fill",
            "data-indeterminate": a === null,
            style: { width: a === null ? "35%" : `${a}%` }
          }
        )
      }
    )
  ] });
}
export {
  b as Progress
};

