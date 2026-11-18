import { jsx as s } from "react/jsx-runtime";
import { c } from "./chunks/tokens-BqLZJxpY.js";
import './assets/skeleton.css';function k({
  width: e,
  height: r = 16,
  circle: t,
  className: o,
  style: i,
  ...n
}) {
  return /* @__PURE__ */ s(
    "div",
    {
      "aria-hidden": "true",
      ...n,
      className: c("yk-skeleton", t && "yk-skeleton--circle", o),
      style: { width: e, height: r, ...i }
    }
  );
}
export {
  k as Skeleton
};

