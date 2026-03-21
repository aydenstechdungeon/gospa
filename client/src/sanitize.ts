/**
 * Lazy-loaded DOMPurify sanitizer module
 *
 * DOMPurify is only loaded when sanitization is actually needed,
 * reducing initial bundle size by ~20KB minified.
 *
 * SECURITY NOTES:
 * - This module provides the strongest XSS protection using DOMPurify 3.3.1
 * - For untrusted user content, always use this full sanitizer, not the simple variant
 * - Configure CSP with require-trusted-types-for 'script' for additional protection
 */

// DOMPurify configuration type
interface DOMPurifyConfig {
  ALLOWED_TAGS?: string[];
  ALLOWED_ATTR?: string[];
  ALLOW_DATA_ATTR?: boolean;
  FORBID_TAGS?: string[];
  FORBID_ATTR?: string[];
  ADD_ATTR?: string[];
  FORCE_BODY?: boolean;
  SANITIZE_DOM?: boolean;
  SANITIZE_NAMED_PROPS?: boolean;
  WHOLE_DOCUMENT?: boolean;
  RETURN_DOM?: boolean;
  RETURN_DOM_FRAGMENT?: boolean;
  RETURN_TRUSTED_TYPE?: boolean;
  ALLOWED_URI_REGEXP?: RegExp;
  KEEP_CONTENT?: boolean;
}

// DOMPurify instance type
type DOMPurifyInstance = {
  sanitize(html: string | Node, config?: DOMPurifyConfig): string | TrustedHTML;
};

/**
 * DOMPurify configuration for GoSPA
 *
 * This configuration prioritizes security while allowing common use cases:
 * - Blocks all script execution vectors
 * - Prevents DOM Clobbering via name/id attributes
 * - Sanitizes URLs in href/src attributes
 * - Maintains semantic HTML support
 *
 * @see https://github.com/cure53/DOMPurify/wiki/Configuration-Help
 */
export const PURIFY_CONFIG: DOMPurifyConfig = {
  // Allowed HTML tags - semantic HTML5 focus
  ALLOWED_TAGS: [
    // Semantic HTML5 structure
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
    // Text content
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
    // Lists
    "ul",
    "ol",
    "li",
    "dl",
    "dt",
    "dd",
    // Tables
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
    // Media (safe variants only)
    "img",
    "picture",
    "source",
    "audio",
    "video",
    "track",
    // Interactive (non-scripted)
    "details",
    "summary",
    "dialog",
    // Links
    "a",
    // Other semantic
    "address",
    "time",
    "data",
    "meter",
    "progress",
    "output",
    // SVG (safe subset - no animation events)
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
    // Explicit text node support
    "#text",
  ],

  // Allowed attributes - minimal and safe set
  ALLOWED_ATTR: [
    // Core attributes
    "class",
    "id",
    "title",
    "lang",
    "dir",
    // ARIA accessibility
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
    // Links
    "href",
    "target",
    "rel",
    "download",
    // Media
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
    // Audio/Video
    "controls",
    "loop",
    "muted",
    "preload",
    "autoplay",
    "poster",
    // Tables
    "colspan",
    "rowspan",
    "headers",
    "scope",
    "span",
    // Data
    "value",
    "datetime",
    // SVG core
    "xmlns",
    "viewBox",
    "preserveAspectRatio",
    // SVG presentation
    "fill",
    "stroke",
    "stroke-width",
    "stroke-linecap",
    "stroke-linejoin",
    "stroke-dasharray",
    "stroke-dashoffset",
    "stroke-opacity",
    "fill-opacity",
    // SVG geometry
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
    // SVG references
    "href",
  ],

  // Explicitly block data attributes to prevent DOM Clobbering
  ALLOW_DATA_ATTR: false,

  // Forbidden tags - defense in depth even if ALLOWED_TAGS is bypassed
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
    // Dangerous SVG/Math
    "math",
    "animate",
    "animateTransform",
    "animateMotion",
    "set",
  ],

  // Forbidden attributes - all event handlers and dangerous attributes
  FORBID_ATTR: [
    // Event handlers
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
    // SVG events
    "onbegin",
    "onend",
    "onrepeat",
    // Form/DOM Clobbering
    "name",
    "form",
    "formaction",
    "formmethod",
    "formtarget",
    "formenctype",
    "formnovalidate",
    // Dangerous URLs
    "xlink:href",
    "xmlns:xlink",
    // Other
    "action",
    "manifest",
    "http-equiv",
    "content",
  ],

  // Additional security settings
  ADD_ATTR: ["target"], // Already in ALLOWED_ATTR, but explicit here
  FORCE_BODY: true, // Always return body content, never full document
  SANITIZE_DOM: true, // Sanitize DOM Clobbering attributes
  SANITIZE_NAMED_PROPS: true, // Sanitize named properties
  WHOLE_DOCUMENT: false, // Never allow whole document sanitization
  RETURN_DOM: false, // Always return string
  RETURN_DOM_FRAGMENT: false,
  RETURN_TRUSTED_TYPE: false, // We handle Trusted Types separately if needed

  // URL validation - only allow safe protocols
  ALLOWED_URI_REGEXP:
    /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|sms|cid|xmpp|matrix):|[^a-z]|[a-z+.-]+(?:[^a-z+.-:]|$))/i,

  // Keep content of allowed elements but strip dangerous parts
  KEEP_CONTENT: true,
};

// Type for the config object
export type PurifyConfig = typeof PURIFY_CONFIG;

// Cached DOMPurify instance - lazy loaded
let domPurifyInstance: DOMPurifyInstance | null = null;
let domPurifyPromise: Promise<DOMPurifyInstance> | null = null;

/**
 * Get DOMPurify instance (lazy loaded)
 * Returns a promise that resolves to the DOMPurify instance
 */
async function getDOMPurify(): Promise<DOMPurifyInstance> {
  if (domPurifyInstance) {
    return domPurifyInstance;
  }

  if (domPurifyPromise) {
    return domPurifyPromise;
  }

  domPurifyPromise = import("dompurify").then((module) => {
    domPurifyInstance = module.default;
    return domPurifyInstance;
  });

  return domPurifyPromise;
}

/**
 * Async sanitization - preferred for most use cases
 * Loads DOMPurify on first call, then caches for subsequent calls
 *
 * @param html - Raw HTML string to sanitize
 * @returns Sanitized HTML string safe for insertion
 * @throws Never throws; returns empty string on error
 */
export async function sanitize(html: string): Promise<string> {
  try {
    const purify = await getDOMPurify();
    const result = purify.sanitize(html, PURIFY_CONFIG);
    // Handle both string and TrustedHTML returns
    return typeof result === "string"
      ? result
      : (result as unknown as TrustedHTML).toString();
  } catch (error) {
    console.error("[gospa] Sanitization failed:", error);
    // Return empty string on error - fail secure
    return "";
  }
}

/**
 * Sync sanitization - only use if DOMPurify is already loaded
 *
 * WARNING: If DOMPurify is not loaded, this returns an empty string.
 * Use `isSanitizerReady()` to check before calling if you need sync behavior.
 *
 * @param html - Raw HTML string to sanitize
 * @returns Sanitized HTML string, or empty string if DOMPurify not ready
 */
export function sanitizeSync(html: string): string {
  if (!domPurifyInstance) {
    console.warn(
      "[gospa] sanitizeSync: DOMPurify not loaded. Use sanitize() for async sanitization or preloadSanitizer() before use.",
    );
    // Fail secure: return empty string instead of potentially dangerous content
    return "";
  }
  try {
    const result = domPurifyInstance.sanitize(html, PURIFY_CONFIG);
    return typeof result === "string"
      ? result
      : (result as unknown as TrustedHTML).toString();
  } catch (error) {
    console.error("[gospa] Sync sanitization failed:", error);
    return "";
  }
}

/**
 * Check if DOMPurify is loaded and ready for synchronous sanitization
 */
export function isSanitizerReady(): boolean {
  return domPurifyInstance !== null;
}

/**
 * Preload DOMPurify - call during idle time for faster first sanitization
 *
 * Recommended usage:
 * ```typescript
 * // In your app initialization
 * if (typeof window !== 'undefined') {
 *   requestIdleCallback(() => preloadSanitizer());
 * }
 * ```
 */
export function preloadSanitizer(): void {
  if (!domPurifyInstance && !domPurifyPromise) {
    getDOMPurify().catch((err) => {
      console.error("[gospa] Failed to preload DOMPurify:", err);
    });
  }
}

/**
 * Sanitizer function for use with dom.ts setSanitizer
 *
 * This is the preferred sanitizer for GoSPA applications. It:
 * - Uses sync path when DOMPurify is loaded (better performance)
 * - Falls back to async loading when needed (ensures security)
 * - Returns empty string on any error (fail-secure)
 *
 * @param html - Raw HTML string to sanitize
 * @returns Sanitized HTML string, or Promise that resolves to sanitized HTML
 */
export function domPurifySanitizer(html: string): string | Promise<string> {
  // If DOMPurify is already loaded, use sync path for performance
  if (domPurifyInstance) {
    try {
      const result = domPurifyInstance.sanitize(html, PURIFY_CONFIG);
      return typeof result === "string"
        ? result
        : (result as unknown as TrustedHTML).toString();
    } catch (error) {
      console.error("[gospa] Sanitization error:", error);
      return "";
    }
  }
  // Otherwise use async path - this ensures proper sanitization
  return sanitize(html);
}
