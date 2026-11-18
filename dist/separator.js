import { jsx as e } from "react/jsx-runtime";
import { c as p } from "./chunks/tokens-BqLZJxpY.js";
import './assets/separator.css';function n({
  orientation: r = "horizontal",
  decorative: o = !0,
  className: a,
  ...t
}) {
  return /* @__PURE__ */ e(
    "div",
    {
      ...t,
      role: o ? "none" : "separator",
      "aria-orientation": o ? void 0 : r,
      className: p("yk-separator", `yk-separator--${r}`, a)
    }
  );
}
export {
  n as Separator
};

