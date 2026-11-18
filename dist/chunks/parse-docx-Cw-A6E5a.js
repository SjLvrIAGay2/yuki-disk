import n from "mammoth";
import l from "dompurify";
async function u(a, o) {
  const s = await n.convertToHtml(
    { arrayBuffer: a },
    {
      externalFileAccess: !1,
      includeEmbeddedStyleMap: !1,
      idPrefix: o,
      convertImage: n.images.imgElement(async (t) => /^image\/(png|jpeg|gif|webp|bmp)$/.test(t.contentType) ? { src: `data:${t.contentType};base64,${await t.readAsBase64String()}` } : { src: "" })
    }
  ), i = l.sanitize(s.value, {
    ALLOWED_TAGS: [
      "h1",
      "h2",
      "h3",
      "h4",
      "h5",
      "h6",
      "p",
      "br",
      "strong",
      "b",
      "em",
      "i",
      "u",
      "s",
      "del",
      "sup",
      "sub",
      "ul",
      "ol",
      "li",
      "table",
      "thead",
      "tbody",
      "tr",
      "th",
      "td",
      "a",
      "img",
      "blockquote",
      "pre",
      "code",
      "hr"
    ],
    ALLOWED_ATTR: ["href", "src", "alt", "title", "colspan", "rowspan", "start", "id", "scope"],
    SANITIZE_NAMED_PROPS: !0
  }), e = new DOMParser().parseFromString(i, "text/html");
  for (const t of e.querySelectorAll("a[href]")) {
    const r = t.getAttribute("href") ?? "";
    r.startsWith("#") && t.setAttribute("href", `#user-content-${r.slice(1)}`), /^(https?:)?\/\//i.test(r) && (t.setAttribute("target", "_blank"), t.setAttribute("rel", "noopener noreferrer"));
  }
  return {
    html: e.body.innerHTML,
    text: e.body.textContent ?? "",
    warnings: s.messages.map((t) => t.message)
  };
}
export {
  u as parseDocx
};

