import { jsx as r } from "react/jsx-runtime";
import { forwardRef as o } from "react";
import { c as l } from "./chunks/tokens-BqLZJxpY.js";
import './assets/table.css';const u = o(
  function({ className: t, ...n }, a) {
    return /* @__PURE__ */ r("div", { className: "yk-table-scroll", tabIndex: 0, role: "region", "aria-label": "表格滚动区域", children: /* @__PURE__ */ r("table", { ...n, ref: a, className: l("yk-table", t) }) });
  }
);
function f(e) {
  return /* @__PURE__ */ r("thead", { ...e });
}
function d(e) {
  return /* @__PURE__ */ r("tbody", { ...e });
}
function s(e) {
  return /* @__PURE__ */ r("tr", { ...e });
}
function m(e) {
  return /* @__PURE__ */ r("th", { scope: "col", ...e });
}
function T(e) {
  return /* @__PURE__ */ r("td", { ...e });
}
function p(e) {
  return /* @__PURE__ */ r("caption", { ...e });
}
export {
  u as Table,
  d as TableBody,
  p as TableCaption,
  T as TableCell,
  m as TableHead,
  f as TableHeader,
  s as TableRow
};

