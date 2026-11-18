import { jsx as t, jsxs as i, Fragment as f } from "react/jsx-runtime";
import { useState as y, useRef as L, useEffect as E } from "react";
import { f as k } from "./media-wH8MaEKZ.js";
import { Select as w } from "../select.js";
import '../assets/player.css';function v(e) {
  const r = e && Number.isFinite(e.duration) ? Math.max(0, e.duration) : 0;
  return {
    playing: e ? !e.paused && !e.ended : !1,
    time: e?.currentTime || 0,
    duration: r,
    buffered: e && r && e.buffered.length ? e.buffered.end(e.buffered.length - 1) / r * 100 : 0,
    volume: e?.volume ?? 1,
    muted: e?.muted ?? !1,
    rate: e?.playbackRate ?? 1,
    ended: e?.ended ?? !1
  };
}
function z(e) {
  const [r, a] = y(() => v(null)), [d, h] = y(!1), [p, u] = y(""), c = L(e);
  return c.current = e, E(() => {
    if (u(""), h(!1), a(v(e)), !e) return;
    const l = () => a(v(e)), x = () => {
      h(!0), l();
    }, N = () => {
      h(!1), l();
    }, M = [
      "loadedmetadata",
      "durationchange",
      "timeupdate",
      "progress",
      "volumechange",
      "ratechange",
      "play"
    ], C = ["playing", "canplay", "pause", "ended", "error", "emptied", "seeked"];
    return M.forEach((m) => e.addEventListener(m, l)), C.forEach((m) => e.addEventListener(m, N)), e.addEventListener("waiting", x), e.addEventListener("seeking", x), () => {
      M.forEach((m) => e.removeEventListener(m, l)), C.forEach((m) => e.removeEventListener(m, N)), e.removeEventListener("waiting", x), e.removeEventListener("seeking", x);
    };
  }, [e]), {
    ...r,
    waiting: d,
    error: p,
    toggle: async () => {
      if (e) {
        if (u(""), !e.paused && !e.ended) {
          e.pause();
          return;
        }
        try {
          await e.play();
        } catch (l) {
          c.current === e && l?.name !== "AbortError" && u("无法开始播放，请检查媒体文件或浏览器播放权限。");
        }
      }
    },
    seek: (l) => {
      if (!(!e || !r.duration))
        try {
          e.currentTime = Math.min(r.duration, Math.max(0, l)), a(v(e));
        } catch {
          u("当前媒体暂时无法跳转，请稍后重试。");
        }
    },
    volumeTo: (l) => {
      e && (e.volume = Math.max(0, Math.min(1, l)), e.muted = l === 0, a(v(e)));
    },
    toggleMute: () => {
      e && (e.muted = !(e.muted || e.volume === 0), !e.muted && e.volume === 0 && (e.volume = 1), a(v(e)));
    },
    rateTo: (l) => {
      e && (e.playbackRate = l, a(v(e)));
    }
  };
}
function b({
  name: e
}) {
  return /* @__PURE__ */ t(
    "svg",
    {
      viewBox: "0 0 24 24",
      width: "20",
      height: "20",
      fill: "none",
      stroke: "currentColor",
      strokeWidth: "1.6",
      strokeLinecap: "round",
      strokeLinejoin: "round",
      "aria-hidden": "true",
      children: {
        play: /* @__PURE__ */ t("path", { d: "m9 5 11 7-11 7z", fill: "currentColor", stroke: "none" }),
        pause: /* @__PURE__ */ t("path", { d: "M7 5h3v14H7zm7 0h3v14h-3z", fill: "currentColor", stroke: "none" }),
        replay: /* @__PURE__ */ t(f, { children: /* @__PURE__ */ t("path", { d: "M4 10a8 8 0 1 1 1 7M4 4v6h6" }) }),
        volume: /* @__PURE__ */ t(f, { children: /* @__PURE__ */ t("path", { d: "m11 4-6 5H2v6h3l6 5zM15 8a6 6 0 0 1 0 8m3-11a10 10 0 0 1 0 14" }) }),
        mute: /* @__PURE__ */ t(f, { children: /* @__PURE__ */ t("path", { d: "m11 4-6 5H2v6h3l6 5zM16 9l6 6m0-6-6 6" }) }),
        expand: /* @__PURE__ */ t("path", { d: "M8 3H3v5m13-5h5v5M3 16v5h5m13-5v5h-5" }),
        collapse: /* @__PURE__ */ t("path", { d: "M3 8h5V3m8 0v5h5M8 21v-5H3m18 0h-5v5" }),
        back: /* @__PURE__ */ i(f, { children: [
          /* @__PURE__ */ t("path", { d: "m4 8 5-5M4 8h7m-7 0a9 9 0 1 1-1 8" }),
          /* @__PURE__ */ t("text", { x: "9", y: "16", fill: "currentColor", stroke: "none", fontSize: "9", fontWeight: "600", children: "10" })
        ] }),
        forward: /* @__PURE__ */ i(f, { children: [
          /* @__PURE__ */ t("path", { d: "m20 8-5-5m5 5h-7m7 0a9 9 0 1 0 1 8" }),
          /* @__PURE__ */ t("text", { x: "7", y: "16", fill: "currentColor", stroke: "none", fontSize: "9", fontWeight: "600", children: "10" })
        ] }),
        music: /* @__PURE__ */ i(f, { children: [
          /* @__PURE__ */ t("path", { d: "M9 18V5l11-2v13M9 8l11-2" }),
          /* @__PURE__ */ t("ellipse", { cx: "6", cy: "18", rx: "3", ry: "3" }),
          /* @__PURE__ */ t("ellipse", { cx: "17", cy: "16", rx: "3", ry: "3" })
        ] })
      }[e]
    }
  );
}
function V({
  player: e,
  enabled: r,
  playbackRates: a = [0.5, 1, 1.5, 2],
  extra: d,
  label: h,
  menuContainer: p,
  onMenuOpenChange: u,
  compact: c = !1,
  summary: g
}) {
  const _ = [
    .../* @__PURE__ */ new Set([
      1,
      ...a.filter((n) => Number.isFinite(n) && n >= 0.25 && n <= 4),
      e.rate
    ])
  ].sort((n, s) => n - s), o = e.duration ? Math.min(100, e.time / e.duration * 100) : 0;
  return /* @__PURE__ */ i(
    "div",
    {
      className: `yk-player-controls${c ? " yk-player-controls--video" : ""}`,
      role: "group",
      "aria-label": `${h}播放控制`,
      children: [
        /* @__PURE__ */ i("div", { className: "yk-player-controls__timeline", children: [
          /* @__PURE__ */ t("span", { className: "yk-player-controls__time", children: k(e.time) }),
          /* @__PURE__ */ t(
            "input",
            {
              className: "yk-player-controls__range",
              type: "range",
              min: 0,
              max: e.duration || 1,
              step: "0.1",
              value: Math.min(e.time, e.duration || 1),
              disabled: !r || !e.duration,
              "aria-label": "播放进度",
              "aria-valuetext": `${k(e.time)} / ${k(e.duration || null)}`,
              style: {
                "--yk-player-progress": `${o}%`,
                "--yk-player-buffered": `${Math.min(100, e.buffered)}%`
              },
              onChange: (n) => e.seek(Number(n.target.value))
            }
          ),
          /* @__PURE__ */ t("span", { className: "yk-player-controls__time", children: k(e.duration || null) })
        ] }),
        /* @__PURE__ */ i("div", { className: "yk-player-controls__row", children: [
          /* @__PURE__ */ t(
            "button",
            {
              type: "button",
              className: "yk-player-controls__play",
              disabled: !r,
              onClick: () => {
                e.toggle();
              },
              "aria-label": e.playing ? "暂停" : e.ended ? "重新播放" : "播放",
              children: /* @__PURE__ */ t(b, { name: e.playing ? "pause" : e.ended ? "replay" : "play" })
            }
          ),
          c && /* @__PURE__ */ i("span", { className: "yk-player-controls__clock", children: [
            k(e.time),
            /* @__PURE__ */ t("span", { "aria-hidden": "true", children: " / " }),
            k(e.duration || null)
          ] }),
          /* @__PURE__ */ t(
            "button",
            {
              type: "button",
              className: "yk-player-controls__button yk-player-controls__skip",
              disabled: !r || !e.duration,
              "aria-label": "后退 10 秒",
              onClick: () => e.seek(e.time - 10),
              children: /* @__PURE__ */ t(b, { name: "back" })
            }
          ),
          /* @__PURE__ */ t(
            "button",
            {
              type: "button",
              className: "yk-player-controls__button yk-player-controls__skip",
              disabled: !r || !e.duration,
              "aria-label": "前进 10 秒",
              onClick: () => e.seek(e.time + 10),
              children: /* @__PURE__ */ t(b, { name: "forward" })
            }
          ),
          c && g && /* @__PURE__ */ t("span", { className: "yk-player-controls__summary", children: g }),
          /* @__PURE__ */ t("span", { className: "yk-player-controls__spacer" }),
          /* @__PURE__ */ i("div", { className: "yk-player-controls__volume", children: [
            /* @__PURE__ */ t(
              "button",
              {
                type: "button",
                className: "yk-player-controls__button",
                "aria-label": e.muted || !e.volume ? "取消静音" : "静音",
                onClick: e.toggleMute,
                children: /* @__PURE__ */ t(b, { name: e.muted || !e.volume ? "mute" : "volume" })
              }
            ),
            /* @__PURE__ */ t(
              "input",
              {
                className: "yk-player-controls__range",
                type: "range",
                min: "0",
                max: "1",
                step: "0.01",
                "aria-label": "音量",
                "aria-valuetext": `${Math.round((e.muted ? 0 : e.volume) * 100)}%`,
                value: e.muted ? 0 : e.volume,
                onChange: (n) => e.volumeTo(Number(n.target.value)),
                style: {
                  "--yk-player-progress": `${(e.muted ? 0 : e.volume) * 100}%`
                }
              }
            )
          ] }),
          /* @__PURE__ */ t(
            w,
            {
              className: "yk-player-controls__select",
              "aria-label": "播放速度",
              value: String(e.rate),
              onValueChange: (n) => e.rateTo(Number(n)),
              options: _.map((n) => ({ value: String(n), label: `${n}×` })),
              portalContainer: p,
              onOpenChange: u
            }
          ),
          d
        ] }),
        e.error && /* @__PURE__ */ t("p", { className: "yk-player-controls__error", role: "alert", children: e.error }),
        /* @__PURE__ */ t("span", { className: "yk-sr-only", role: "status", children: e.waiting ? "正在缓冲" : "" })
      ]
    }
  );
}
function H({
  media: e,
  container: r,
  onMenuOpenChange: a
}) {
  const [d, h] = y(!1), [p, u] = y(""), [c, g] = y(
    []
  );
  E(() => {
    if (!e) {
      g([]);
      return;
    }
    const o = () => g(
      Array.from(e.textTracks).map((s, l) => ({ track: s, index: l })).filter(({ track: s }) => ["subtitles", "captions"].includes(s.kind)).map(({ track: s, index: l }) => ({
        label: s.label || s.language || `字幕 ${l + 1}`,
        index: l,
        showing: s.mode === "showing"
      }))
    );
    o();
    const n = ["addtrack", "removetrack", "change"];
    return n.forEach((s) => e.textTracks.addEventListener(s, o)), e.addEventListener("loadedmetadata", o), () => {
      n.forEach((s) => e.textTracks.removeEventListener(s, o)), e.removeEventListener("loadedmetadata", o);
    };
  }, [e]), E(() => {
    const o = () => h(document.fullscreenElement === r && !!r);
    return document.addEventListener("fullscreenchange", o), o(), () => document.removeEventListener("fullscreenchange", o);
  }, [r]);
  const _ = async () => {
    try {
      u(""), d ? await document.exitFullscreen() : await r?.requestFullscreen();
    } catch {
      u("当前环境无法进入全屏。");
    }
  };
  return /* @__PURE__ */ i(f, { children: [
    c.length > 0 && /* @__PURE__ */ t(
      w,
      {
        className: "yk-player-controls__select yk-player-controls__captions",
        "aria-label": "字幕",
        value: String(c.find((o) => o.showing)?.index ?? -1),
        portalContainer: d ? r : void 0,
        onOpenChange: a,
        onValueChange: (o) => {
          e && Array.from(e.textTracks).forEach((n, s) => {
            ["subtitles", "captions"].includes(n.kind) && (n.mode = s === Number(o) ? "showing" : "disabled");
          });
        },
        options: [
          { value: "-1", label: "字幕关闭" },
          ...c.map((o) => ({ value: String(o.index), label: o.label }))
        ]
      }
    ),
    r?.requestFullscreen && /* @__PURE__ */ t(
      "button",
      {
        className: "yk-player-controls__button",
        type: "button",
        "aria-label": d ? "退出全屏" : "全屏",
        onClick: () => {
          _();
        },
        children: /* @__PURE__ */ t(b, { name: d ? "collapse" : "expand" })
      }
    ),
    p && /* @__PURE__ */ t("span", { role: "alert", className: "yk-player-controls__error", children: p })
  ] });
}
export {
  b as P,
  H as V,
  V as a,
  z as u
};

