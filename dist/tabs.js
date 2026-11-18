import { jsx as n } from "react/jsx-runtime";
import { forwardRef as a } from "react";
import * as o from "@radix-ui/react-tabs";
import { c as e } from "./chunks/tokens-BqLZJxpY.js";
import './assets/tabs.css';const f = a(function({ variant: t = "line", className: s, ...r }, c) {
  return /* @__PURE__ */ n(
    o.Root,
    {
      ...r,
      ref: c,
      className: e("yk-tabs", `yk-tabs--${t}`, s)
    }
  );
}), g = a(
  function({ className: t, ...s }, r) {
    return /* @__PURE__ */ n(o.List, { ...s, ref: r, className: e("yk-tabs-list", t) });
  }
), u = a(function({ className: t, ...s }, r) {
  return /* @__PURE__ */ n(o.Trigger, { ...s, ref: r, className: e("yk-tabs-trigger", t) });
}), l = a(function({ className: t, ...s }, r) {
  return /* @__PURE__ */ n(o.Content, { ...s, ref: r, className: e("yk-tabs-content", t) });
});
export {
  f as Tabs,
  l as TabsContent,
  g as TabsList,
  u as TabsTrigger
};

