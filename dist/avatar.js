import { jsxs as u, jsx as t } from "react/jsx-runtime";
import { useState as v } from "react";
import { c as e } from "./chunks/tokens-BqLZJxpY.js";
import './assets/avatar.css';function x({
  src: a,
  alt: r,
  fallback: n,
  size: o = "md",
  status: i,
  className: s,
  ...m
}) {
  const [l, p] = v(), c = n ?? r.trim().split(/\s+/).map((d) => d[0]).slice(0, 2).join("").toUpperCase();
  return /* @__PURE__ */ u(
    "span",
    {
      ...m,
      className: e("yk-avatar", `yk-avatar--${o}`, s),
      role: "img",
      "aria-label": `${r}${i ? ` (${i})` : ""}`,
      children: [
        a && l !== a ? /* @__PURE__ */ t("img", { src: a, alt: "", onError: () => p(a) }) : /* @__PURE__ */ t("span", { "aria-hidden": "true", children: c || "?" }),
        i && /* @__PURE__ */ t("span", { className: "yk-avatar__status", "data-status": i, "aria-hidden": "true" })
      ]
    }
  );
}
function y({ className: a, ...r }) {
  return /* @__PURE__ */ t("div", { ...r, className: e("yk-avatar-group", a) });
}
export {
  x as Avatar,
  y as AvatarGroup
};

