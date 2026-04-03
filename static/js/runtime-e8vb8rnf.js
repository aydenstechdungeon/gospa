import {
  __require,
  __toESM
} from "./runtime-3hqyeswk.js";

// client/src/sanitize.ts
var PURIFY_CONFIG = {
  ALLOWED_TAGS: [
    "header",
    "footer",
    "nav",
    "main",
    "section",
    "article",
    "aside",
    "figure",
    "figcaption",
    "h1",
    "h2",
    "h3",
    "h4",
    "h5",
    "h6",
    "p",
    "br",
    "hr",
    "blockquote",
    "pre",
    "code",
    "span",
    "div",
    "em",
    "strong",
    "i",
    "b",
    "u",
    "small",
    "mark",
    "del",
    "ins",
    "abbr",
    "cite",
    "dfn",
    "kbd",
    "samp",
    "var",
    "sub",
    "sup",
    "ul",
    "ol",
    "li",
    "dl",
    "dt",
    "dd",
    "table",
    "thead",
    "tbody",
    "tfoot",
    "tr",
    "td",
    "th",
    "caption",
    "colgroup",
    "col",
    "img",
    "picture",
    "source",
    "audio",
    "video",
    "track",
    "details",
    "summary",
    "dialog",
    "a",
    "address",
    "time",
    "data",
    "meter",
    "progress",
    "output",
    "svg",
    "path",
    "rect",
    "circle",
    "line",
    "polyline",
    "polygon",
    "ellipse",
    "g",
    "defs",
    "use",
    "symbol",
    "text",
    "tspan",
    "#text"
  ],
  ALLOWED_ATTR: [
    "class",
    "id",
    "title",
    "lang",
    "dir",
    "role",
    "aria-label",
    "aria-labelledby",
    "aria-describedby",
    "aria-hidden",
    "aria-expanded",
    "aria-selected",
    "aria-checked",
    "aria-pressed",
    "aria-disabled",
    "href",
    "target",
    "rel",
    "download",
    "src",
    "alt",
    "width",
    "height",
    "loading",
    "decoding",
    "crossorigin",
    "srcset",
    "sizes",
    "media",
    "type",
    "controls",
    "loop",
    "muted",
    "preload",
    "autoplay",
    "poster",
    "colspan",
    "rowspan",
    "headers",
    "scope",
    "span",
    "value",
    "datetime",
    "xmlns",
    "viewBox",
    "preserveAspectRatio",
    "fill",
    "stroke",
    "stroke-width",
    "stroke-linecap",
    "stroke-linejoin",
    "stroke-dasharray",
    "stroke-dashoffset",
    "stroke-opacity",
    "fill-opacity",
    "d",
    "x",
    "y",
    "width",
    "height",
    "rx",
    "ry",
    "points",
    "cx",
    "cy",
    "r",
    "x1",
    "y1",
    "x2",
    "y2",
    "transform",
    "href"
  ],
  ALLOW_DATA_ATTR: false,
  FORBID_TAGS: [
    "script",
    "iframe",
    "object",
    "embed",
    "applet",
    "frame",
    "frameset",
    "form",
    "input",
    "button",
    "select",
    "textarea",
    "fieldset",
    "label",
    "meta",
    "link",
    "base",
    "head",
    "body",
    "html",
    "noscript",
    "noframes",
    "noembed",
    "math",
    "animate",
    "animateTransform",
    "animateMotion",
    "set"
  ],
  FORBID_ATTR: [
    "onerror",
    "onload",
    "onclick",
    "onmouseover",
    "onfocus",
    "onblur",
    "onchange",
    "onsubmit",
    "oninput",
    "onkeydown",
    "onkeypress",
    "onkeyup",
    "onmousedown",
    "onmouseup",
    "onmousemove",
    "onmouseenter",
    "onmouseleave",
    "ondblclick",
    "oncontextmenu",
    "onwheel",
    "onscroll",
    "onresize",
    "onselect",
    "onselectionchange",
    "oncut",
    "oncopy",
    "onpaste",
    "ondrag",
    "ondragstart",
    "ondragend",
    "ondrop",
    "ondragover",
    "ondragenter",
    "ondragleave",
    "onanimationstart",
    "onanimationend",
    "onanimationiteration",
    "ontransitionstart",
    "ontransitionend",
    "ontransitionrun",
    "onbegin",
    "onend",
    "onrepeat",
    "name",
    "form",
    "formaction",
    "formmethod",
    "formtarget",
    "formenctype",
    "formnovalidate",
    "xlink:href",
    "xmlns:xlink",
    "action",
    "manifest",
    "http-equiv",
    "content"
  ],
  ADD_ATTR: ["target"],
  FORCE_BODY: true,
  SANITIZE_DOM: true,
  SANITIZE_NAMED_PROPS: true,
  WHOLE_DOCUMENT: false,
  RETURN_DOM: false,
  RETURN_DOM_FRAGMENT: false,
  RETURN_TRUSTED_TYPE: false,
  ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|sms|cid|xmpp|matrix):|[^a-z]|[a-z+.-]+(?:[^a-z+.-:]|$))/i,
  KEEP_CONTENT: true
};
var domPurifyInstance = null;
var domPurifyPromise = null;
async function getDOMPurify() {
  if (domPurifyInstance) {
    return domPurifyInstance;
  }
  if (domPurifyPromise) {
    return domPurifyPromise;
  }
  domPurifyPromise = import("./purify.es-nf7jtsvx.js").then((module) => {
    domPurifyInstance = module.default;
    return domPurifyInstance;
  });
  return domPurifyPromise;
}
async function sanitize(html) {
  try {
    const purify = await getDOMPurify();
    const result = purify.sanitize(html, PURIFY_CONFIG);
    return typeof result === "string" ? result : result.toString();
  } catch (error) {
    console.error("[gospa] Sanitization failed:", error);
    return "";
  }
}
function sanitizeSync(html) {
  if (!domPurifyInstance) {
    console.warn("[gospa] sanitizeSync: DOMPurify not loaded. Use sanitize() for async sanitization or preloadSanitizer() before use.");
    return "";
  }
  try {
    const result = domPurifyInstance.sanitize(html, PURIFY_CONFIG);
    return typeof result === "string" ? result : result.toString();
  } catch (error) {
    console.error("[gospa] Sync sanitization failed:", error);
    return "";
  }
}
function isSanitizerReady() {
  return domPurifyInstance !== null;
}
function preloadSanitizer() {
  if (!domPurifyInstance && !domPurifyPromise) {
    getDOMPurify().catch((err) => {
      console.error("[gospa] Failed to preload DOMPurify:", err);
    });
  }
}
function domPurifySanitizer(html) {
  if (domPurifyInstance) {
    try {
      const result = domPurifyInstance.sanitize(html, PURIFY_CONFIG);
      return typeof result === "string" ? result : result.toString();
    } catch (error) {
      console.error("[gospa] Sanitization error:", error);
      return "";
    }
  }
  return sanitize(html);
}

export { PURIFY_CONFIG, sanitize, sanitizeSync, isSanitizerReady, preloadSanitizer, domPurifySanitizer };
