import { jsx as t, jsxs as n, Fragment as C } from "react/jsx-runtime";
import { createContext as P, useState as u, useRef as b, useId as g, useEffect as I, useCallback as m, useContext as $ } from "react";
import { createPortal as j } from "react-dom";
import * as o from "@radix-ui/react-toast";
import { I as f, c as D } from "./chunks/tokens-BqLZJxpY.js";
import { useTheme as E } from "./theme.js";
import './assets/toast.css';const p = P(null);
function S({
  children: r,
  duration: h = 5e3,
  label: i = "通知",
  closeLabel: k = "关闭通知"
}) {
  const [y, c] = u([]), [v, _] = u(!1), x = b(0), d = g(), { theme: N, tokens: T } = E();
  I(() => {
    _(!0);
  }, []);
  const w = m(
    (e) => {
      const s = `${d}-${++x.current}`;
      return c((a) => [...a.slice(-4), { ...e, id: s }]), s;
    },
    [d]
  ), l = m(
    (e) => c((s) => s.filter((a) => a.id !== e)),
    []
  );
  return /* @__PURE__ */ t(p.Provider, { value: { toast: w, dismiss: l }, children: /* @__PURE__ */ n(o.Provider, { duration: h, swipeDirection: "right", label: i, children: [
    r,
    v && j(
      /* @__PURE__ */ n(C, { children: [
        /* @__PURE__ */ t(
          o.Viewport,
          {
            className: "yk-theme yk-toast-viewport",
            "data-theme": N,
            style: T,
            label: `${i} (F8)`
          }
        ),
        y.map((e) => /* @__PURE__ */ n(
          o.Root,
          {
            className: D("yk-toast", `yk-toast--${e.variant ?? "info"}`),
            duration: e.duration,
            onOpenChange: (s) => {
              s || l(e.id);
            },
            children: [
              /* @__PURE__ */ t("span", { className: "yk-toast__icon", children: /* @__PURE__ */ t(
                f,
                {
                  name: e.variant === "success" ? "check" : e.variant === "danger" ? "warning" : "info"
                }
              ) }),
              /* @__PURE__ */ n("div", { className: "yk-toast__body", children: [
                /* @__PURE__ */ t(o.Title, { className: "yk-toast__title", children: e.title }),
                e.description && /* @__PURE__ */ t(o.Description, { className: "yk-toast__description", children: e.description })
              ] }),
              /* @__PURE__ */ t(o.Close, { className: "yk-toast__close", "aria-label": k, children: /* @__PURE__ */ t(f, { name: "close" }) })
            ]
          },
          e.id
        ))
      ] }),
      document.body
    )
  ] }) });
}
function V() {
  const r = $(p);
  if (!r) throw new Error("useToast must be used inside ToastProvider.");
  return r;
}
export {
  S as ToastProvider,
  V as useToast
};

