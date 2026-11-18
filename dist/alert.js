import { jsxs as r, jsx as l } from "react/jsx-runtime";
import { I as c, c as d } from "./chunks/tokens-BqLZJxpY.js";
import './assets/alert.css';function y({
  title: t,
  variant: e = "info",
  onClose: s,
  closeLabel: n = "关闭提示",
  children: a,
  className: i,
  ...o
}) {
  return /* @__PURE__ */ r(
    "div",
    {
      role: e === "danger" ? "alert" : "status",
      ...o,
      className: d("yk-alert", `yk-alert--${e}`, i),
      children: [
        /* @__PURE__ */ l(
          c,
          {
            name: e === "success" ? "check" : e === "warning" || e === "danger" ? "warning" : "info"
          }
        ),
        /* @__PURE__ */ r("div", { className: "yk-alert__body", children: [
          /* @__PURE__ */ l("div", { className: "yk-alert__title", children: t }),
          a && /* @__PURE__ */ l("div", { className: "yk-alert__description", children: a })
        ] }),
        s && /* @__PURE__ */ l("button", { className: "yk-alert__close", type: "button", onClick: s, "aria-label": n, children: /* @__PURE__ */ l(c, { name: "close" }) })
      ]
    }
  );
}
export {
  y as Alert
};

