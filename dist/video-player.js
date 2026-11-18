import { jsxs as y, jsx as r, Fragment as z } from "react/jsx-runtime";
import { useState as v, useEffect as T, forwardRef as A, useImperativeHandle as B, createElement as D } from "react";
import { u as G, P as H, a as J, V as K } from "./chunks/player-_TVa3IPY.js";
import { c as Q } from "./chunks/tokens-BqLZJxpY.js";
import { u as U, m as W, r as X, f as Y } from "./chunks/media-wH8MaEKZ.js";
import { p as ie } from "./chunks/media-wH8MaEKZ.js";
import './assets/video-player.css';function Z(e, d) {
  const [p, i] = v(!1), [g, h] = v(!1);
  return T(() => {
    const n = () => i(!!e && document.fullscreenElement === e);
    return document.addEventListener("fullscreenchange", n), n(), () => document.removeEventListener("fullscreenchange", n);
  }, [e]), T(() => {
    if (h(!1), !e || !p || !d) return;
    let n, L = !0, f = !1, s = e.contains(document.activeElement) && !!document.activeElement?.matches(":focus-visible");
    const o = () => {
      clearTimeout(n), h(!1), !f && !s && (n = setTimeout(() => h(!0), 3e3));
    }, k = () => {
      s = !1, o();
    }, l = () => {
      f = !0, s = !1, o();
    }, t = () => {
      f && (f = !1, o());
    }, _ = () => {
      s = !0, o();
    }, b = () => {
      s = s || !!document.activeElement?.matches(":focus-visible"), o();
    }, M = () => queueMicrotask(() => {
      L && (e.contains(document.activeElement) || (s = !1), o());
    });
    return e.addEventListener("pointermove", k), e.addEventListener("pointerdown", l, !0), document.addEventListener("pointerup", t, !0), document.addEventListener("pointercancel", t, !0), e.addEventListener("keydown", _, !0), e.addEventListener("focusin", b), e.addEventListener("focusout", M), o(), () => {
      L = !1, clearTimeout(n), e.removeEventListener("pointermove", k), e.removeEventListener("pointerdown", l, !0), document.removeEventListener("pointerup", t, !0), document.removeEventListener("pointercancel", t, !0), e.removeEventListener("keydown", _, !0), e.removeEventListener("focusin", b), e.removeEventListener("focusout", M);
    };
  }, [e, p, d]), { fullscreen: p, hidden: p && d && g };
}
const se = A(function({
  src: d,
  tracks: p = [],
  title: i = "视频",
  showMetadata: g = !0,
  onMetadata: h,
  onMediaError: n,
  onLoadedMetadata: L,
  onError: f,
  className: s,
  style: o,
  children: k,
  controls: l = !0,
  controlsVariant: t = "custom",
  playbackRates: _,
  preload: b = "metadata",
  playsInline: M = !0,
  ...$
}, S) {
  const u = U(d), [N, V] = v(null), [w, q] = v(null);
  B(S, () => N, [N]);
  const c = G(N), [F, O] = v(), [j, C] = v(), m = F?.url === u ? F?.value : void 0, x = j?.url === u ? j?.value : void 0, [R, P] = v(!1), I = Z(
    w,
    l && t === "custom" && c.playing && !c.error && !x && !R
  );
  return T(() => {
    O(void 0), C(void 0), P(!1);
  }, [d]), /* @__PURE__ */ y(
    "figure",
    {
      ref: q,
      className: Q("yk-video-player", s),
      style: o,
      "data-controls-hidden": I.hidden,
      "data-custom-controls": l && t === "custom",
      children: [
        /* @__PURE__ */ y("div", { className: "yk-video-player__stage", children: [
          /* @__PURE__ */ D(
            "video",
            {
              ...$,
              key: u,
              src: u || void 0,
              ref: V,
              title: i,
              "aria-label": $["aria-label"] ?? i,
              controls: l && t === "native",
              preload: b,
              playsInline: M,
              onLoadedMetadata: (a) => {
                const E = X(a.currentTarget, d);
                O({ url: u ?? "", value: E }), C(void 0), h?.(E), L?.(a);
              },
              onError: (a) => {
                const E = W(a.currentTarget.error);
                C({ url: u ?? "", value: E }), n?.(E), f?.(a);
              }
            },
            p.map((a) => /* @__PURE__ */ r(
              "track",
              {
                ...a,
                kind: a.kind ?? "captions"
              },
              `${a.src}-${a.srcLang}`
            )),
            k
          ),
          l && t === "custom" && !c.playing && /* @__PURE__ */ r(
            "button",
            {
              className: "yk-video-player__start",
              type: "button",
              disabled: !u,
              "aria-label": c.ended ? "重新播放视频" : "播放视频",
              onClick: () => {
                c.toggle();
              },
              children: /* @__PURE__ */ r(H, { name: c.ended ? "replay" : "play" })
            }
          ),
          c.waiting && /* @__PURE__ */ r("span", { className: "yk-video-player__loading", "aria-hidden": "true", children: "正在缓冲…" })
        ] }),
        /* @__PURE__ */ y("figcaption", { className: "yk-video-player__chrome", children: [
          /* @__PURE__ */ y("div", { className: "yk-video-player__caption", children: [
            /* @__PURE__ */ r("span", { children: i }),
            g && /* @__PURE__ */ r("span", { className: "yk-video-player__metadata", children: m ? `${m.width} × ${m.height} · ${Y(m.duration)}` : d ? "等待媒体元数据" : "请选择视频文件" })
          ] }),
          l && t === "custom" && /* @__PURE__ */ r(
            J,
            {
              compact: !0,
              summary: /* @__PURE__ */ y(z, { children: [
                /* @__PURE__ */ r("span", { title: i, children: i }),
                g && m && /* @__PURE__ */ y("span", { className: "yk-video-player__metadata", children: [
                  m.width,
                  " × ",
                  m.height
                ] })
              ] }),
              player: c,
              enabled: !!u,
              label: i,
              playbackRates: _,
              menuContainer: I.fullscreen ? w : void 0,
              onMenuOpenChange: P,
              extra: /* @__PURE__ */ r(
                K,
                {
                  media: N,
                  container: w,
                  onMenuOpenChange: P
                }
              )
            }
          )
        ] }),
        x && /* @__PURE__ */ r("p", { className: "yk-video-player__error", role: "alert", children: x.message })
      ]
    }
  );
});
export {
  se as VideoPlayer,
  Y as formatMediaTime,
  ie as parseMediaMetadata
};

