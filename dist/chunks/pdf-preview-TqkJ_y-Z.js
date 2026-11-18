import { jsx as t, jsxs as b, Fragment as L } from "react/jsx-runtime";
import { useState as p, useRef as _, useEffect as M } from "react";
import { PDFWorker as S, getDocument as j, TextLayer as q } from "pdfjs-dist";
import { Button as D } from "../button.js";
import { Input as I } from "../input.js";
function Z({ data: w, filename: n, options: s, onLoad: d, onError: u }) {
  const [c, h] = p(), [a, v] = p(1), [r, y] = p(1), [F, l] = p(""), [k, x] = p(""), [i, m] = p(!1), e = _(void 0), f = _({ onLoad: d, onError: u });
  f.current = { onLoad: d, onError: u };
  const g = s?.password, N = s?.cMapUrl, E = s?.standardFontDataUrl, R = s?.wasmUrl;
  return M(() => {
    let o = !0;
    h(void 0), v(1), y(1), l(""), m(!1);
    const $ = new Worker(new URL(
      /* @vite-ignore */
      "" + new URL("../assets/pdf-worker-DmMLNyE6.js", import.meta.url).href,
      import.meta.url
    ), { type: "module" }), U = S.create({ port: $ }), C = j({
      data: new Uint8Array(w.slice(0)),
      worker: U,
      password: g,
      cMapUrl: N,
      cMapPacked: !0,
      standardFontDataUrl: E,
      wasmUrl: R,
      useSystemFonts: !0,
      enableXfa: !1,
      maxImageSize: 16777216
    });
    return C.onPassword = (P) => {
      o && (e.current = P, m(!0));
    }, C.promise.then((P) => {
      o && (h(P), m(!1), f.current.onLoad?.({
        format: "pdf",
        filename: n,
        size: w.byteLength,
        pageCount: P.numPages
      }));
    }).catch((P) => {
      if (!o) return;
      const z = new Error(P instanceof Error ? P.message : "PDF 解析失败。");
      l(z.message), f.current.onError?.(z);
    }), () => {
      o = !1, e.current = void 0, C.destroy().catch(() => {
      }).finally(() => {
        U.destroy(), $.terminate();
      });
    };
  }, [w, n, g, N, E, R]), F ? /* @__PURE__ */ t("p", { className: "yk-document-viewer__error", role: "alert", children: "暂时无法预览此 PDF，可下载后查看。" }) : i ? /* @__PURE__ */ b(
    "form",
    {
      className: "yk-pdf__password",
      onSubmit: (o) => {
        o.preventDefault(), e.current?.(k), x("");
      },
      children: [
        /* @__PURE__ */ t(
          I,
          {
            label: "文档密码",
            type: "password",
            required: !0,
            autoComplete: "off",
            value: k,
            onChange: (o) => x(o.target.value),
            description: "该 PDF 已加密，请输入正确密码。"
          }
        ),
        /* @__PURE__ */ t(D, { type: "submit", children: "解锁文档" })
      ]
    }
  ) : c ? /* @__PURE__ */ b("div", { className: "yk-pdf", children: [
    /* @__PURE__ */ b("div", { className: "yk-pdf__toolbar", children: [
      /* @__PURE__ */ t(
        D,
        {
          variant: "ghost",
          size: "sm",
          "aria-label": "PDF 上一页",
          disabled: a === 1,
          onClick: () => v(a - 1),
          children: "‹"
        }
      ),
      /* @__PURE__ */ b("span", { "aria-live": "polite", children: [
        "第 ",
        a,
        " / ",
        c.numPages,
        " 页"
      ] }),
      /* @__PURE__ */ t(
        D,
        {
          variant: "ghost",
          size: "sm",
          "aria-label": "PDF 下一页",
          disabled: a === c.numPages,
          onClick: () => v(a + 1),
          children: "›"
        }
      ),
      /* @__PURE__ */ t("span", { className: "yk-pdf__toolbar-spacer" }),
      /* @__PURE__ */ t(
        D,
        {
          variant: "ghost",
          size: "sm",
          "aria-label": "缩小 PDF",
          disabled: r <= 0.5,
          onClick: () => y(Math.max(0.5, r - 0.25)),
          children: "−"
        }
      ),
      /* @__PURE__ */ b(
        "button",
        {
          className: "yk-pdf__zoom",
          type: "button",
          onClick: () => y(1),
          "aria-label": "重置 PDF 缩放",
          children: [
            Math.round(r * 100),
            "%"
          ]
        }
      ),
      /* @__PURE__ */ t(
        D,
        {
          variant: "ghost",
          size: "sm",
          "aria-label": "放大 PDF",
          disabled: r >= 2,
          onClick: () => y(Math.min(2, r + 0.25)),
          children: "+"
        }
      )
    ] }),
    /* @__PURE__ */ t(
      T,
      {
        document: c,
        page: a,
        zoom: r,
        onError: (o) => {
          l(o.message), f.current.onError?.(o);
        }
      }
    )
  ] }) : /* @__PURE__ */ t("p", { className: "yk-document-viewer__loading", role: "status", children: "正在打开 PDF…" });
}
function T({
  document: w,
  page: n,
  zoom: s,
  onError: d
}) {
  const u = _(null), [c, h] = p(600);
  return M(() => {
    if (!u.current) return;
    const a = new ResizeObserver((v) => {
      const r = v[0]?.contentRect.width;
      r && r > 0 && h(r);
    });
    return a.observe(u.current), () => a.disconnect();
  }, []), /* @__PURE__ */ t(
    "div",
    {
      className: "yk-pdf__viewport",
      ref: u,
      tabIndex: 0,
      role: "region",
      "aria-label": `PDF 第 ${n} 页`,
      children: /* @__PURE__ */ t(
        W,
        {
          document: w,
          page: n,
          width: c,
          zoom: s,
          onError: d
        }
      )
    }
  );
}
function W({
  document: w,
  page: n,
  width: s,
  zoom: d,
  onError: u
}) {
  const c = _(null), h = _(null), [a, v] = p({ width: 1, height: 1, scale: 1 }), [r, y] = p(!1), F = _(u);
  return F.current = u, M(() => {
    let l = !0;
    y(!1);
    let k, x;
    return (async () => {
      const i = await w.getPage(n);
      if (!l || !c.current || !h.current) return;
      const m = i.getViewport({ scale: 1 });
      if (!Number.isFinite(m.width) || !Number.isFinite(m.height) || m.width <= 0 || m.height <= 0)
        throw new Error("无效 PDF 页面尺寸");
      const e = i.getViewport({
        scale: Math.min(Math.max(100, s - 32) / m.width, 1.5) * d
      }), f = Math.min(
        window.devicePixelRatio || 1,
        2,
        4096 / Math.max(e.width, e.height)
      ), g = c.current;
      if (g.width = Math.ceil(e.width * f), g.height = Math.ceil(e.height * f), g.style.width = `${e.width}px`, g.style.height = `${e.height}px`, v({ width: e.width, height: e.height, scale: e.scale }), k = i.render({
        canvas: g,
        viewport: e,
        transform: [f, 0, 0, f, 0, 0]
      }), await k.promise, !l) return;
      const N = await i.getTextContent();
      !l || !h.current || (x = new q({
        textContentSource: N,
        container: h.current,
        viewport: e
      }), await x.render(), l && y(!0));
    })().catch((i) => {
      l && i?.name !== "RenderingCancelledException" && i?.name !== "AbortException" && F.current(i instanceof Error ? i : new Error("PDF 页面绘制失败。"));
    }), () => {
      l = !1, k?.cancel(), x?.cancel();
    };
  }, [w, n, s, d]), /* @__PURE__ */ b(L, { children: [
    /* @__PURE__ */ b(
      "div",
      {
        className: "yk-pdf__page",
        style: {
          width: a.width,
          height: a.height,
          "--total-scale-factor": a.scale
        },
        "data-rendered": r,
        children: [
          /* @__PURE__ */ t("canvas", { ref: c, "aria-hidden": "true" }, `canvas-${n}-${s}-${d}`),
          /* @__PURE__ */ t("div", { ref: h, className: "yk-pdf__text" }, `text-${n}-${s}-${d}`)
        ]
      }
    ),
    /* @__PURE__ */ t("span", { className: r ? "yk-sr-only" : "yk-document-viewer__loading", role: "status", children: r ? `第 ${n} 页已渲染` : "正在绘制页面…" })
  ] });
}
export {
  Z as default
};

