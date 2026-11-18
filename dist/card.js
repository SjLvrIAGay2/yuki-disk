import { jsx as t } from "react/jsx-runtime";
import { forwardRef as d } from "react";
import { c as e } from "./chunks/tokens-BqLZJxpY.js";
import './assets/card.css';const f = d(function({ variant: r = "outline", className: n, ...c }, o) {
  return /* @__PURE__ */ t("div", { ...c, ref: o, className: e("yk-card", `yk-card--${r}`, n) });
});
function m({ className: a, ...r }) {
  return /* @__PURE__ */ t("div", { ...r, className: e("yk-card__header", a) });
}
function _({ className: a, ...r }) {
  return /* @__PURE__ */ t("h3", { ...r, className: e("yk-card__title", a) });
}
function l({ className: a, ...r }) {
  return /* @__PURE__ */ t("p", { ...r, className: e("yk-card__description", a) });
}
function C({ className: a, ...r }) {
  return /* @__PURE__ */ t("div", { ...r, className: e("yk-card__content", a) });
}
function k({ className: a, ...r }) {
  return /* @__PURE__ */ t("div", { ...r, className: e("yk-card__footer", a) });
}
export {
  f as Card,
  C as CardContent,
  l as CardDescription,
  k as CardFooter,
  m as CardHeader,
  _ as CardTitle
};

