import { jsx as r } from "react/jsx-runtime";
import { CodeBlock as n } from "./code-block.js";
import { Markdown as c } from "./markdown.js";
import { DocumentViewer as m } from "./document-viewer.js";
import { VideoPlayer as i } from "./video-player.js";
import { AudioPlayer as y } from "./audio-player.js";
import './assets/content-renderer.css';function _(t) {
  switch (t.type) {
    case "code": {
      const { type: o, ...e } = t;
      return /* @__PURE__ */ r(n, { ...e });
    }
    case "markdown": {
      const { type: o, ...e } = t;
      return /* @__PURE__ */ r(c, { ...e });
    }
    case "document": {
      const { type: o, ...e } = t;
      return /* @__PURE__ */ r(m, { ...e });
    }
    case "video": {
      const { type: o, ...e } = t;
      return /* @__PURE__ */ r(i, { ...e });
    }
    case "audio": {
      const { type: o, ...e } = t;
      return /* @__PURE__ */ r(y, { ...e });
    }
  }
}
export {
  _ as ContentRenderer
};

