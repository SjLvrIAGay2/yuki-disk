import { jsxs as c, jsx as a } from "react/jsx-runtime";
import { forwardRef as H, useState as t, useImperativeHandle as L, useEffect as R, createElement as q } from "react";
import { u as z, P as D, a as G } from "./chunks/player-_TVa3IPY.js";
import { c as M } from "./chunks/tokens-BqLZJxpY.js";
import { u as J, f as K, m as O, r as Q } from "./chunks/media-wH8MaEKZ.js";
import { p as ra } from "./chunks/media-wH8MaEKZ.js";
import './assets/audio-player.css';const V = H(function({
  src: i,
  title: d = "音频",
  description: m,
  cover: r,
  showMetadata: N = !0,
  onMetadata: b,
  onMediaError: P,
  onLoadedMetadata: x,
  onError: E,
  className: C,
  style: T,
  children: j,
  controls: y = !0,
  controlsVariant: p = "custom",
  playbackRates: w,
  preload: A = "metadata",
  ...f
}, F) {
  const e = J(i), [n, I] = t(null);
  L(F, () => n, [n]);
  const s = z(n), [_, S] = t(), [h, k] = t(), [g, u] = t(), B = h?.url === e ? h?.value : void 0, v = g?.url === e ? g?.value : void 0;
  return R(() => {
    k(void 0), u(void 0);
  }, [i]), /* @__PURE__ */ c(
    "figure",
    {
      className: M("yk-audio-player", C),
      style: T,
      "data-playing": s.playing,
      children: [
        /* @__PURE__ */ c("figcaption", { children: [
          /* @__PURE__ */ a(
            "span",
            {
              className: M(
                "yk-audio-player__art",
                (!r || _ === r) && "yk-audio-player__art--vinyl"
              ),
              "aria-hidden": "true",
              children: r && _ !== r ? /* @__PURE__ */ a("img", { src: r, alt: "", onError: () => S(r) }) : /* @__PURE__ */ a("span", { className: "yk-audio-player__disc", children: /* @__PURE__ */ a(D, { name: "music" }) })
            }
          ),
          /* @__PURE__ */ c("div", { children: [
            /* @__PURE__ */ a("span", { className: "yk-audio-player__eyebrow", children: s.playing ? "正在播放" : s.ended ? "播放完毕" : "准备播放" }),
            /* @__PURE__ */ a("strong", { children: d }),
            m && /* @__PURE__ */ a("p", { children: m })
          ] }),
          N && /* @__PURE__ */ a("span", { className: "yk-audio-player__duration", children: K(B?.duration ?? null) })
        ] }),
        /* @__PURE__ */ q(
          "audio",
          {
            ...f,
            key: e,
            src: e || void 0,
            ref: I,
            title: d,
            "aria-label": f["aria-label"] ?? d,
            controls: y && p === "native",
            preload: A,
            onLoadedMetadata: (l) => {
              const o = Q(l.currentTarget, i);
              k({ url: e ?? "", value: o }), u(void 0), b?.(o), x?.(l);
            },
            onError: (l) => {
              const o = O(l.currentTarget.error);
              u({ url: e ?? "", value: o }), P?.(o), E?.(l);
            }
          },
          j
        ),
        y && p === "custom" && /* @__PURE__ */ a(
          G,
          {
            player: s,
            enabled: !!e,
            label: d,
            playbackRates: w
          }
        ),
        !i && /* @__PURE__ */ a("p", { className: "yk-audio-player__hint", children: "请选择音频文件" }),
        v && /* @__PURE__ */ a("p", { className: "yk-audio-player__error", role: "alert", children: v.message })
      ]
    }
  );
});
export {
  V as AudioPlayer,
  K as formatMediaTime,
  ra as parseMediaMetadata
};

