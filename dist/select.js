import { jsxs as r, jsx as e } from "react/jsx-runtime";
import { forwardRef as N, useId as x } from "react";
import * as l from "@radix-ui/react-select";
import { getNonce as S } from "get-nonce";
import { I as c, c as j } from "./chunks/tokens-BqLZJxpY.js";
import { useTheme as w } from "./theme.js";
import './assets/select.css';const $ = N(function({
  options: o,
  label: n,
  placeholder: d = "请选择",
  error: s,
  description: m,
  id: h,
  className: f,
  portalContainer: p,
  "aria-label": u,
  "aria-describedby": y,
  ...g
}, v) {
  const k = x(), a = h ?? k, i = s || m, { theme: b, tokens: I } = w();
  return /* @__PURE__ */ r("div", { className: j("yk-select-field", f), children: [
    n && /* @__PURE__ */ e("label", { htmlFor: a, children: n }),
    /* @__PURE__ */ r(l.Root, { ...g, children: [
      /* @__PURE__ */ r(
        l.Trigger,
        {
          ref: v,
          id: a,
          className: "yk-select-trigger",
          "aria-label": u,
          "aria-invalid": !!s || void 0,
          "aria-describedby": [y, i ? `${a}-message` : null].filter(Boolean).join(" ") || void 0,
          children: [
            /* @__PURE__ */ e(l.Value, { placeholder: d }),
            /* @__PURE__ */ e(l.Icon, { asChild: !0, children: /* @__PURE__ */ e(c, { name: "chevron" }) })
          ]
        }
      ),
      /* @__PURE__ */ e(l.Portal, { container: p, children: /* @__PURE__ */ r(
        l.Content,
        {
          className: "yk-theme yk-select-content",
          "data-theme": b,
          style: I,
          position: "popper",
          sideOffset: 5,
          children: [
            /* @__PURE__ */ e(l.ScrollUpButton, { className: "yk-select-scroll", children: /* @__PURE__ */ e(c, { name: "chevron", style: { transform: "rotate(180deg)" } }) }),
            /* @__PURE__ */ e(l.Viewport, { nonce: S(), children: o.map((t) => /* @__PURE__ */ r(
              l.Item,
              {
                value: t.value,
                disabled: t.disabled,
                className: "yk-select-item",
                children: [
                  /* @__PURE__ */ e(l.ItemText, { children: t.label }),
                  /* @__PURE__ */ e(l.ItemIndicator, { children: /* @__PURE__ */ e(c, { name: "check" }) })
                ]
              },
              t.value
            )) }),
            /* @__PURE__ */ e(l.ScrollDownButton, { className: "yk-select-scroll", children: /* @__PURE__ */ e(c, { name: "chevron" }) })
          ]
        }
      ) })
    ] }),
    i && /* @__PURE__ */ e("p", { className: "yk-select-message", id: `${a}-message`, "data-error": !!s, children: i })
  ] });
});
export {
  $ as Select
};

