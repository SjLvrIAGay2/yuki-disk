import { jsx as o } from "react/jsx-runtime";
import { createContext as f, useContext as l, useState as v, useEffect as p } from "react";
import { c as x } from "./chunks/tokens-BqLZJxpY.js";
const a = f({
  theme: "light"
});
function C({
  theme: e = "light",
  tokens: s,
  className: m,
  style: c,
  children: i,
  ...d
}) {
  const [h, u] = v(!1);
  p(() => {
    if (e !== "system") return;
    const t = window.matchMedia("(prefers-color-scheme: dark)"), r = () => u(t.matches);
    return r(), t.addEventListener("change", r), () => t.removeEventListener("change", r);
  }, [e]);
  const n = e === "system" ? h ? "dark" : "light" : e;
  return /* @__PURE__ */ o(a.Provider, { value: { theme: n, tokens: s }, children: /* @__PURE__ */ o(
    "div",
    {
      ...d,
      className: x("yk-theme", m),
      "data-theme": n,
      style: { ...s, ...c },
      children: i
    }
  ) });
}
function E() {
  return l(a);
}
export {
  C as ThemeProvider,
  E as useTheme
};

