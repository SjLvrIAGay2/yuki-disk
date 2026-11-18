import { jsx as r, jsxs as n, Fragment as t } from "react/jsx-runtime";
import '../assets/tokens.css';function h(...e) {
  return e.filter(Boolean).join(" ");
}
function a({
  name: e,
  ...c
}) {
  return /* @__PURE__ */ r(
    "svg",
    {
      width: "16",
      height: "16",
      viewBox: "0 0 24 24",
      fill: "none",
      stroke: "currentColor",
      strokeWidth: "1.8",
      strokeLinecap: "round",
      strokeLinejoin: "round",
      "aria-hidden": "true",
      ...c,
      children: {
        check: /* @__PURE__ */ r("path", { d: "m5 12 4 4L19 6" }),
        chevron: /* @__PURE__ */ r("path", { d: "m6 9 6 6 6-6" }),
        close: /* @__PURE__ */ r("path", { d: "m6 6 12 12M6 18 18 6" }),
        info: /* @__PURE__ */ n(t, { children: [
          /* @__PURE__ */ r("circle", { cx: "12", cy: "12", r: "9" }),
          /* @__PURE__ */ r("path", { d: "M12 11v6m0-10v.1" })
        ] }),
        warning: /* @__PURE__ */ n(t, { children: [
          /* @__PURE__ */ r("path", { d: "m12 3 10 18H2L12 3Z" }),
          /* @__PURE__ */ r("path", { d: "M12 9v5m0 3v.1" })
        ] }),
        search: /* @__PURE__ */ n(t, { children: [
          /* @__PURE__ */ r("circle", { cx: "10.5", cy: "10.5", r: "7" }),
          /* @__PURE__ */ r("path", { d: "m16 16 5 5" })
        ] })
      }[e]
    }
  );
}
export {
  a as I,
  h as c
};

