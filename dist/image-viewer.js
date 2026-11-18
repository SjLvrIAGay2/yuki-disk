import { jsx as s } from "react/jsx-runtime";
import { forwardRef as t } from "react";
import { u as c } from "./chunks/media-wH8MaEKZ.js";
import { c as f } from "./chunks/tokens-BqLZJxpY.js";
import './assets/image-viewer.css';const d = t(function({ src: r, alt: e, className: o, ...m }, i) {
  const a = c(r);
  return /* @__PURE__ */ s("img", { ...m, ref: i, src: a || void 0, alt: e, className: f("yk-image-viewer", o) });
});
export {
  d as ImageViewer
};

