import { jsxs as i, jsx as o } from "react/jsx-runtime";
import { useId as _, useRef as p } from "react";
import * as e from "@radix-ui/react-dialog";
import { I as v, c as g } from "./chunks/tokens-BqLZJxpY.js";
import { useTheme as C } from "./theme.js";
import './assets/dialog.css';function I({
  trigger: l,
  title: n,
  description: s,
  footer: t,
  children: r,
  className: d,
  size: m = "md",
  closeLabel: u = "关闭对话框",
  ...y
}) {
  const { theme: h, tokens: f } = C(), c = _(), a = p(null);
  return /* @__PURE__ */ i(e.Root, { ...y, children: [
    l && /* @__PURE__ */ o(e.Trigger, { asChild: !0, children: l }),
    /* @__PURE__ */ i(e.Portal, { children: [
      /* @__PURE__ */ o(e.Overlay, { className: "yk-dialog-overlay" }),
      /* @__PURE__ */ i(
        e.Content,
        {
          className: g("yk-theme", "yk-dialog", `yk-dialog--${m}`, d),
          "data-theme": h,
          style: f,
          "aria-describedby": s ? c : void 0,
          onOpenAutoFocus: () => {
            a.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
          },
          onCloseAutoFocus: (k) => {
            !l && a.current?.isConnected && (k.preventDefault(), a.current.focus());
          },
          children: [
            /* @__PURE__ */ o(e.Title, { className: "yk-dialog__title", children: n }),
            s && /* @__PURE__ */ o(e.Description, { id: c, className: "yk-dialog__description", children: s }),
            /* @__PURE__ */ o("div", { className: "yk-dialog__body", children: r }),
            t && /* @__PURE__ */ o("div", { className: "yk-dialog__footer", children: t }),
            /* @__PURE__ */ o(e.Close, { className: "yk-dialog__close", "aria-label": u, children: /* @__PURE__ */ o(v, { name: "close" }) })
          ]
        }
      )
    ] })
  ] });
}
const T = e.Close;
export {
  I as Dialog,
  T as DialogClose
};

