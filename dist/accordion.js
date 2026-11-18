import { jsx as n, jsxs as s } from "react/jsx-runtime";
import { forwardRef as i } from "react";
import * as e from "@radix-ui/react-accordion";
import { c as t, I as m } from "./chunks/tokens-BqLZJxpY.js";
import './assets/accordion.css';const u = i(function({ className: o, ...c }, r) {
  return /* @__PURE__ */ n(e.Root, { ...c, ref: r, className: t("yk-accordion", o) });
}), A = i(function({ className: o, ...c }, r) {
  return /* @__PURE__ */ n(e.Item, { ...c, ref: r, className: t("yk-accordion-item", o) });
}), h = i(function({ children: o, className: c, ...r }, d) {
  return /* @__PURE__ */ n(e.Header, { className: "yk-accordion-header", children: /* @__PURE__ */ s(e.Trigger, { ...r, ref: d, className: t("yk-accordion-trigger", c), children: [
    o,
    /* @__PURE__ */ n(m, { name: "chevron" })
  ] }) });
}), k = i(function({ className: o, children: c, ...r }, d) {
  return /* @__PURE__ */ n(e.Content, { ...r, ref: d, className: t("yk-accordion-content", o), children: /* @__PURE__ */ n("div", { children: c }) });
});
export {
  u as Accordion,
  k as AccordionContent,
  A as AccordionItem,
  h as AccordionTrigger
};

