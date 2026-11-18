import { jsx as e } from "react/jsx-runtime";
import './assets/icon.css';const o = {
  folder: "M3 7h6l2 2h10v11H3z M3 7V4h6l2 3",
  image: "M3 4h18v16H3z M3 17l6-6 4 4 3-3 5 5 M16 8h.01",
  text: "M5 3h10l4 4v14H5z M8 10h8 M8 14h8 M8 18h5",
  code: "M8 7l-5 5 5 5 M16 7l5 5-5 5 M14 4l-4 16",
  pdf: "M5 3h10l4 4v14H5z M14 3v5h5 M8 12h8 M8 16h5",
  audio: "M9 17V5l11-2v12 M9 8l11-2 M9 17a3 3 0 1 1-3-3h3 M20 15a3 3 0 1 1-3-3h3",
  video: "M3 4h18v16H3z M10 8l6 4-6 4z",
  archive: "M5 3h14v18H5z M10 3h4 M10 6h4 M10 9h4 M11 13h3v4h-3z",
  document: "M5 3h10l4 4v14H5z M8 10l2 7 2-5 2 5 2-7",
  spreadsheet: "M4 3h16v18H4z M4 9h16 M4 15h16 M10 9v12",
  presentation: "M3 4h18v13H3z M12 17v4 M8 21h8 M8 8v5 M12 10v3 M16 7v6",
  file: "M5 3h10l4 4v14H5z M14 3v5h5",
  menu: "M4 6h16 M4 12h16 M4 18h16",
  close: "M6 6l12 12 M18 6L6 18",
  download: "M12 3v12 M7 10l5 5 5-5 M4 16v5h16v-5"
}, r = Object.keys(o);
function n({ name: v, size: h = 24, label: M, className: l = "", ...i }) {
  return /* @__PURE__ */ e(
    "svg",
    {
      ...i,
      className: `yk-icon ${l}`.trim(),
      viewBox: "0 0 24 24",
      width: h,
      height: h,
      fill: "none",
      stroke: "currentColor",
      strokeWidth: 1.5,
      strokeLinecap: "round",
      strokeLinejoin: "round",
      role: M ? "img" : void 0,
      "aria-label": M,
      "aria-hidden": M ? void 0 : !0,
      focusable: "false",
      children: /* @__PURE__ */ e("path", { d: o[v] })
    }
  );
}
export {
  n as Icon,
  r as iconNames
};

