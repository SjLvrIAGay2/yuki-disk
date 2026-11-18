import { jsx as t, jsxs as r } from "react/jsx-runtime";
import * as e from "@radix-ui/react-tooltip";
import { useTheme as a } from "./theme.js";
import './assets/tooltip.css';function d({
  children: i,
  content: s,
  side: m = "top",
  delayDuration: o = 250,
  ...l
}) {
  const { theme: n, tokens: h } = a();
  return /* @__PURE__ */ t(e.Provider, { delayDuration: o, children: /* @__PURE__ */ r(e.Root, { ...l, delayDuration: o, children: [
    /* @__PURE__ */ t(e.Trigger, { asChild: !0, children: i }),
    /* @__PURE__ */ t(e.Portal, { children: /* @__PURE__ */ r(
      e.Content,
      {
        side: m,
        sideOffset: 7,
        className: "yk-theme yk-tooltip",
        "data-theme": n,
        style: h,
        children: [
          s,
          /* @__PURE__ */ t(e.Arrow, { className: "yk-tooltip__arrow" })
        ]
      }
    ) })
  ] }) });
}
export {
  d as Tooltip
};

