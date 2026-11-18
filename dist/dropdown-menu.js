import { jsxs as c, jsx as n } from "react/jsx-runtime";
import { forwardRef as d } from "react";
import * as r from "@radix-ui/react-dropdown-menu";
import { c as m } from "./chunks/tokens-BqLZJxpY.js";
import { useTheme as p } from "./theme.js";
import './assets/dropdown-menu.css';function w({
  trigger: t,
  children: e,
  align: a = "end",
  className: o,
  ...s
}) {
  const { theme: i, tokens: u } = p();
  return /* @__PURE__ */ c(r.Root, { ...s, children: [
    /* @__PURE__ */ n(r.Trigger, { asChild: !0, children: t }),
    /* @__PURE__ */ n(r.Portal, { children: /* @__PURE__ */ n(
      r.Content,
      {
        align: a,
        sideOffset: 6,
        className: m("yk-theme", "yk-menu", o),
        "data-theme": i,
        style: u,
        children: e
      }
    ) })
  ] });
}
const D = d(
  function({ destructive: e, icon: a, shortcut: o, children: s, className: i, ...u }, l) {
    return /* @__PURE__ */ c(
      r.Item,
      {
        ...u,
        ref: l,
        className: m("yk-menu-item", e && "yk-menu-item--danger", i),
        children: [
          a,
          /* @__PURE__ */ n("span", { children: s }),
          o && /* @__PURE__ */ n("span", { className: "yk-menu-shortcut", children: o })
        ]
      }
    );
  }
);
function M({
  className: t,
  ...e
}) {
  return /* @__PURE__ */ n(r.Label, { ...e, className: m("yk-menu-label", t) });
}
function N({
  className: t,
  ...e
}) {
  return /* @__PURE__ */ n(r.Separator, { ...e, className: m("yk-menu-separator", t) });
}
export {
  w as DropdownMenu,
  D as DropdownMenuItem,
  M as DropdownMenuLabel,
  N as DropdownMenuSeparator
};

