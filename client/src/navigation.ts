// GoSPA Client-side Navigation
// Enables SPA-style navigation without full page reloads

// Navigation state
const state = {
  currentPath: window.location.pathname,
  isNavigating: false,
  pendingNavigation: null as Promise<boolean> | null,
  abortController: null as AbortController | null,
};

// Navigation options
export interface NavigateOptions {
  replace?: boolean;
  scrollToTop?: boolean;
  preserveState?: boolean;
}

export interface SpeculativePrefetchingConfig {
  enabled?: boolean;
  ttl?: number;
  hoverDelay?: number;
  viewportMargin?: number;
}

export interface URLParsingCacheConfig {
  enabled?: boolean;
  maxSize?: number;
  ttl?: number;
}

export interface IdleCallbackBatchUpdatesConfig {
  enabled?: boolean;
  fallbackToMicrotask?: boolean;
}

export interface LazyRuntimeInitializationConfig {
  enabled?: boolean;
  deferBindings?: boolean;
}

export interface ServiceWorkerNavigationCachingConfig {
  enabled?: boolean;
  cacheName?: string;
  path?: string;
}

export interface ViewTransitionsConfig {
  enabled?: boolean;
  fallbackToClassic?: boolean;
}

export interface ProgressBarConfig {
  enabled?: boolean;
  color?: string;
  height?: string;
}

export interface ScriptExecutionConfig {
  executeMarkedOnly?: boolean;
}

export interface NavigationOptions {
  speculativePrefetching?: SpeculativePrefetchingConfig;
  urlParsingCache?: URLParsingCacheConfig;
  idleCallbackBatchUpdates?: IdleCallbackBatchUpdatesConfig;
  lazyRuntimeInitialization?: LazyRuntimeInitializationConfig;
  serviceWorkerNavigationCaching?: ServiceWorkerNavigationCachingConfig;
  viewTransitions?: ViewTransitionsConfig;
  progressBar?: ProgressBarConfig;
  scriptExecution?: ScriptExecutionConfig;
}

// Navigation event handlers
type NavigationCallback = (path: string) => void;
const beforeNavCallbacks: Set<NavigationCallback> = new Set();
const afterNavCallbacks: Set<NavigationCallback> = new Set();

// Register callbacks
export function onBeforeNavigate(cb: NavigationCallback): () => void {
  beforeNavCallbacks.add(cb);
  return () => beforeNavCallbacks.delete(cb);
}

export function onAfterNavigate(cb: NavigationCallback): () => void {
  afterNavCallbacks.add(cb);
  return () => afterNavCallbacks.delete(cb);
}

const DEFAULT_NAVIGATION_OPTIONS: Required<NavigationOptions> = {
  speculativePrefetching: {
    enabled: true,
    ttl: 30_000,
    hoverDelay: 50,
    viewportMargin: 150,
  },
  urlParsingCache: {
    enabled: true,
    maxSize: 100,
    ttl: 30_000,
  },
  idleCallbackBatchUpdates: {
    enabled: true,
    fallbackToMicrotask: true,
  },
  lazyRuntimeInitialization: {
    enabled: true,
    deferBindings: true,
  },
  serviceWorkerNavigationCaching: {
    enabled: false,
    cacheName: "gospa-navigation-cache",
    path: "/gospa-navigation-sw.js",
  },
  viewTransitions: {
    enabled: true,
    fallbackToClassic: true,
  },
  progressBar: {
    enabled: true,
    color: "#3b82f6",
    height: "2px",
  },
  scriptExecution: {
    executeMarkedOnly: true,
  },
};

let navigationOptionsConfig: Required<NavigationOptions> = {
  ...DEFAULT_NAVIGATION_OPTIONS,
  speculativePrefetching: {
    ...DEFAULT_NAVIGATION_OPTIONS.speculativePrefetching,
  },
  urlParsingCache: { ...DEFAULT_NAVIGATION_OPTIONS.urlParsingCache },
  idleCallbackBatchUpdates: {
    ...DEFAULT_NAVIGATION_OPTIONS.idleCallbackBatchUpdates,
  },
  lazyRuntimeInitialization: {
    ...DEFAULT_NAVIGATION_OPTIONS.lazyRuntimeInitialization,
  },
  serviceWorkerNavigationCaching: {
    ...DEFAULT_NAVIGATION_OPTIONS.serviceWorkerNavigationCaching,
  },
  viewTransitions: { ...DEFAULT_NAVIGATION_OPTIONS.viewTransitions },
  progressBar: { ...DEFAULT_NAVIGATION_OPTIONS.progressBar },
  scriptExecution: { ...DEFAULT_NAVIGATION_OPTIONS.scriptExecution },
};

interface CachedURL {
  url: URL;
  expiresAt: number;
}

const parsedURLCache = new Map<string, CachedURL>();
const hoverPrefetchTimers = new Map<string, number>();
let prefetchObserver: IntersectionObserver | null = null;
let clickDelegateContainer: Element | Document = document;

export function setNavigationOptions(config: NavigationOptions): void {
  // Clear URL parsing cache if being disabled
  if (
    config.urlParsingCache?.enabled === false &&
    navigationOptionsConfig.urlParsingCache.enabled
  ) {
    parsedURLCache.clear();
  }

  // Clear prefetch cache if speculative prefetching is being disabled
  if (
    config.speculativePrefetching?.enabled === false &&
    navigationOptionsConfig.speculativePrefetching.enabled
  ) {
    prefetchCache.clear();
  }

  navigationOptionsConfig = {
    ...navigationOptionsConfig,
    speculativePrefetching: {
      ...navigationOptionsConfig.speculativePrefetching,
      ...(config.speculativePrefetching ?? {}),
    },
    urlParsingCache: {
      ...navigationOptionsConfig.urlParsingCache,
      ...(config.urlParsingCache ?? {}),
    },
    idleCallbackBatchUpdates: {
      ...navigationOptionsConfig.idleCallbackBatchUpdates,
      ...(config.idleCallbackBatchUpdates ?? {}),
    },
    lazyRuntimeInitialization: {
      ...navigationOptionsConfig.lazyRuntimeInitialization,
      ...(config.lazyRuntimeInitialization ?? {}),
    },
    serviceWorkerNavigationCaching: {
      ...navigationOptionsConfig.serviceWorkerNavigationCaching,
      ...(config.serviceWorkerNavigationCaching ?? {}),
    },
    viewTransitions: {
      ...navigationOptionsConfig.viewTransitions,
      ...(config.viewTransitions ?? {}),
    },
    progressBar: {
      ...navigationOptionsConfig.progressBar,
      ...(config.progressBar ?? {}),
    },
    scriptExecution: {
      ...navigationOptionsConfig.scriptExecution,
      ...(config.scriptExecution ?? {}),
    },
  };
}

class ProgressBar {
  private el: HTMLDivElement | null = null;
  private interval: number | null = null;
  private progress = 0;

  start() {
    if (!navigationOptionsConfig.progressBar.enabled) return;
    this.reset();
    this.el = document.createElement("div");
    const cfg = navigationOptionsConfig.progressBar;
    Object.assign(this.el.style, {
      position: "fixed",
      top: "0",
      left: "0",
      height: cfg.height ?? "2px",
      backgroundColor: cfg.color ?? "#3b82f6",
      zIndex: "9999",
      transition: "width 0.3s ease-out, opacity 0.3s ease-in-out",
      width: "0%",
      opacity: "1",
      boxShadow: `0 0 10px ${cfg.color ?? "#3b82f6"}`,
    });
    document.body.appendChild(this.el);

    this.progress = 0;
    this.interval = window.setInterval(() => {
      if (this.progress < 90) {
        this.progress += (90 - this.progress) * 0.1;
        if (this.el) this.el.style.width = `${this.progress}%`;
      }
    }, 100);
  }

  finish() {
    if (!this.el) return;
    clearInterval(this.interval!);
    this.el.style.width = "100%";
    setTimeout(() => {
      if (this.el) {
        this.el.style.opacity = "0";
        setTimeout(() => this.reset(), 300);
      }
    }, 100);
  }

  private reset() {
    if (this.el) {
      this.el.remove();
      this.el = null;
    }
    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }
  }
}

const progressBar = new ProgressBar();
const scrollPositions = new Map<string, number>();

function getCachedURL(href: string): URL | null {
  const cacheCfg = navigationOptionsConfig.urlParsingCache;
  if (!cacheCfg.enabled) {
    try {
      return new URL(href, window.location.origin);
    } catch {
      return null;
    }
  }

  const now = Date.now();
  const cached = parsedURLCache.get(href);
  if (cached && cached.expiresAt > now) {
    parsedURLCache.delete(href);
    parsedURLCache.set(href, cached);
    return cached.url;
  }

  if (cached) {
    parsedURLCache.delete(href);
  }

  let parsed: URL;
  try {
    parsed = new URL(href, window.location.origin);
  } catch {
    return null;
  }

  parsedURLCache.set(href, {
    url: parsed,
    expiresAt: now + Math.max(1000, cacheCfg.ttl ?? 30000),
  });

  while (parsedURLCache.size > Math.max(1, cacheCfg.maxSize ?? 100)) {
    const first = parsedURLCache.keys().next().value;
    if (!first) break;
    parsedURLCache.delete(first);
  }

  return parsed;
}

// Check if a link is internal (same origin)
function isInternalLink(link: HTMLAnchorElement): boolean {
  const href = link.getAttribute("href");

  if (
    !href ||
    href.startsWith("#") ||
    href.startsWith("javascript:") ||
    href.startsWith("mailto:") ||
    href.startsWith("tel:") ||
    href.startsWith("sms:") ||
    href.startsWith("blob:") ||
    href.startsWith("data:")
  ) {
    return false;
  }

  const urlObj = getCachedURL(href);
  if (!urlObj) {
    return false;
  }

  if (urlObj.origin !== window.location.origin) {
    return false;
  }

  // Explicit manual overrides to bypass SPA routing
  if (
    link.hasAttribute("data-gospa-reload") ||
    link.hasAttribute("data-external") ||
    link.hasAttribute("download") ||
    link.getAttribute("target") === "_blank"
  ) {
    return false;
  }

  // Explicit manual override to force SPA routing (bypasses dot heuristic)
  if (link.hasAttribute("data-gospa-link")) {
    return true;
  }

  // The Heuristic: URLs with a file extension in the last segment are treated as assets by default.
  // This entirely replaces the brittle 100+ item IGNORED_EXTENSIONS blacklist.
  const pathname = urlObj.pathname;
  const lastSegment = pathname.slice(pathname.lastIndexOf("/") + 1);
  const dotIndex = lastSegment.lastIndexOf(".");
  if (dotIndex !== -1 && dotIndex < lastSegment.length - 1) {
    const ext = lastSegment.slice(dotIndex + 1).toLowerCase();
    // Only explicitly allow known HTML page extensions to bypass this rule
    if (ext !== "html" && ext !== "htm") {
      return false;
    }
  }

  return true;
}

// Page data type
interface PageData {
  content: string;
  title: string;
  head: string;
}

// Prefetch cache
interface PrefetchEntry {
  data: PageData;
  expiresAt: number;
}
const prefetchCache = new Map<string, PrefetchEntry>();

// Fetch page content from server
async function fetchPageFromServer(
  path: string,
  signal?: AbortSignal,
): Promise<PageData | null> {
  try {
    const response = await fetch(path, {
      signal,
      headers: {
        "X-Requested-With": "GoSPA-Navigate",
        Accept: "text/html",
      },
    });

    if (!response.ok) {
      console.error("[GoSPA] Navigation failed:", response.status);
      return null;
    }

    // Security & Robustness Phase: Validate Content-Type
    // If the server unexpectedly returned JSON, an image, or binary data, abort SPA navigation.
    // This prevents attempting to parse non-HTML data, which causes fatal JS errors.
    const contentType = response.headers.get("content-type");
    if (contentType && !contentType.includes("text/html")) {
      console.warn(
        `[GoSPA] Intercepted non-HTML response (${contentType}) for path ${path}. Falling back to standard navigation.`,
      );
      return null;
    }

    const html = await response.text();

    // Parse the HTML response
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");

    // Extract content
    let content: string;

    const rootEl = doc.querySelector("[data-gospa-root]");
    const pageContentEl = doc.querySelector("[data-gospa-page-content]");
    const mainEl = doc.querySelector("main");

    if (rootEl) {
      content = rootEl.innerHTML;
    } else if (pageContentEl) {
      content = pageContentEl.innerHTML;
    } else if (mainEl) {
      content = mainEl.innerHTML;
    } else {
      content = doc.body.innerHTML;
    }

    // Extract title
    const title = doc.querySelector("title")?.textContent || "";

    // Extract head elements (for head management)
    const headEl = doc.querySelector("head");
    const head = headEl ? headEl.innerHTML : "";

    return { content, title, head };
  } catch (error) {
    console.error("[GoSPA] Navigation error:", error);
    return null;
  }
}

// Get page data (from cache or server)
async function getPageData(
  path: string,
  signal?: AbortSignal,
): Promise<PageData | null> {
  const cached = prefetchCache.get(path);
  if (cached && cached.expiresAt > Date.now()) {
    prefetchCache.delete(path);
    prefetchCache.set(path, cached);
    return cached.data;
  }
  if (cached) prefetchCache.delete(path);
  return fetchPageFromServer(path, signal);
}

// Content is trusted - Templ auto-escapes on the server
// For user-generated content, use 'gospa/runtime-secure' which includes DOMPurify
// For data-bind="html:*" bindings, we add sanitization as a safety layer
async function prepareContent(html: string): Promise<string> {
  // Return HTML as-is - server is trusted, CSP provides XSS protection
  return html;
}

// Sanitize HTML for data-bind="html:*" bindings
// By default, this trusts the server (Templ auto-escapes).
// For user-generated content, use 'gospa/runtime-secure' which enables DOMPurify.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let DOMPurify: ((dirty: string) => string) | null = null;
async function sanitizeHTML(html: string): Promise<string> {
  // Try to use DOMPurify if loaded (from runtime-secure)
  if (DOMPurify != null) {
    return DOMPurify(html);
  }

  // Check if DOMPurify is available globally
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const globalPurify = (window as any).DOMPurify as
    | ((dirty: string) => string)
    | null;
  if (globalPurify != null) {
    DOMPurify = globalPurify;
    return globalPurify(html);
  }

  // Default: Trust the server (Templ auto-escapes)
  // For UGC, you should be using runtime-secure which sets the sanitizer
  return html;
}

function patchAttributes(current: Element, incoming: Element): void {
  for (const attr of Array.from(current.attributes)) {
    if (!incoming.hasAttribute(attr.name)) {
      current.removeAttribute(attr.name);
    }
  }

  for (const attr of Array.from(incoming.attributes)) {
    if (current.getAttribute(attr.name) !== attr.value) {
      current.setAttribute(attr.name, attr.value);
    }
  }
}

function patchNode(currentNode: Node, incomingNode: Node): void {
  if (currentNode.isEqualNode(incomingNode)) {
    return;
  }

  if (currentNode.nodeType !== incomingNode.nodeType) {
    currentNode.parentNode?.replaceChild(
      incomingNode.cloneNode(true),
      currentNode,
    );
    return;
  }

  if (currentNode.nodeType === Node.TEXT_NODE) {
    if (currentNode.textContent !== incomingNode.textContent) {
      currentNode.textContent = incomingNode.textContent;
    }
    return;
  }

  if (!(currentNode instanceof Element) || !(incomingNode instanceof Element)) {
    return;
  }

  if (currentNode.tagName !== incomingNode.tagName) {
    currentNode.parentNode?.replaceChild(
      incomingNode.cloneNode(true),
      currentNode,
    );
    return;
  }

  // Prevent morphing structurally unrelated elements
  if (
    (currentNode.id && currentNode.id !== incomingNode.id) ||
    (incomingNode.id && currentNode.id !== incomingNode.id) ||
    currentNode.getAttribute("data-gospa-page") !==
      incomingNode.getAttribute("data-gospa-page")
  ) {
    currentNode.parentNode?.replaceChild(
      incomingNode.cloneNode(true),
      currentNode,
    );
    return;
  }

  // Skip patching if the element is marked as permanent.
  // This allows client-side scripts to manage the element's content without server interference.
  if (currentNode.hasAttribute("data-gospa-permanent")) {
    return;
  }

  patchAttributes(currentNode, incomingNode);

  const currentChildren = Array.from(currentNode.childNodes);
  const incomingChildren = Array.from(incomingNode.childNodes);
  const max = Math.max(currentChildren.length, incomingChildren.length);

  for (let i = 0; i < max; i += 1) {
    const currentChild = currentChildren[i];
    const incomingChild = incomingChildren[i];
    if (!currentChild && incomingChild) {
      currentNode.appendChild(incomingChild.cloneNode(true));
      continue;
    }
    if (currentChild && !incomingChild) {
      currentChild.remove();
      continue;
    }
    if (currentChild && incomingChild) {
      patchNode(currentChild, incomingChild);
    }
  }
}

function patchInnerHTML(target: Element, nextHTML: string): void {
  const template = document.createElement("template");
  template.innerHTML = nextHTML;
  const incomingChildren = Array.from(template.content.childNodes);
  const existingChildren = Array.from(target.childNodes);
  const max = Math.max(existingChildren.length, incomingChildren.length);

  for (let i = 0; i < max; i += 1) {
    const currentChild = existingChildren[i];
    const incomingChild = incomingChildren[i];
    if (!currentChild && incomingChild) {
      target.appendChild(incomingChild.cloneNode(true));
      continue;
    }
    if (currentChild && !incomingChild) {
      currentChild.remove();
      continue;
    }
    if (currentChild && incomingChild) {
      patchNode(currentChild, incomingChild);
    }
  }
}

// Update the DOM with new content
async function updateDOM(data: PageData, pageContent: string): Promise<void> {
  // Update title
  if (data.title) {
    document.title = data.title;
  }

  // Try to update from outer-most to inner-most based on what's available
  const rootEl = document.querySelector("[data-gospa-root]");
  const contentEl = document.querySelector("[data-gospa-page-content]");
  const mainEl = document.querySelector("main");

  if (rootEl) {
    patchInnerHTML(rootEl, pageContent);
  } else if (contentEl) {
    patchInnerHTML(contentEl, pageContent);
  } else if (mainEl) {
    patchInnerHTML(mainEl, pageContent);
  } else {
    document.body.innerHTML = pageContent;
  }

  // Update head (managed head elements)
  runOnIdle(() => updateHead(data.head));

  // Re-initialize runtime for new content
  const targetEl = rootEl || contentEl || mainEl || document.body;
  await initNewContent(targetEl);

  // Update active links
  updateActiveLinks();

  // Focus management for accessibility
  const focusTarget = document.querySelector(
    "h1, [data-gospa-page-content], main",
  ) as HTMLElement;
  if (focusTarget) {
    focusTarget.tabIndex = -1;
    focusTarget.focus({ preventScroll: true });
  }
}

function updateActiveLinks() {
  const currentPath = window.location.pathname;
  document.querySelectorAll("a[href]").forEach((link) => {
    const href = link.getAttribute("href");
    if (
      href &&
      (href === currentPath || (href !== "/" && currentPath.startsWith(href)))
    ) {
      link.classList.add("gospa-active");
      link.setAttribute("aria-current", "page");
    } else {
      link.classList.remove("gospa-active");
      link.removeAttribute("aria-current");
    }
  });
}

function runOnIdle(callback: () => void): void {
  const idleCfg = navigationOptionsConfig.idleCallbackBatchUpdates;
  if (!idleCfg.enabled) {
    callback();
    return;
  }

  if ("requestIdleCallback" in window) {
    (window as any).requestIdleCallback(() => callback());
    return;
  }

  if (idleCfg.fallbackToMicrotask) {
    queueMicrotask(callback);
    return;
  }

  setTimeout(callback, 0);
}

// Update head elements - smart reconciliation to avoid CSS flashes
// and clean up elements that are no longer needed
function updateHead(headHtml: string): void {
  const escapeSelectorValue = (value: string): string => {
    if (typeof CSS !== "undefined" && typeof CSS.escape === "function") {
      return CSS.escape(value);
    }
    return value.replace(/["\\]/g, "\\$&");
  };

  // Parse head HTML to extract elements
  const parser = new DOMParser();
  const doc = parser.parseFromString(
    `<html><head>${headHtml}</head></html>`,
    "text/html",
  );
  const newHead = doc.querySelector("head");

  if (!newHead) return;

  // 1. Update title explicitly if it changed
  const newTitle = doc.querySelector("title")?.textContent;
  if (newTitle && newTitle !== document.title) {
    document.title = newTitle;
  }

  // Track which GoSPA-managed elements are still needed
  const neededSelectors = new Set<string>();

  // 2. Smart reconciliation for link tags (CSS)
  // Never remove existing stylesheets to avoid FOUC (Flash of Unstyled Content)
  const newLinkElements = Array.from(newHead.querySelectorAll("link"));

  newLinkElements.forEach((newEl) => {
    const href = newEl.getAttribute("href");
    const rel = newEl.getAttribute("rel");

    // Build a unique selector for tracking
    const selector = href
      ? `link[href="${escapeSelectorValue(href)}"]`
      : null;
    if (selector) neededSelectors.add(selector);

    // Check if this link already exists in the document
    const existingEl = selector ? document.head.querySelector(selector) : null;

    if (!existingEl) {
      // Only add if it doesn't exist
      const clone = newEl.cloneNode(true) as HTMLElement;
      clone.setAttribute("data-gospa-head", "true");
      document.head.appendChild(clone);
    }
  });

  // 3. Handle meta tags - update existing or add new
  const newMetaElements = Array.from(newHead.querySelectorAll("meta"));

  newMetaElements.forEach((newEl) => {
    const name = newEl.getAttribute("name");
    const property = newEl.getAttribute("property");
    const httpEquiv = newEl.getAttribute("http-equiv");

    // Build selector to find existing meta and for tracking
    let selector = "";
    if (name) selector = `meta[name="${escapeSelectorValue(name)}"]`;
    else if (property) selector = `meta[property="${escapeSelectorValue(property)}"]`;
    else if (httpEquiv) {
      selector = `meta[http-equiv="${escapeSelectorValue(httpEquiv)}"]`;
    }

    if (selector) neededSelectors.add(selector);

    const existingEl = selector ? document.head.querySelector(selector) : null;

    if (existingEl) {
      // Update content attribute only
      const content = newEl.getAttribute("content");
      if (content) existingEl.setAttribute("content", content);
    } else {
      // Add new meta tag
      const clone = newEl.cloneNode(true) as HTMLElement;
      clone.setAttribute("data-gospa-head", "true");
      document.head.appendChild(clone);
    }
  });

  // 4. Handle style tags - only add new ones, don't remove existing
  const newStyleElements = Array.from(newHead.querySelectorAll("style"));

  newStyleElements.forEach((newEl) => {
    const id = newEl.id;
    const selector = id ? `style#${id}` : null;

    if (selector) neededSelectors.add(selector);

    const existingEl = selector ? document.head.querySelector(selector) : null;

    if (!existingEl) {
      const clone = newEl.cloneNode(true) as HTMLElement;
      clone.setAttribute("data-gospa-head", "true");
      document.head.appendChild(clone);
    }
  });

  // 5. Handle scripts separately if marked
  newHead.querySelectorAll("script[data-gospa-head]").forEach((el) => {
    const src = el.getAttribute("src");
    const selector = src
      ? `script[src="${escapeSelectorValue(src)}"]`
      : `script`;

    neededSelectors.add(selector);

    const existingEl = src
      ? document.head.querySelector(
        `script[src="${escapeSelectorValue(src)}"]`,
      )
      : null;

    if (!existingEl) {
      const script = document.createElement("script");
      Array.from(el.attributes).forEach((attr) =>
        script.setAttribute(attr.name, attr.value),
      );
      script.textContent = el.textContent;
      document.head.appendChild(script);
    }
  });

  // 6. Clean up old GoSPA-managed head elements that are no longer needed
  // This prevents memory leaks and DOM bloat during long SPA sessions
  const existingGoSPAElements =
    document.head.querySelectorAll("[data-gospa-head]");
  existingGoSPAElements.forEach((el) => {
    let shouldRemove = true;

    // Check if this element matches any of the needed selectors
    for (const needed of neededSelectors) {
      if (el.matches(needed)) {
        shouldRemove = false;
        break;
      }
    }

    // For link and meta elements, also check by attribute patterns
    if (el.matches("link[href]")) {
      const href = el.getAttribute("href");
      if (href && neededSelectors.has(`link[href="${escapeSelectorValue(href)}"]`)) {
        shouldRemove = false;
      }
    } else if (el.matches("meta[name]")) {
      const name = el.getAttribute("name");
      if (name && neededSelectors.has(`meta[name="${escapeSelectorValue(name)}"]`)) {
        shouldRemove = false;
      }
    } else if (el.matches("meta[property]")) {
      const property = el.getAttribute("property");
      if (
        property &&
        neededSelectors.has(`meta[property="${escapeSelectorValue(property)}"]`)
      ) {
        shouldRemove = false;
      }
    } else if (el.matches("meta[http-equiv]")) {
      const httpEquiv = el.getAttribute("http-equiv");
      if (
        httpEquiv &&
        neededSelectors.has(
          `meta[http-equiv="${escapeSelectorValue(httpEquiv)}"]`,
        )
      ) {
        shouldRemove = false;
      }
    } else if (el.matches("style[id]")) {
      const id = el.id;
      if (id && neededSelectors.has(`style#${id}`)) {
        shouldRemove = false;
      }
    } else if (el.matches("script[data-gospa-head]")) {
      const src = el.getAttribute("src");
      if (src && neededSelectors.has(`script[src="${escapeSelectorValue(src)}"]`)) {
        shouldRemove = false;
      }
    }

    if (shouldRemove) {
      el.remove();
    }
  });
}

// Execute scripts in the given container
function executeScripts(container: Element | Document): void {
  const scripts = Array.from(container.querySelectorAll("script"));
  scripts.forEach((oldScript) => {
    // Skip scripts marked as permanent or already processed
    if (oldScript.closest("[data-gospa-permanent]")) return;
    if (
      navigationOptionsConfig.scriptExecution.executeMarkedOnly &&
      oldScript.getAttribute("data-gospa-exec") !== "true"
    ) return;

    const newScript = document.createElement("script");
    // Copy all attributes
    Array.from(oldScript.attributes).forEach((attr) => {
      newScript.setAttribute(attr.name, attr.value);
    });
    // Copy script content
    newScript.textContent = oldScript.textContent;

    // Replace old script with new one to trigger browser execution
    if (oldScript.parentNode) {
      oldScript.parentNode.replaceChild(newScript, oldScript);
    }
  });
}

// Initialize new content (re-run runtime setup)
async function initCriticalContent(
  container: Element | Document = document,
): Promise<void> {
  const eventElements = container.querySelectorAll("[data-on]");
  const gospa = (window as any).__gospa__;
  const ws = gospa?._ws;

  eventElements.forEach((element) => {
    const attr = element.getAttribute("data-on");
    if (!attr) return;

    const [eventType, action] = attr.split(":");
    if (!eventType || !action) return;

    const newElement = element.cloneNode(true) as Element;
    element.parentNode?.replaceChild(newElement, element);

    newElement.addEventListener(eventType, async () => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "action", action }));
        return;
      }

      const websocketModule = await import("./websocket.ts");
      websocketModule.sendAction(action);
    });
  });
}

async function initDeferredBindings(
  container: Element | Document = document,
): Promise<void> {
  const boundElements = container.querySelectorAll("[data-bind]");
  const gospa = (window as any).__gospa__;

  for (const element of boundElements) {
    const attr = element.getAttribute("data-bind");
    if (!attr) continue;

    const [bindingType, stateKey] = attr.split(":");
    if (!bindingType || !stateKey) continue;

    const rune = gospa?.state?.get(stateKey);
    if (!rune) continue;

    const update = async (value: any) => {
      switch (bindingType) {
        case "text":
          element.textContent = value;
          break;
        case "html":
          // FIX: Sanitize HTML content to prevent XSS
          element.innerHTML = await sanitizeHTML(value as string);
          break;
        case "value":
          (element as HTMLInputElement).value = value;
          break;
        case "checked":
          (element as HTMLInputElement).checked = value;
          break;
        case "show":
          (element as HTMLElement).style.display = value ? "" : "none";
          break;
      }
    };

    await update(rune.get());
    rune.subscribe((value: any) => update(value));
  }
}

async function initNewContent(
  container: Element = document.body,
): Promise<void> {
  // First, re-execute any scripts found in the new content
  executeScripts(container);

  await initCriticalContent(container);
  if (
    !navigationOptionsConfig.lazyRuntimeInitialization.enabled ||
    !navigationOptionsConfig.lazyRuntimeInitialization.deferBindings
  ) {
    await initDeferredBindings(container);
    return;
  }

  runOnIdle(() => {
    void initDeferredBindings(container);
  });
}

async function performDOMUpdateWithTransitions(
  data: PageData,
  options: NavigateOptions,
): Promise<void> {
  const viewCfg = navigationOptionsConfig.viewTransitions;
  const canTransition = viewCfg.enabled && "startViewTransition" in document;

  const pageContent = await prepareContent(data.content);

  const update = async () => {
    await updateDOM(data, pageContent);
    if (options.scrollToTop !== false) {
      window.scrollTo(0, 0);
    }
  };

  if (!canTransition) {
    await update();
    return;
  }

  // FIX: Wrap View Transition API in try/catch to handle browser compatibility issues
  try {
    const transition = (document as any).startViewTransition(update);
    await transition.finished;
  } catch (transitionError) {
    // Fallback to classic DOM update if View Transitions fail
    console.warn(
      "[GoSPA] View Transition failed, falling back to classic update:",
      transitionError,
    );
    await update();
  }
}

// Navigate to a new path
export async function navigate(
  path: string,
  options: NavigateOptions = {},
): Promise<boolean> {
  // Don't navigate if already at this path
  if (path === state.currentPath && !options.replace) {
    return false;
  }

  // Serialize navigations: chain onto the previous promise so concurrent
  // calls don't interleave. Each call awaits the previous one before starting,
  // eliminating the TOCTOU gap in the old nullable-pendingNavigation pattern.
  const previous = state.pendingNavigation ?? Promise.resolve(true);
  const current: Promise<boolean> = previous.then(async () => {
    // Re-check path after waiting — a preceding navigation may have already served it
    if (path === state.currentPath && !options.replace) {
      return false;
    }

    state.isNavigating = true;
    beforeNavCallbacks.forEach((cb) => cb(path));

    // Cancel previous fetch if any
    if (state.abortController) {
      state.abortController.abort();
    }
    state.abortController = new AbortController();

    try {
      // Save current scroll position before leaving
      scrollPositions.set(state.currentPath, window.scrollY);

      progressBar.start();
      const data = await getPageData(path, state.abortController.signal);

      if (!data) {
        progressBar.finish();
        window.location.href = path;
        return false;
      }

      if (options.replace) {
        window.history.replaceState({ path }, "", path);
      } else {
        window.history.pushState({ path }, "", path);
      }

      state.currentPath = path;
      await performDOMUpdateWithTransitions(data, options);

      progressBar.finish();
      afterNavCallbacks.forEach((cb) => cb(path));
      document.dispatchEvent(
        new CustomEvent("gospa:navigated", { detail: { path } }),
      );

      return true;
    } catch (error) {
      progressBar.finish();
      if ((error as Error).name === "AbortError") {
        return false;
      }
      // BUG FIX: Ensure navigation state is cleared on error
      // Otherwise isNavigating flag gets stuck, blocking future navigations
      console.error("[GoSPA] Navigation error:", error);
      state.isNavigating = false;
      state.pendingNavigation = null;
      return false;
    } finally {
      state.isNavigating = false;
      // Only clear the pending reference if it still points to this request
      if (state.pendingNavigation === current) {
        state.pendingNavigation = null;
      }
    }
  });

  // Store the chained promise so the next call can serialize onto it
  state.pendingNavigation = current;
  return current;
}

// Go back in history
export function back(): void {
  window.history.back();
}

// Go forward in history
export function forward(): void {
  window.history.forward();
}

// Go to specific position in history
export function go(delta: number): void {
  window.history.go(delta);
}

// Get current path
export function getCurrentPath(): string {
  return state.currentPath;
}

// Check if currently navigating
export function isNavigating(): boolean {
  return state.isNavigating;
}

// Handle popstate (back/forward button)
function handlePopState(event: PopStateEvent): void {
  const path = window.location.pathname;

  // Notify before navigation
  beforeNavCallbacks.forEach((cb) => cb(path));

  // Fetch and update
  progressBar.start();
  getPageData(path).then((data) => {
    if (data) {
      state.currentPath = path;
      performDOMUpdateWithTransitions(data, { scrollToTop: false }).then(() => {
        progressBar.finish();
        // Restore scroll position for historical paths
        const savedPos = scrollPositions.get(path);
        if (savedPos !== undefined) {
          window.scrollTo(0, savedPos);
        }
        afterNavCallbacks.forEach((cb) => cb(path));
        document.dispatchEvent(
          new CustomEvent("gospa:navigated", { detail: { path } }),
        );
      });
    } else {
      progressBar.finish();
      // Fallback to reload
      window.location.reload();
    }
  });
}

function getAnchorFromPath(path: EventTarget[]): HTMLAnchorElement | null {
  for (const target of path) {
    if (!(target instanceof Element)) continue;
    if (target instanceof HTMLAnchorElement && target.hasAttribute("href")) {
      return target;
    }
    const candidate = target.closest("a[href]");
    if (candidate instanceof HTMLAnchorElement) {
      return candidate;
    }
  }
  return null;
}

// Handle link clicks
function handleLinkClick(event: MouseEvent): void {
  if (
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  ) {
    return;
  }

  const path = (event.composedPath?.() ?? []) as EventTarget[];
  const link = getAnchorFromPath(path);
  if (!link) return;
  if (!isInternalLink(link)) return;

  event.preventDefault();
  const href = link.getAttribute("href");
  if (!href) return;
  void navigate(href);
}

function setupSpeculativePrefetching(): void {
  const cfg = navigationOptionsConfig.speculativePrefetching;
  if (!cfg.enabled) return;

  if ("IntersectionObserver" in window) {
    prefetchObserver?.disconnect();
    prefetchObserver = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          const anchor = entry.target as HTMLAnchorElement;
          const href = anchor.getAttribute("href");
          if (!href || !isInternalLink(anchor)) continue;
          void prefetch(href);
          prefetchObserver?.unobserve(anchor);
        }
      },
      { rootMargin: `${cfg.viewportMargin ?? 150}px` },
    );

    document.querySelectorAll("a[href]").forEach((anchor) => {
      if (anchor instanceof HTMLAnchorElement && isInternalLink(anchor)) {
        prefetchObserver?.observe(anchor);
      }
    });
  }

  window.addEventListener("mouseover", handleHoverPrefetch);
}

function handleHoverPrefetch(event: MouseEvent): void {
  const cfg = navigationOptionsConfig.speculativePrefetching;
  if (!cfg.enabled) return;
  const target = event.target;
  if (!(target instanceof Element)) return;
  const anchor = target.closest("a[href]");
  if (!(anchor instanceof HTMLAnchorElement) || !isInternalLink(anchor)) return;
  const href = anchor.getAttribute("href");
  if (!href) return;
  if (hoverPrefetchTimers.has(href)) return;
  const timer = window.setTimeout(
    () => {
      hoverPrefetchTimers.delete(href);
      void prefetch(href);
    },
    Math.max(0, cfg.hoverDelay ?? 60),
  );
  hoverPrefetchTimers.set(href, timer);
}

function teardownSpeculativePrefetching(): void {
  window.removeEventListener("mouseover", handleHoverPrefetch);
  prefetchObserver?.disconnect();
  prefetchObserver = null;
  for (const timer of hoverPrefetchTimers.values()) {
    clearTimeout(timer);
  }
  hoverPrefetchTimers.clear();
}

async function registerNavigationServiceWorker(): Promise<void> {
  const cfg = navigationOptionsConfig.serviceWorkerNavigationCaching;
  if (!cfg.enabled || !("serviceWorker" in navigator)) return;
  try {
    const path = cfg.path ?? "/gospa-navigation-sw.js";
    const swPath = cfg.cacheName
      ? `${path}?cacheName=${encodeURIComponent(cfg.cacheName)}`
      : path;
    await navigator.serviceWorker.register(swPath, { scope: "/" });
  } catch (error) {
    console.warn("[GoSPA] Service worker registration failed:", error);
  }
}

// Initialize navigation
export function initNavigation(): void {
  // Setup link click handler
  const root = document.querySelector(
    "[data-gospa-page-content], [data-gospa-root]",
  );
  clickDelegateContainer = root ?? document;
  clickDelegateContainer.addEventListener(
    "click",
    handleLinkClick as EventListener,
  );

  // Setup popstate handler
  window.addEventListener("popstate", handlePopState);

  // Check for global configuration
  const config = (window as any).__GOSPA_CONFIG__;
  if (config) {
    if (config.navigationOptions) {
      setNavigationOptions(config.navigationOptions);
    }
  }

  setupSpeculativePrefetching();
  void registerNavigationServiceWorker();

  // Mark as SPA-enabled
  document.documentElement.setAttribute("data-gospa-spa", "true");

  // Initial active link update
  updateActiveLinks();
}

// Cleanup navigation
export function destroyNavigation(): void {
  clickDelegateContainer.removeEventListener(
    "click",
    handleLinkClick as EventListener,
  );
  window.removeEventListener("popstate", handlePopState);
  teardownSpeculativePrefetching();
  document.documentElement.removeAttribute("data-gospa-spa");
}

// Prefetch a page for faster navigation
export async function prefetch(path: string): Promise<void> {
  // SECURITY: Validate that the path is internal before prefetching
  // This prevents SSRF-style attacks where an attacker could trigger
  // prefetch requests to internal services or external sites
  try {
    const url = new URL(path, window.location.origin);
    // Only allow same-origin prefetches
    if (url.origin !== window.location.origin) {
      console.debug("[GoSPA] Prefetch skipped: cross-origin URL:", path);
      return;
    }
    // Block potentially dangerous paths
    const normalizedPath = url.pathname;
    if (
      normalizedPath.startsWith("//") ||
      normalizedPath.startsWith("/..") ||
      normalizedPath.includes("/../")
    ) {
      console.debug("[GoSPA] Prefetch skipped: unsafe path:", path);
      return;
    }
  } catch {
    console.debug("[GoSPA] Prefetch skipped: invalid URL:", path);
    return;
  }

  const existing = prefetchCache.get(path);
  if (existing && existing.expiresAt > Date.now()) return;
  if (existing) prefetchCache.delete(path);

  const data = await fetchPageFromServer(path);
  if (data) {
    const ttl = Math.max(
      1000,
      navigationOptionsConfig.speculativePrefetching.ttl ?? 30000,
    );
    const expiresAt = Date.now() + ttl;
    prefetchCache.set(path, { data, expiresAt });
    setTimeout(() => {
      const current = prefetchCache.get(path);
      if (current && current.expiresAt <= Date.now()) {
        prefetchCache.delete(path);
      }
    }, ttl + 50);
  }
}

// Export navigation state as reactive
export function createNavigationState() {
  return {
    get path() {
      return state.currentPath;
    },
    get isNavigating() {
      return state.isNavigating;
    },
    navigate,
    back,
    forward,
    go,
    prefetch,
  };
}

// Auto-initialize when DOM is ready
if (typeof document !== "undefined") {
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initNavigation);
  } else {
    initNavigation();
  }
}

// Extend window type
declare global {
  interface Window {
    __gospa__?: {
      state: Map<string, any>;
      _ws?: WebSocket;
    };
    __GOSPA_CONFIG__?: {
      navigationOptions?: NavigationOptions;
    };
  }
}
