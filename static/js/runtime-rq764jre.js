import {
  Derived,
  Effect,
  Rune,
  StateMap,
  batch,
  effect,
  getCurrentEffect,
  getWebSocketClient,
  initWebSocket,
  sendAction,
  watch
} from "./websocket-g18v2mwh.js";
import {
  __require,
  __toESM
} from "./runtime-3hqyeswk.js";

// client/src/dom.ts
var defaultSanitizerUsed = false;
var sanitizeHtml = (html) => {
  if (!defaultSanitizerUsed) {
    console.warn(`[GoSPA] Security Warning: Using default pass-through HTML sanitizer for data-bind="html:*". For user-generated content, use 'gospa/runtime-secure' to enable DOMPurify.`);
    defaultSanitizerUsed = true;
  }
  return html;
};
var pendingDOMUpdates = [];
var rafScheduled = false;
var rafId = null;
function scheduleDOMUpdate(update) {
  pendingDOMUpdates.push(update);
  if (!rafScheduled) {
    rafScheduled = true;
    rafId = requestAnimationFrame(flushDOMUpdates);
  }
}
function flushDOMUpdates() {
  const updates = pendingDOMUpdates;
  pendingDOMUpdates = [];
  rafScheduled = false;
  rafId = null;
  for (const update of updates) {
    try {
      update();
    } catch (error) {
      console.error("[GoSPA] DOM update failed:", error);
    }
  }
}
function cancelPendingDOMUpdates() {
  if (rafId !== null) {
    cancelAnimationFrame(rafId);
    rafId = null;
  }
  pendingDOMUpdates = [];
  rafScheduled = false;
}
function flushDOMUpdatesNow() {
  if (rafScheduled) {
    if (rafId !== null) {
      cancelAnimationFrame(rafId);
      rafId = null;
    }
    flushDOMUpdates();
  }
}
function setSanitizer(fn) {
  sanitizeHtml = fn;
}
var bindings = new Map;
var elementBindings = new WeakMap;
var elementVersions = new WeakMap;
var bindingId = 0;
function nextBindingId() {
  return `binding-${++bindingId}`;
}
function registerBinding(binding) {
  const id = nextBindingId();
  if (!bindings.has(id)) {
    bindings.set(id, new Set);
  }
  bindings.get(id).add(binding);
  if (!elementBindings.has(binding.element)) {
    elementBindings.set(binding.element, new Set);
  }
  elementBindings.get(binding.element).add(binding);
  return id;
}
function unregisterBinding(id) {
  const bindingSet = bindings.get(id);
  if (bindingSet) {
    bindingSet.forEach((binding) => {
      const elemBindings = elementBindings.get(binding.element);
      if (elemBindings) {
        elemBindings.delete(binding);
        if (elemBindings.size === 0) {
          elementBindings.delete(binding.element);
        }
      }
    });
    bindings.delete(id);
  }
}
async function updateElement(binding, value) {
  const { element, type, attribute, transform } = binding;
  const transformedValue = transform ? transform(value) : value;
  const version = (elementVersions.get(element) || 0) + 1;
  elementVersions.set(element, version);
  scheduleDOMUpdate(() => {
    switch (type) {
      case "text":
        if (element instanceof HTMLElement || element instanceof SVGElement) {
          element.textContent = String(transformedValue ?? "");
        }
        break;
      case "html":
        if (element instanceof HTMLElement) {
          const htmlValue = String(transformedValue ?? "");
          const sanitized = sanitizeHtml(htmlValue);
          if (sanitized instanceof Promise) {
            sanitized.then((result) => {
              if (elementVersions.get(element) === version) {
                element.innerHTML = result;
              }
            }).catch((error) => {
              console.error("[GoSPA] HTML sanitization failed:", error);
            });
          } else {
            if (elementVersions.get(element) === version) {
              element.innerHTML = sanitized;
            }
          }
        }
        break;
      case "value":
        if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement) {
          if (element.value !== String(transformedValue ?? "")) {
            element.value = String(transformedValue ?? "");
          }
        }
        break;
      case "checked":
        if (element instanceof HTMLInputElement) {
          element.checked = Boolean(transformedValue);
        }
        break;
      case "class":
        if (element instanceof Element) {
          if (attribute) {
            if (transformedValue) {
              element.classList.add(attribute);
            } else {
              element.classList.remove(attribute);
            }
          } else if (typeof transformedValue === "string") {
            element.className = transformedValue;
          } else if (Array.isArray(transformedValue)) {
            element.className = transformedValue.join(" ");
          } else if (typeof transformedValue === "object" && transformedValue !== null) {
            Object.entries(transformedValue).forEach(([cls, enabled]) => {
              if (enabled) {
                element.classList.add(cls);
              } else {
                element.classList.remove(cls);
              }
            });
          }
        }
        break;
      case "style":
        if (element instanceof HTMLElement || element instanceof SVGElement) {
          if (attribute) {
            element.style[attribute] = String(transformedValue ?? "");
          } else if (typeof transformedValue === "string") {
            element.setAttribute("style", transformedValue);
          } else if (typeof transformedValue === "object" && transformedValue !== null) {
            Object.entries(transformedValue).forEach(([prop, val]) => {
              element.style[prop] = val;
            });
          }
        }
        break;
      case "attr":
        if (attribute) {
          if (transformedValue === null || transformedValue === undefined || transformedValue === false) {
            element.removeAttribute(attribute);
          } else if (transformedValue === true) {
            element.setAttribute(attribute, "");
          } else {
            element.setAttribute(attribute, String(transformedValue));
          }
        }
        break;
      case "prop":
        if (attribute && element instanceof HTMLElement) {
          element[attribute] = transformedValue;
        }
        break;
    }
  });
}
function bindElement(element, rune, options = {}) {
  const binding = {
    type: options.type || "text",
    key: options.key || "",
    element,
    attribute: options.attribute,
    transform: options.transform
  };
  const id = registerBinding(binding);
  updateElement(binding, rune.get()).catch((error) => {
    console.error("[GoSPA] Binding update failed:", error);
  });
  const unsubscribe = rune.subscribe((value) => {
    updateElement(binding, value).catch((error) => {
      console.error("[GoSPA] Binding update failed:", error);
    });
  });
  return () => {
    unsubscribe();
    unregisterBinding(id);
  };
}
function bindDerived(element, derived, options = {}) {
  const binding = {
    type: options.type || "text",
    key: options.key || "",
    element,
    attribute: options.attribute,
    transform: options.transform
  };
  const id = registerBinding(binding);
  updateElement(binding, derived.get()).catch((error) => {
    console.error("[GoSPA] Binding update failed:", error);
  });
  const unsubscribe = derived.subscribe((value) => {
    updateElement(binding, value).catch((error) => {
      console.error("[GoSPA] Binding update failed:", error);
    });
  });
  return () => {
    unsubscribe();
    unregisterBinding(id);
  };
}
function bindTwoWay(element, rune) {
  const isNumber = element instanceof HTMLInputElement && element.type === "number";
  const isCheckbox = element instanceof HTMLInputElement && element.type === "checkbox";
  if (isCheckbox) {
    element.checked = Boolean(rune.get());
  } else {
    element.value = String(rune.get() ?? "");
  }
  const unsubscribe = rune.subscribe((value) => {
    if (isCheckbox) {
      element.checked = Boolean(value);
    } else {
      if (element.value !== String(value ?? "")) {
        element.value = String(value ?? "");
      }
    }
  });
  const inputHandler = () => {
    let newValue;
    if (isCheckbox) {
      newValue = element.checked;
    } else if (isNumber) {
      newValue = element.value ? parseFloat(element.value) : 0;
    } else {
      newValue = element.value;
    }
    batch(() => {
      rune.set(newValue);
    });
  };
  element.addEventListener("input", inputHandler);
  element.addEventListener("change", inputHandler);
  return () => {
    unsubscribe();
    element.removeEventListener("input", inputHandler);
    element.removeEventListener("change", inputHandler);
  };
}
function querySelectorAll(selector) {
  return document.querySelectorAll(selector);
}
function find(selector) {
  return document.querySelector(selector);
}
function findAll(selector) {
  return Array.from(document.querySelectorAll(selector));
}
function addClass(el, ...classes) {
  el.classList.add(...classes);
}
function removeClass(el, ...classes) {
  el.classList.remove(...classes);
}
function hasClass(el, cls) {
  return el.classList.contains(cls);
}
function attr(el, name, value) {
  if (value === undefined) {
    return el.getAttribute(name);
  }
  el.setAttribute(name, value);
}
function data(el, name, value) {
  if (value === undefined) {
    return el.dataset[name];
  }
  el.dataset[name] = value;
}
function createElement(tag, attrs = {}, children) {
  const element = document.createElement(tag);
  Object.entries(attrs).forEach(([key, value]) => {
    if (key.startsWith("on") && typeof value === "function") {
      const eventName = key.slice(2).toLowerCase();
      element.addEventListener(eventName, value);
    } else if (key === "class") {
      if (typeof value === "string") {
        element.className = value;
      } else if (Array.isArray(value)) {
        element.className = value.join(" ");
      } else if (typeof value === "object" && value !== null) {
        Object.entries(value).forEach(([cls, enabled]) => {
          if (enabled)
            element.classList.add(cls);
        });
      }
    } else if (key === "style" && typeof value === "object") {
      Object.entries(value).forEach(([prop, val]) => {
        element.style[prop] = val;
      });
    } else if (value instanceof Rune) {
      bindElement(element, value, { type: "attr", attribute: key });
    } else {
      element.setAttribute(key, String(value));
    }
  });
  if (children) {
    children.forEach((child) => {
      if (typeof child === "string") {
        element.appendChild(document.createTextNode(child));
      } else {
        element.appendChild(child);
      }
    });
  }
  return element;
}
function renderIf(condition, trueRender, falseRender) {
  let current = null;
  const update = (value) => {
    if (value) {
      if (!current) {
        current = trueRender();
      }
    } else {
      if (current && falseRender) {
        current = falseRender();
      } else {
        current = null;
      }
    }
  };
  update(condition.get());
  const unsubscribe = condition.subscribe(update);
  return {
    element: current,
    cleanup: () => {
      unsubscribe();
    }
  };
}
function renderList(items, render, getKey) {
  const container = document.createDocumentFragment();
  const containerElement = document.createElement("div");
  container.appendChild(containerElement);
  const itemMap = new Map;
  const update = (newItems) => {
    const newKeys = new Set;
    newItems.forEach((item, index) => {
      const key = getKey(item, index);
      newKeys.add(key);
      if (!itemMap.has(key)) {
        const element = render(item, index);
        itemMap.set(key, { element, index });
        const refNode = containerElement.children[index] || null;
        containerElement.insertBefore(element, refNode);
      } else {
        const existing = itemMap.get(key);
        existing.index = index;
        if (containerElement.children[index] !== existing.element) {
          containerElement.insertBefore(existing.element, containerElement.children[index] || null);
        }
      }
    });
    itemMap.forEach((value, key) => {
      if (!newKeys.has(key)) {
        value.element.remove();
        itemMap.delete(key);
      }
    });
  };
  update(items.get());
  const unsubscribe = items.subscribe(update);
  return {
    container: containerElement,
    cleanup: () => {
      unsubscribe();
      itemMap.clear();
    }
  };
}

// client/src/events.ts
var modifiers = {
  prevent: (event, handler) => {
    event.preventDefault();
    return handler(event);
  },
  stop: (event, handler) => {
    event.stopPropagation();
    return handler(event);
  },
  capture: (event, handler) => handler(event),
  once: (event, handler) => handler(event),
  passive: (event, handler) => handler(event),
  self: (event, handler) => {
    if (event.target === event.currentTarget) {
      return handler(event);
    }
  }
};
var listenerRegistry = new WeakMap;
function createWrappedHandler(handler, mods) {
  return (event) => {
    for (const mod of mods) {
      if (mod === "capture" || mod === "once" || mod === "passive") {
        continue;
      }
      const modHandler = modifiers[mod];
      modHandler(event, handler);
    }
    const activeMods = mods.filter((m) => !["capture", "once", "passive"].includes(m));
    if (activeMods.length === 0) {
      return handler(event);
    }
  };
}
function parseEventString(eventStr) {
  const parts = eventStr.split(":");
  const event = parts[0];
  const mods = parts.slice(1);
  return { event, modifiers: mods };
}
function on(target, eventStr, handler) {
  const { event, modifiers: mods } = parseEventString(eventStr);
  const options = {
    capture: mods.includes("capture"),
    once: mods.includes("once"),
    passive: mods.includes("passive")
  };
  const wrappedHandler = createWrappedHandler(handler, mods);
  target.addEventListener(event, wrappedHandler, options);
  if (!listenerRegistry.has(target)) {
    listenerRegistry.set(target, new Map);
  }
  const targetMap = listenerRegistry.get(target);
  if (!targetMap.has(eventStr)) {
    targetMap.set(eventStr, new Set);
  }
  targetMap.get(eventStr).add(wrappedHandler);
  return () => {
    target.removeEventListener(event, wrappedHandler, options);
    const set = targetMap.get(eventStr);
    if (set) {
      set.delete(wrappedHandler);
      if (set.size === 0) {
        targetMap.delete(eventStr);
      }
    }
  };
}
function offAll(target) {
  const targetMap = listenerRegistry.get(target);
  if (!targetMap)
    return;
  for (const [eventStr, listeners] of targetMap) {
    const { event, modifiers: mods } = parseEventString(eventStr);
    const options = {
      capture: mods.includes("capture")
    };
    for (const listener of listeners) {
      target.removeEventListener(event, listener, options);
    }
  }
  listenerRegistry.delete(target);
}
function debounce(handler, wait) {
  let timeoutId = null;
  const cancel = () => {
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
  };
  const debouncedHandler = (event) => {
    cancel();
    timeoutId = setTimeout(() => {
      handler(event);
      timeoutId = null;
    }, wait);
  };
  return { handler: debouncedHandler, cancel };
}
function throttle(handler, limit) {
  let inThrottle = false;
  let lastEvent = null;
  const cancel = () => {
    inThrottle = false;
    lastEvent = null;
  };
  const throttledHandler = (event) => {
    if (!inThrottle) {
      handler(event);
      inThrottle = true;
      setTimeout(() => {
        inThrottle = false;
        if (lastEvent) {
          handler(lastEvent);
          lastEvent = null;
        }
      }, limit);
    } else {
      lastEvent = event;
    }
  };
  return { handler: throttledHandler, cancel };
}
function bindEvent(target, eventStr, rune, transformer) {
  return on(target, eventStr, (event) => {
    const value = transformer(event);
    rune.set(value);
  });
}
var transformers = {
  value: (event) => event.target.value,
  checked: (event) => event.target.checked,
  numberValue: (event) => Number(event.target.value),
  files: (event) => event.target.files,
  formData: (event) => {
    event.preventDefault();
    return new FormData(event.target);
  }
};
function delegate(root, selector, eventStr, handler) {
  const { event, modifiers: mods } = parseEventString(eventStr);
  const wrappedHandler = createWrappedHandler(handler, mods);
  const delegatedHandler = (e) => {
    const target = e.target;
    const matched = target.closest(selector);
    if (matched) {
      wrappedHandler(e);
    }
  };
  const options = {
    capture: mods.includes("capture"),
    passive: mods.includes("passive")
  };
  root.addEventListener(event, delegatedHandler, options);
  return () => {
    root.removeEventListener(event, delegatedHandler, options);
  };
}
function onKey(keys, handler, options) {
  const keyArray = Array.isArray(keys) ? keys : [keys];
  return (event) => {
    if (keyArray.includes(event.key)) {
      if (options?.preventDefault) {
        event.preventDefault();
      }
      handler(event);
    }
  };
}
var keys = {
  enter: "Enter",
  escape: "Escape",
  tab: "Tab",
  space: " ",
  arrowUp: "ArrowUp",
  arrowDown: "ArrowDown",
  arrowLeft: "ArrowLeft",
  arrowRight: "ArrowRight"
};
function setupEventDelegation(root) {
  const events = [
    "click",
    "input",
    "change",
    "submit",
    "focusin",
    "focusout",
    "mouseenter",
    "mouseleave"
  ];
  events.forEach((eventName) => {
    root.addEventListener(eventName, (event) => {
      let target = event.target;
      while (target && target !== root) {
        const attr2 = target.getAttribute("data-gospa-on");
        if (attr2) {
          const [eventStr, handlerName] = attr2.split(":");
          if (eventStr === eventName || eventStr === "focus" && eventName === "focusin" || eventStr === "blur" && eventName === "focusout") {
            const islandEl = target.closest("[data-gospa-island]");
            if (islandEl) {
              const islandId = islandEl.id;
              const islandInstance = window[`__GOSPA_ISLAND_${islandId}__`];
              if (islandInstance && islandInstance.handlers && islandInstance.handlers[handlerName]) {
                islandInstance.handlers[handlerName](event);
              }
            }
          }
        }
        target = target.parentElement;
      }
    }, { passive: eventName !== "submit" });
  });
}

// client/src/navigation.ts
var state = {
  currentPath: window.location.pathname,
  isNavigating: false,
  pendingNavigation: null,
  abortController: null
};
var beforeNavCallbacks = new Set;
var afterNavCallbacks = new Set;
function onBeforeNavigate(cb) {
  beforeNavCallbacks.add(cb);
  return () => beforeNavCallbacks.delete(cb);
}
function onAfterNavigate(cb) {
  afterNavCallbacks.add(cb);
  return () => afterNavCallbacks.delete(cb);
}
var DEFAULT_NAVIGATION_OPTIONS = {
  speculativePrefetching: {
    enabled: true,
    ttl: 30000,
    hoverDelay: 50,
    viewportMargin: 150
  },
  urlParsingCache: {
    enabled: true,
    maxSize: 100,
    ttl: 30000
  },
  idleCallbackBatchUpdates: {
    enabled: true,
    fallbackToMicrotask: true
  },
  lazyRuntimeInitialization: {
    enabled: true,
    deferBindings: true
  },
  serviceWorkerNavigationCaching: {
    enabled: false,
    cacheName: "gospa-navigation-cache",
    path: "/gospa-navigation-sw.js"
  },
  viewTransitions: {
    enabled: true,
    fallbackToClassic: true
  },
  progressBar: {
    enabled: true,
    color: "#3b82f6",
    height: "2px"
  },
  scriptExecution: {
    executeMarkedOnly: true
  }
};
var navigationOptionsConfig = {
  ...DEFAULT_NAVIGATION_OPTIONS,
  speculativePrefetching: {
    ...DEFAULT_NAVIGATION_OPTIONS.speculativePrefetching
  },
  urlParsingCache: { ...DEFAULT_NAVIGATION_OPTIONS.urlParsingCache },
  idleCallbackBatchUpdates: {
    ...DEFAULT_NAVIGATION_OPTIONS.idleCallbackBatchUpdates
  },
  lazyRuntimeInitialization: {
    ...DEFAULT_NAVIGATION_OPTIONS.lazyRuntimeInitialization
  },
  serviceWorkerNavigationCaching: {
    ...DEFAULT_NAVIGATION_OPTIONS.serviceWorkerNavigationCaching
  },
  viewTransitions: { ...DEFAULT_NAVIGATION_OPTIONS.viewTransitions },
  progressBar: { ...DEFAULT_NAVIGATION_OPTIONS.progressBar },
  scriptExecution: { ...DEFAULT_NAVIGATION_OPTIONS.scriptExecution }
};
var parsedURLCache = new Map;
var hoverPrefetchTimers = new Map;
var prefetchObserver = null;
var pendingRequests = new Map;
var clickDelegateContainer = document;
function setNavigationOptions(config) {
  if (config.urlParsingCache?.enabled === false && navigationOptionsConfig.urlParsingCache.enabled) {
    parsedURLCache.clear();
  }
  if (config.speculativePrefetching?.enabled === false && navigationOptionsConfig.speculativePrefetching.enabled) {
    prefetchCache.clear();
  }
  navigationOptionsConfig = {
    ...navigationOptionsConfig,
    speculativePrefetching: {
      ...navigationOptionsConfig.speculativePrefetching,
      ...config.speculativePrefetching ?? {}
    },
    urlParsingCache: {
      ...navigationOptionsConfig.urlParsingCache,
      ...config.urlParsingCache ?? {}
    },
    idleCallbackBatchUpdates: {
      ...navigationOptionsConfig.idleCallbackBatchUpdates,
      ...config.idleCallbackBatchUpdates ?? {}
    },
    lazyRuntimeInitialization: {
      ...navigationOptionsConfig.lazyRuntimeInitialization,
      ...config.lazyRuntimeInitialization ?? {}
    },
    serviceWorkerNavigationCaching: {
      ...navigationOptionsConfig.serviceWorkerNavigationCaching,
      ...config.serviceWorkerNavigationCaching ?? {}
    },
    viewTransitions: {
      ...navigationOptionsConfig.viewTransitions,
      ...config.viewTransitions ?? {}
    },
    progressBar: {
      ...navigationOptionsConfig.progressBar,
      ...config.progressBar ?? {}
    },
    scriptExecution: {
      ...navigationOptionsConfig.scriptExecution,
      ...config.scriptExecution ?? {}
    }
  };
}

class ProgressBar {
  el = null;
  interval = null;
  progress = 0;
  start() {
    if (!navigationOptionsConfig.progressBar.enabled)
      return;
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
      boxShadow: `0 0 10px ${cfg.color ?? "#3b82f6"}`
    });
    document.body.appendChild(this.el);
    this.progress = 0;
    this.interval = window.setInterval(() => {
      if (this.progress < 90) {
        this.progress += (90 - this.progress) * 0.1;
        if (this.el)
          this.el.style.width = `${this.progress}%`;
      }
    }, 100);
  }
  finish() {
    if (!this.el)
      return;
    clearInterval(this.interval);
    this.el.style.width = "100%";
    setTimeout(() => {
      if (this.el) {
        this.el.style.opacity = "0";
        setTimeout(() => this.reset(), 300);
      }
    }, 100);
  }
  reset() {
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
var progressBar = new ProgressBar;
var scrollPositions = new Map;
function getCachedURL(href) {
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
  let parsed;
  try {
    parsed = new URL(href, window.location.origin);
  } catch {
    return null;
  }
  parsedURLCache.set(href, {
    url: parsed,
    expiresAt: now + Math.max(1000, cacheCfg.ttl ?? 30000)
  });
  while (parsedURLCache.size > Math.max(1, cacheCfg.maxSize ?? 100)) {
    const first = parsedURLCache.keys().next().value;
    if (!first)
      break;
    parsedURLCache.delete(first);
  }
  return parsed;
}
function isInternalLink(link) {
  const href = link.getAttribute("href");
  if (!href || href.startsWith("#") || href.startsWith("javascript:") || href.startsWith("mailto:") || href.startsWith("tel:") || href.startsWith("sms:") || href.startsWith("blob:") || href.startsWith("data:")) {
    return false;
  }
  const urlObj = getCachedURL(href);
  if (!urlObj) {
    return false;
  }
  if (urlObj.origin !== window.location.origin) {
    return false;
  }
  if (link.hasAttribute("data-gospa-reload") || link.hasAttribute("data-external") || link.hasAttribute("download") || link.getAttribute("target") === "_blank") {
    return false;
  }
  if (link.hasAttribute("data-gospa-link")) {
    return true;
  }
  const pathname = urlObj.pathname;
  const lastSegment = pathname.slice(pathname.lastIndexOf("/") + 1);
  const dotIndex = lastSegment.lastIndexOf(".");
  if (dotIndex !== -1 && dotIndex < lastSegment.length - 1) {
    const ext = lastSegment.slice(dotIndex + 1).toLowerCase();
    if (ext !== "html" && ext !== "htm") {
      return false;
    }
  }
  return true;
}
var prefetchCache = new Map;
async function fetchPageFromServer(path, signal) {
  const existing = pendingRequests.get(path);
  if (existing) {
    return existing;
  }
  const request = (async () => {
    try {
      const response = await fetch(path, {
        signal,
        headers: {
          "X-Requested-With": "GoSPA-Navigate",
          Accept: "text/html"
        }
      });
      if (!response.ok) {
        console.error("[GoSPA] Navigation failed:", response.status);
        return null;
      }
      const contentType = response.headers.get("content-type");
      if (contentType && !contentType.includes("text/html")) {
        console.warn(`[GoSPA] Intercepted non-HTML response (${contentType}) for path ${path}. Falling back to standard navigation.`);
        return null;
      }
      const html = await response.text();
      const parser = new DOMParser;
      const doc = parser.parseFromString(html, "text/html");
      let content;
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
      const title = doc.querySelector("title")?.textContent || "";
      const headEl = doc.querySelector("head");
      const head = headEl ? headEl.innerHTML : "";
      return { content, title, head };
    } catch (error) {
      console.error("[GoSPA] Navigation error:", error);
      return null;
    } finally {
      pendingRequests.delete(path);
    }
  })();
  pendingRequests.set(path, request);
  return request;
}
async function getPageData(path, signal) {
  const cached = prefetchCache.get(path);
  if (cached && cached.expiresAt > Date.now()) {
    prefetchCache.delete(path);
    prefetchCache.set(path, cached);
    return cached.data;
  }
  if (cached)
    prefetchCache.delete(path);
  return fetchPageFromServer(path, signal);
}
async function prepareContent(html) {
  return html;
}
var DOMPurify = null;
async function sanitizeHTML(html) {
  if (DOMPurify != null) {
    return DOMPurify(html);
  }
  const globalPurify = window.DOMPurify;
  if (globalPurify != null) {
    DOMPurify = globalPurify;
    return globalPurify(html);
  }
  return html;
}
function patchAttributes(current, incoming) {
  for (const attr2 of Array.from(current.attributes)) {
    if (!incoming.hasAttribute(attr2.name)) {
      current.removeAttribute(attr2.name);
    }
  }
  for (const attr2 of Array.from(incoming.attributes)) {
    if (current.getAttribute(attr2.name) !== attr2.value) {
      current.setAttribute(attr2.name, attr2.value);
    }
  }
}
function patchNode(currentNode, incomingNode) {
  if (currentNode.isEqualNode(incomingNode)) {
    return;
  }
  if (currentNode.nodeType !== incomingNode.nodeType) {
    currentNode.parentNode?.replaceChild(incomingNode.cloneNode(true), currentNode);
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
    currentNode.parentNode?.replaceChild(incomingNode.cloneNode(true), currentNode);
    return;
  }
  if (currentNode.id && currentNode.id !== incomingNode.id || incomingNode.id && currentNode.id !== incomingNode.id || currentNode.getAttribute("data-gospa-page") !== incomingNode.getAttribute("data-gospa-page")) {
    currentNode.parentNode?.replaceChild(incomingNode.cloneNode(true), currentNode);
    return;
  }
  if (currentNode.hasAttribute("data-gospa-permanent")) {
    return;
  }
  patchAttributes(currentNode, incomingNode);
  const currentChildren = Array.from(currentNode.childNodes);
  const incomingChildren = Array.from(incomingNode.childNodes);
  const max = Math.max(currentChildren.length, incomingChildren.length);
  for (let i = 0;i < max; i += 1) {
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
function patchInnerHTML(target, nextHTML) {
  const template = document.createElement("template");
  template.innerHTML = nextHTML;
  const incomingChildren = Array.from(template.content.childNodes);
  const existingChildren = Array.from(target.childNodes);
  const max = Math.max(existingChildren.length, incomingChildren.length);
  for (let i = 0;i < max; i += 1) {
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
async function updateDOM(data2, pageContent) {
  if (data2.title) {
    document.title = data2.title;
  }
  const rootEl = document.querySelector("[data-gospa-root]");
  const contentEl = document.querySelector("[data-gospa-page-content]");
  const mainEl = document.querySelector("main");
  const container = rootEl || contentEl || mainEl || document.body;
  container.removeAttribute("data-gospa-loading");
  if (rootEl) {
    patchInnerHTML(rootEl, pageContent);
  } else if (contentEl) {
    patchInnerHTML(contentEl, pageContent);
  } else if (mainEl) {
    patchInnerHTML(mainEl, pageContent);
  } else {
    document.body.innerHTML = pageContent;
  }
  runOnIdle(() => updateHead(data2.head));
  const targetEl = rootEl || contentEl || mainEl || document.body;
  await initNewContent(targetEl);
  runOnIdle(() => {
    updateActiveLinks();
  });
  const focusTarget = document.querySelector("h1, [data-gospa-page-content], main");
  if (focusTarget) {
    focusTarget.tabIndex = -1;
    focusTarget.focus({ preventScroll: true });
  }
}
function updateActiveLinks() {
  const currentPath = window.location.pathname;
  document.querySelectorAll("a[href]").forEach((link) => {
    const href = link.getAttribute("href");
    if (href && (href === currentPath || href !== "/" && currentPath.startsWith(href))) {
      link.classList.add("gospa-active");
      link.setAttribute("aria-current", "page");
    } else {
      link.classList.remove("gospa-active");
      link.removeAttribute("aria-current");
    }
  });
}
function runOnIdle(callback) {
  const idleCfg = navigationOptionsConfig.idleCallbackBatchUpdates;
  if (!idleCfg.enabled) {
    callback();
    return;
  }
  if ("requestIdleCallback" in window) {
    window.requestIdleCallback(() => callback());
    return;
  }
  if (idleCfg.fallbackToMicrotask) {
    queueMicrotask(callback);
    return;
  }
  setTimeout(callback, 0);
}
function updateHead(headHtml) {
  const escapeSelectorValue = (value) => {
    if (typeof CSS !== "undefined" && typeof CSS.escape === "function") {
      return CSS.escape(value);
    }
    return value.replace(/["\\]/g, "\\$&");
  };
  const parser = new DOMParser;
  const doc = parser.parseFromString(`<html><head>${headHtml}</head></html>`, "text/html");
  const newHead = doc.querySelector("head");
  if (!newHead)
    return;
  const newTitle = doc.querySelector("title")?.textContent;
  if (newTitle && newTitle !== document.title) {
    document.title = newTitle;
  }
  const neededSelectors = new Set;
  const newLinkElements = Array.from(newHead.querySelectorAll("link"));
  newLinkElements.forEach((newEl) => {
    const href = newEl.getAttribute("href");
    const selector = href ? `link[href="${escapeSelectorValue(href)}"]` : null;
    if (selector)
      neededSelectors.add(selector);
    const existingEl = selector ? document.head.querySelector(selector) : null;
    if (!existingEl) {
      const clone = newEl.cloneNode(true);
      clone.setAttribute("data-gospa-head", "true");
      document.head.appendChild(clone);
    }
  });
  const newMetaElements = Array.from(newHead.querySelectorAll("meta"));
  newMetaElements.forEach((newEl) => {
    const name = newEl.getAttribute("name");
    const property = newEl.getAttribute("property");
    const httpEquiv = newEl.getAttribute("http-equiv");
    let selector = "";
    if (name)
      selector = `meta[name="${escapeSelectorValue(name)}"]`;
    else if (property)
      selector = `meta[property="${escapeSelectorValue(property)}"]`;
    else if (httpEquiv) {
      selector = `meta[http-equiv="${escapeSelectorValue(httpEquiv)}"]`;
    }
    if (selector)
      neededSelectors.add(selector);
    const existingEl = selector ? document.head.querySelector(selector) : null;
    if (existingEl) {
      const content = newEl.getAttribute("content");
      if (content)
        existingEl.setAttribute("content", content);
    } else {
      const clone = newEl.cloneNode(true);
      clone.setAttribute("data-gospa-head", "true");
      document.head.appendChild(clone);
    }
  });
  const newStyleElements = Array.from(newHead.querySelectorAll("style"));
  newStyleElements.forEach((newEl) => {
    const id = newEl.id;
    const selector = id ? `style#${id}` : null;
    if (selector)
      neededSelectors.add(selector);
    const existingEl = selector ? document.head.querySelector(selector) : null;
    if (!existingEl) {
      const clone = newEl.cloneNode(true);
      clone.setAttribute("data-gospa-head", "true");
      document.head.appendChild(clone);
    }
  });
  newHead.querySelectorAll("script[data-gospa-head]").forEach((el) => {
    const src = el.getAttribute("src");
    const selector = src ? `script[src="${escapeSelectorValue(src)}"]` : `script`;
    neededSelectors.add(selector);
    const existingEl = src ? document.head.querySelector(`script[src="${escapeSelectorValue(src)}"]`) : null;
    if (!existingEl) {
      const script = document.createElement("script");
      Array.from(el.attributes).forEach((attr2) => script.setAttribute(attr2.name, attr2.value));
      script.textContent = el.textContent;
      document.head.appendChild(script);
    }
  });
  const existingGoSPAElements = document.head.querySelectorAll("[data-gospa-head]");
  existingGoSPAElements.forEach((el) => {
    let shouldRemove = true;
    for (const needed of neededSelectors) {
      if (el.matches(needed)) {
        shouldRemove = false;
        break;
      }
    }
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
      if (property && neededSelectors.has(`meta[property="${escapeSelectorValue(property)}"]`)) {
        shouldRemove = false;
      }
    } else if (el.matches("meta[http-equiv]")) {
      const httpEquiv = el.getAttribute("http-equiv");
      if (httpEquiv && neededSelectors.has(`meta[http-equiv="${escapeSelectorValue(httpEquiv)}"]`)) {
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
function executeScripts(container) {
  const scripts = Array.from(container.querySelectorAll("script"));
  scripts.forEach((oldScript) => {
    if (oldScript.closest("[data-gospa-permanent]"))
      return;
    if (oldScript.getAttribute("data-gospa-exec") !== "true") {
      return;
    }
    const newScript = document.createElement("script");
    Array.from(oldScript.attributes).forEach((attr2) => {
      newScript.setAttribute(attr2.name, attr2.value);
    });
    newScript.textContent = oldScript.textContent;
    if (oldScript.parentNode) {
      oldScript.parentNode.replaceChild(newScript, oldScript);
    }
  });
}
async function initCriticalContent(container = document) {
  const eventElements = container.querySelectorAll("[data-on]");
  const gospa = window.__gospa__;
  const ws = gospa?._ws;
  eventElements.forEach((element) => {
    const attr2 = element.getAttribute("data-on");
    if (!attr2)
      return;
    const [eventType, action] = attr2.split(":");
    if (!eventType || !action)
      return;
    const newElement = element.cloneNode(true);
    element.parentNode?.replaceChild(newElement, element);
    newElement.addEventListener(eventType, async () => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "action", action }));
        return;
      }
      const websocketModule = await import("./websocket-g18v2mwh.js");
      websocketModule.sendAction(action);
    });
  });
}
async function initDeferredBindings(container = document) {
  const boundElements = container.querySelectorAll("[data-bind]");
  const gospa = window.__gospa__;
  for (const element of boundElements) {
    const attr2 = element.getAttribute("data-bind");
    if (!attr2)
      continue;
    const [bindingType, stateKey] = attr2.split(":");
    if (!bindingType || !stateKey)
      continue;
    const rune = gospa?.state?.get(stateKey);
    if (!rune)
      continue;
    const update = async (value) => {
      switch (bindingType) {
        case "text":
          element.textContent = value;
          break;
        case "html":
          element.innerHTML = await sanitizeHTML(value);
          break;
        case "value":
          element.value = value;
          break;
        case "checked":
          element.checked = value;
          break;
        case "show":
          element.style.display = value ? "" : "none";
          break;
      }
    };
    await update(rune.get());
    rune.subscribe((value) => update(value));
  }
}
async function initNewContent(container = document.body) {
  executeScripts(container);
  await initCriticalContent(container);
  if (!navigationOptionsConfig.lazyRuntimeInitialization.enabled || !navigationOptionsConfig.lazyRuntimeInitialization.deferBindings) {
    await initDeferredBindings(container);
    return;
  }
  runOnIdle(() => {
    initDeferredBindings(container);
  });
}
async function performDOMUpdateWithTransitions(data2, options) {
  const viewCfg = navigationOptionsConfig.viewTransitions;
  const canTransition = viewCfg.enabled && "startViewTransition" in document;
  const pageContent = await prepareContent(data2.content);
  const update = async () => {
    await updateDOM(data2, pageContent);
    if (options.scrollToTop !== false) {
      window.scrollTo(0, 0);
    }
  };
  if (!canTransition) {
    await update();
    return;
  }
  try {
    const transition = document.startViewTransition(update);
    await transition.finished;
  } catch (transitionError) {
    console.warn("[GoSPA] View Transition failed, falling back to classic update:", transitionError);
    await update();
  }
}
async function navigate(path, options = {}) {
  if (path === state.currentPath && !options.replace) {
    return false;
  }
  const previous = state.pendingNavigation ?? Promise.resolve(true);
  const current = previous.then(async () => {
    if (path === state.currentPath && !options.replace) {
      return false;
    }
    state.isNavigating = true;
    beforeNavCallbacks.forEach((cb) => cb(path));
    if (state.abortController) {
      state.abortController.abort();
    }
    state.abortController = new AbortController;
    try {
      if (options.replace) {
        window.history.replaceState({ path }, "", path);
      } else {
        window.history.pushState({ path }, "", path);
      }
      updateActiveLinks();
      const container = document.querySelector("[data-gospa-page-content], [data-gospa-root]") || document.body;
      container.setAttribute("data-gospa-loading", "true");
      scrollPositions.set(state.currentPath, window.scrollY);
      progressBar.start();
      const data2 = await getPageData(path, state.abortController.signal);
      if (!data2) {
        progressBar.finish();
        container.removeAttribute("data-gospa-loading");
        window.location.href = path;
        return false;
      }
      state.currentPath = path;
      await performDOMUpdateWithTransitions(data2, options);
      progressBar.finish();
      afterNavCallbacks.forEach((cb) => cb(path));
      document.dispatchEvent(new CustomEvent("gospa:navigated", { detail: { path } }));
      return true;
    } catch (error) {
      progressBar.finish();
      const container = document.querySelector("[data-gospa-page-content], [data-gospa-root]") || document.body;
      container.removeAttribute("data-gospa-loading");
      if (error.name === "AbortError") {
        return false;
      }
      console.error("[GoSPA] Navigation error:", error);
      state.isNavigating = false;
      state.pendingNavigation = null;
      return false;
    } finally {
      state.isNavigating = false;
      if (state.pendingNavigation === current) {
        state.pendingNavigation = null;
      }
    }
  });
  state.pendingNavigation = current;
  return current;
}
function back() {
  window.history.back();
}
function forward() {
  window.history.forward();
}
function go(delta) {
  window.history.go(delta);
}
function getCurrentPath() {
  return state.currentPath;
}
function isNavigating() {
  return state.isNavigating;
}
function handlePopState(_event) {
  const path = window.location.pathname;
  updateActiveLinks();
  const container = document.querySelector("[data-gospa-page-content], [data-gospa-root]") || document.body;
  container.setAttribute("data-gospa-loading", "true");
  beforeNavCallbacks.forEach((cb) => cb(path));
  if (state.abortController) {
    state.abortController.abort();
  }
  state.abortController = new AbortController;
  progressBar.start();
  getPageData(path, state.abortController.signal).then((data2) => {
    if (data2) {
      state.currentPath = path;
      performDOMUpdateWithTransitions(data2, { scrollToTop: false }).then(() => {
        progressBar.finish();
        const savedPos = scrollPositions.get(path);
        if (savedPos !== undefined) {
          window.scrollTo(0, savedPos);
        }
        afterNavCallbacks.forEach((cb) => cb(path));
        document.dispatchEvent(new CustomEvent("gospa:navigated", { detail: { path } }));
      });
    } else {
      progressBar.finish();
      container.removeAttribute("data-gospa-loading");
      window.location.reload();
    }
  }).catch((error) => {
    if (error.name === "AbortError")
      return;
    progressBar.finish();
    container.removeAttribute("data-gospa-loading");
    console.error("[GoSPA] Popstate navigation error:", error);
  });
}
function getAnchorFromPath(path) {
  for (const target of path) {
    if (!(target instanceof Element))
      continue;
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
function handleLinkClick(event) {
  if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
    return;
  }
  const path = event.composedPath?.() ?? [];
  const link = getAnchorFromPath(path);
  if (!link)
    return;
  if (!isInternalLink(link))
    return;
  event.preventDefault();
  const href = link.getAttribute("href");
  if (!href)
    return;
  navigate(href);
}
function shouldPrefetch() {
  const connection = navigator.connection;
  if (!connection)
    return true;
  if (connection.saveData)
    return false;
  if (connection.effectiveType === "slow-2g" || connection.effectiveType === "2g") {
    return false;
  }
  return true;
}
function setupSpeculativePrefetching() {
  const cfg = navigationOptionsConfig.speculativePrefetching;
  if (!cfg.enabled || !shouldPrefetch())
    return;
  if ("IntersectionObserver" in window) {
    prefetchObserver?.disconnect();
    prefetchObserver = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (!entry.isIntersecting)
          continue;
        const anchor = entry.target;
        const href = anchor.getAttribute("href");
        if (!href || !isInternalLink(anchor))
          continue;
        prefetch(href);
        prefetchObserver?.unobserve(anchor);
      }
    }, { rootMargin: `${cfg.viewportMargin ?? 150}px` });
    document.querySelectorAll("a[href]").forEach((anchor) => {
      if (anchor instanceof HTMLAnchorElement && isInternalLink(anchor)) {
        prefetchObserver?.observe(anchor);
      }
    });
  }
  window.addEventListener("mouseover", handleHoverPrefetch);
}
function handleHoverPrefetch(event) {
  const cfg = navigationOptionsConfig.speculativePrefetching;
  if (!cfg.enabled)
    return;
  const target = event.target;
  if (!(target instanceof Element))
    return;
  const anchor = target.closest("a[href]");
  if (!(anchor instanceof HTMLAnchorElement) || !isInternalLink(anchor))
    return;
  const href = anchor.getAttribute("href");
  if (!href)
    return;
  if (hoverPrefetchTimers.has(href))
    return;
  const timer = window.setTimeout(() => {
    hoverPrefetchTimers.delete(href);
    prefetch(href);
  }, Math.max(0, cfg.hoverDelay ?? 60));
  hoverPrefetchTimers.set(href, timer);
}
function teardownSpeculativePrefetching() {
  window.removeEventListener("mouseover", handleHoverPrefetch);
  prefetchObserver?.disconnect();
  prefetchObserver = null;
  for (const timer of hoverPrefetchTimers.values()) {
    clearTimeout(timer);
  }
  hoverPrefetchTimers.clear();
}
async function registerNavigationServiceWorker() {
  const cfg = navigationOptionsConfig.serviceWorkerNavigationCaching;
  if (!cfg.enabled || !("serviceWorker" in navigator))
    return;
  try {
    const path = cfg.path ?? "/gospa-navigation-sw.js";
    const swPath = cfg.cacheName ? `${path}?cacheName=${encodeURIComponent(cfg.cacheName)}` : path;
    await navigator.serviceWorker.register(swPath, { scope: "/" });
  } catch (error) {
    console.warn("[GoSPA] Service worker registration failed:", error);
  }
}
function initNavigation() {
  const root = document.querySelector("[data-gospa-page-content], [data-gospa-root]");
  clickDelegateContainer = root ?? document;
  clickDelegateContainer.addEventListener("click", handleLinkClick);
  window.addEventListener("popstate", handlePopState);
  const config = window.__GOSPA_CONFIG__;
  if (config) {
    if (config.navigationOptions) {
      setNavigationOptions(config.navigationOptions);
    }
  }
  setupSpeculativePrefetching();
  registerNavigationServiceWorker();
  document.documentElement.setAttribute("data-gospa-spa", "true");
  updateActiveLinks();
}
function destroyNavigation() {
  clickDelegateContainer.removeEventListener("click", handleLinkClick);
  window.removeEventListener("popstate", handlePopState);
  teardownSpeculativePrefetching();
  document.documentElement.removeAttribute("data-gospa-spa");
}
async function prefetch(path) {
  try {
    const url = new URL(path, window.location.origin);
    if (url.origin !== window.location.origin) {
      console.debug("[GoSPA] Prefetch skipped: cross-origin URL:", path);
      return;
    }
    const normalizedPath = url.pathname;
    if (normalizedPath.startsWith("//") || normalizedPath.startsWith("/..") || normalizedPath.includes("/../")) {
      console.debug("[GoSPA] Prefetch skipped: unsafe path:", path);
      return;
    }
  } catch {
    console.debug("[GoSPA] Prefetch skipped: invalid URL:", path);
    return;
  }
  const existing = prefetchCache.get(path);
  if (existing && existing.expiresAt > Date.now())
    return;
  if (existing)
    prefetchCache.delete(path);
  const data2 = await fetchPageFromServer(path);
  if (data2) {
    const ttl = Math.max(1000, navigationOptionsConfig.speculativePrefetching.ttl ?? 30000);
    const expiresAt = Date.now() + ttl;
    prefetchCache.set(path, { data: data2, expiresAt });
    setTimeout(() => {
      const current = prefetchCache.get(path);
      if (current && current.expiresAt <= Date.now()) {
        prefetchCache.delete(path);
      }
    }, ttl + 50);
  }
}
function createNavigationState() {
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
    prefetch
  };
}
if (typeof document !== "undefined") {
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initNavigation);
  } else {
    initNavigation();
  }
}

// client/src/remote.ts
var remotePrefix = "/_gospa/remote";
function getCookie(name) {
  if (typeof document === "undefined")
    return;
  const cookie = document.cookie.split("; ").find((row) => row.startsWith(`${name}=`));
  return cookie ? decodeURIComponent(cookie.split("=").slice(1).join("=")) : undefined;
}
function configureRemote(options) {
  if (options.prefix) {
    remotePrefix = options.prefix;
  }
}
function getRemotePrefix() {
  return remotePrefix;
}
async function remote(name, input, options = {}) {
  const url = `${remotePrefix}/${encodeURIComponent(name)}`;
  const timeout = options.timeout ?? 30000;
  const externalSignal = options.signal;
  const forbiddenHeaders = ["x-csrf-token", "content-type", "accept"];
  if (options.headers) {
    for (const key of Object.keys(options.headers)) {
      if (forbiddenHeaders.includes(key.toLowerCase())) {
        return {
          error: `Invalid custom header: ${key}`,
          code: "INVALID_HEADER",
          status: 0,
          ok: false
        };
      }
    }
  }
  if (externalSignal?.aborted) {
    return {
      error: "Request aborted",
      code: "NETWORK_ERROR",
      status: 0,
      ok: false
    };
  }
  const controller = new AbortController;
  const csrfToken = typeof window !== "undefined" && window.__GOSPA_CONFIG__?.csrfToken || getCookie("csrf_token");
  let abortListener;
  if (externalSignal) {
    abortListener = () => controller.abort();
    externalSignal.addEventListener("abort", abortListener);
  }
  let timeoutId;
  const timeoutPromise = new Promise((_, reject) => {
    timeoutId = setTimeout(() => {
      controller.abort();
      reject(new Error("__GOSPA_TIMEOUT__"));
    }, timeout);
  });
  try {
    const response = await Promise.race([
      fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          ...csrfToken ? { "X-CSRF-Token": csrfToken } : {},
          ...options.headers
        },
        body: input !== undefined ? JSON.stringify(input) : undefined,
        signal: controller.signal,
        credentials: "same-origin"
      }),
      timeoutPromise
    ]);
    if (timeoutId !== undefined)
      clearTimeout(timeoutId);
    let data2;
    let error;
    let code;
    const contentType = response.headers.get("content-type");
    if (contentType?.includes("application/json")) {
      try {
        const json = await response.json();
        code = json.code;
        if (!response.ok) {
          error = json.error || `HTTP ${response.status}`;
        } else {
          data2 = json.data !== undefined ? json.data : json;
        }
      } catch (parseErr) {
        error = parseErr instanceof Error ? `Invalid JSON: ${parseErr.message}` : "Invalid JSON response";
        code = "PARSE_ERROR";
      }
    } else if (!response.ok) {
      error = `HTTP ${response.status}: ${response.statusText}`;
      code = "HTTP_ERROR";
    }
    return {
      data: data2,
      error,
      code,
      status: response.status,
      ok: response.ok
    };
  } catch (err) {
    if (timeoutId !== undefined)
      clearTimeout(timeoutId);
    if (err instanceof Error && err.message === "__GOSPA_TIMEOUT__") {
      return {
        error: "Request timeout",
        code: "TIMEOUT",
        status: 0,
        ok: false
      };
    }
    if (err instanceof Error) {
      if (err.name === "AbortError") {
        return {
          error: externalSignal?.aborted ? "Request aborted" : err.message,
          code: "NETWORK_ERROR",
          status: 0,
          ok: false
        };
      }
      return {
        error: err.message,
        code: "NETWORK_ERROR",
        status: 0,
        ok: false
      };
    }
    return {
      error: "Unknown error",
      code: "UNKNOWN_ERROR",
      status: 0,
      ok: false
    };
  } finally {
    if (abortListener && externalSignal) {
      externalSignal.removeEventListener("abort", abortListener);
    }
  }
}
function remoteAction(name) {
  return (input, options) => {
    return remote(name, input, options);
  };
}
if (typeof window !== "undefined") {
  window.__GOSPA_REMOTE__ = {
    remote,
    remoteAction,
    configureRemote,
    getRemotePrefix
  };
}

// client/src/signals.ts
var REACTIVE_SYMBOL = Symbol("gospa-reactive");
var RAW_SYMBOL = Symbol("gospa-raw");
function reactive(initial) {
  if (initial && initial[REACTIVE_SYMBOL]) {
    return initial;
  }
  const rawValues = new Map;
  const runes = new Map;
  const subscribers = new Map;
  for (const key of Object.keys(initial)) {
    rawValues.set(key, initial[key]);
    runes.set(key, new Rune(initial[key]));
  }
  const handler = {
    get(target, prop, receiver) {
      if (prop === REACTIVE_SYMBOL)
        return true;
      if (prop === RAW_SYMBOL)
        return Object.fromEntries(rawValues);
      const currentEffect = getCurrentEffect();
      if (currentEffect) {
        const rune2 = runes.get(prop);
        if (rune2) {
          currentEffect.addDependency(rune2);
        }
      }
      const rune = runes.get(prop);
      if (rune) {
        return rune.get();
      }
      const value = Reflect.get(target, prop, receiver);
      if (typeof value === "function") {
        return value.bind(receiver);
      }
      return value;
    },
    set(target, prop, value, _receiver) {
      if (prop === REACTIVE_SYMBOL || prop === RAW_SYMBOL) {
        return false;
      }
      const oldValue = rawValues.get(prop);
      if (Object.is(oldValue, value)) {
        return true;
      }
      rawValues.set(prop, value);
      let rune = runes.get(prop);
      if (rune) {
        rune.set(value);
      } else {
        rune = new Rune(value);
        runes.set(prop, rune);
      }
      const propSubscribers = subscribers.get(prop);
      if (propSubscribers) {
        batch(() => {
          propSubscribers.forEach((fn) => fn());
        });
      }
      return true;
    },
    has(target, prop) {
      if (prop === REACTIVE_SYMBOL || prop === RAW_SYMBOL) {
        return true;
      }
      return rawValues.has(prop) || Reflect.has(target, prop);
    },
    ownKeys(_target) {
      return Array.from(rawValues.keys()).filter((k) => typeof k === "string");
    },
    getOwnPropertyDescriptor(target, prop) {
      if (rawValues.has(prop)) {
        return {
          enumerable: true,
          configurable: true,
          value: rawValues.get(prop)
        };
      }
      return Reflect.getOwnPropertyDescriptor(target, prop);
    }
  };
  const proxy = new Proxy(initial, handler);
  return proxy;
}
function $state(initial) {
  if (typeof initial === "object" && initial !== null) {
    return reactive(initial);
  }
  return new Rune(initial);
}
function derived(compute) {
  const derivedInstance = new Derived(compute);
  return () => derivedInstance.get();
}
function $derived(compute) {
  return derived(compute);
}
function effect2(fn) {
  const effectInstance = new Effect(fn);
  return () => effectInstance.dispose();
}
function $effect(fn) {
  return effect2(fn);
}
function watchProp(obj, prop, callback) {
  if (!obj[REACTIVE_SYMBOL]) {
    throw new Error("watchProp requires a reactive object created with reactive()");
  }
  const derivedProp = new Derived(() => obj[prop]);
  return derivedProp.subscribe((newVal, oldVal) => {
    callback(newVal, oldVal);
  });
}
function toRaw(obj) {
  if (!obj[REACTIVE_SYMBOL]) {
    return obj;
  }
  return obj[RAW_SYMBOL];
}
function isReactive(obj) {
  return obj != null && typeof obj === "object" && obj[REACTIVE_SYMBOL] === true;
}
function reactiveArray(initial) {
  const proxy = reactive(initial);
  const arrayMethods = [
    "push",
    "pop",
    "shift",
    "unshift",
    "splice",
    "sort",
    "reverse"
  ];
  for (const method of arrayMethods) {
    const original = Array.prototype[method];
    proxy[method] = function(...args) {
      const result = original.apply(this, args);
      proxy.__version = Date.now();
      return result;
    };
  }
  return proxy;
}

// client/src/store.ts
class SharedStore {
  static instance;
  stores = new Map;
  constructor() {}
  static getInstance() {
    if (!SharedStore.instance) {
      SharedStore.instance = new SharedStore;
    }
    return SharedStore.instance;
  }
  create(name, initialValue) {
    if (this.stores.has(name)) {
      return this.stores.get(name);
    }
    const store = $state(initialValue);
    this.stores.set(name, store);
    this.updateDevTools();
    return store;
  }
  get(name) {
    return this.stores.get(name);
  }
  has(name) {
    return this.stores.has(name);
  }
  list() {
    return Array.from(this.stores.keys());
  }
  updateDevTools() {
    if (typeof window !== "undefined") {
      const debug = window.__GOSPA_CONFIG__?.debug;
      if (!debug)
        return;
      if (!window.__GOSPA_STORES_TRACKER__) {
        Object.defineProperty(window, "__GOSPA_STORES__", {
          get: () => Object.fromEntries(this.stores),
          configurable: true,
          enumerable: true
        });
        window.__GOSPA_STORES_TRACKER__ = true;
      }
    }
  }
}
function createStore(name, initialValue) {
  return SharedStore.getInstance().create(name, initialValue);
}
function getStore(name) {
  return SharedStore.getInstance().get(name);
}

// client/src/error-boundary.ts
var errorBoundaries = new Map;
var globalErrorHandlers = new Set;
function onComponentError(handler) {
  globalErrorHandlers.add(handler);
  return () => globalErrorHandlers.delete(handler);
}
function withErrorBoundary(componentId, config) {
  if (!errorBoundaries.has(componentId)) {
    errorBoundaries.set(componentId, {
      hasError: false,
      error: null,
      retryCount: 0
    });
  }
  const getState = () => errorBoundaries.get(componentId);
  const handleError = (error) => {
    const state2 = getState();
    state2.hasError = true;
    state2.error = error;
    config.onError?.(error, componentId);
    for (const handler of globalErrorHandlers) {
      try {
        handler(error, componentId);
      } catch (handlerError) {
        console.error("[GoSPA] Error in error handler:", handlerError);
      }
    }
    const element = document.querySelector(`[data-gospa-component="${componentId}"]`);
    if (element) {
      const fallbackEl = typeof config.fallback === "function" ? config.fallback(error, componentId) : config.fallback.cloneNode(true);
      element.replaceChildren(fallbackEl);
      if (config.retryable && state2.retryCount < (config.maxRetries ?? 3)) {
        const retryBtn = document.createElement("button");
        retryBtn.textContent = "Retry";
        retryBtn.className = "gospa-retry-btn";
        retryBtn.onclick = () => {
          state2.retryCount++;
          state2.hasError = false;
          state2.error = null;
          element.dispatchEvent(new CustomEvent("gospa:retry", { detail: { componentId } }));
        };
        element.appendChild(retryBtn);
      }
    }
  };
  const wrapMount = (mountFn) => {
    return () => {
      const state2 = getState();
      if (state2.hasError) {
        return () => {};
      }
      try {
        return mountFn();
      } catch (error) {
        handleError(error);
        return () => {};
      }
    };
  };
  const wrapDestroy = (destroyFn) => {
    return () => {
      try {
        destroyFn();
      } catch (error) {
        console.error(`[GoSPA] Error destroying component ${componentId}:`, error);
      }
    };
  };
  const wrapAction = (actionFn) => {
    return (...args) => {
      const state2 = getState();
      if (state2.hasError) {
        throw new Error(`Component ${componentId} is in error state: ${state2.error?.message}`);
      }
      try {
        return actionFn(...args);
      } catch (error) {
        handleError(error);
        throw error;
      }
    };
  };
  const clearError = () => {
    const state2 = getState();
    state2.hasError = false;
    state2.error = null;
    state2.retryCount = 0;
  };
  return {
    wrapMount,
    wrapDestroy,
    wrapAction,
    clearError,
    getState
  };
}
function createErrorFallback(message) {
  const el = document.createElement("div");
  el.className = "gospa-error-fallback";
  el.setAttribute("role", "alert");
  const content = document.createElement("div");
  content.className = "gospa-error-content";
  const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  icon.setAttribute("class", "gospa-error-icon");
  icon.setAttribute("viewBox", "0 0 24 24");
  icon.setAttribute("fill", "none");
  icon.setAttribute("stroke", "currentColor");
  icon.setAttribute("stroke-width", "2");
  const circle = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  circle.setAttribute("cx", "12");
  circle.setAttribute("cy", "12");
  circle.setAttribute("r", "10");
  const line1 = document.createElementNS("http://www.w3.org/2000/svg", "line");
  line1.setAttribute("x1", "12");
  line1.setAttribute("y1", "8");
  line1.setAttribute("x2", "12");
  line1.setAttribute("y2", "12");
  const line2 = document.createElementNS("http://www.w3.org/2000/svg", "line");
  line2.setAttribute("x1", "12");
  line2.setAttribute("y1", "16");
  line2.setAttribute("x2", "12.01");
  line2.setAttribute("y2", "16");
  icon.appendChild(circle);
  icon.appendChild(line1);
  icon.appendChild(line2);
  const text = document.createElement("p");
  text.className = "gospa-error-message";
  text.textContent = message || "Something went wrong";
  content.appendChild(icon);
  content.appendChild(text);
  el.appendChild(content);
  return el;
}
function getErrorBoundaryState(componentId) {
  return errorBoundaries.get(componentId);
}
function clearAllErrorBoundaries() {
  for (const state2 of errorBoundaries.values()) {
    state2.hasError = false;
    state2.error = null;
    state2.retryCount = 0;
  }
}
function isInErrorState(componentId) {
  return errorBoundaries.get(componentId)?.hasError ?? false;
}

// client/src/runtime-core.ts
var components = new Map;
var globalState = new StateMap;
var setupFunctions = new Map;
function getSetup(name) {
  const local = setupFunctions.get(name);
  if (local)
    return local;
  const globalSetups = window.__GOSPA_SETUPS__;
  if (globalSetups && typeof globalSetups[name] === "function") {
    return globalSetups[name];
  }
  return;
}
var isInitialized = false;
var config = {};
var featuresModule = null;
function init(userConfig = {}) {
  if (isInitialized) {
    if (Object.keys(userConfig).length > 0) {
      config = { ...config, ...userConfig };
    }
    return;
  }
  isInitialized = true;
  config = { ...config, ...userConfig };
  if (config.wsUrl) {
    featuresModule = import("./framework-features-00yexptx.js").then((mod) => {
      const ws = mod.initWebSocket({
        url: config.wsUrl,
        onMessage: handleServerMessage,
        serializationFormat: config.serializationFormat
      });
      ws.connect().catch((err) => {
        if (config.onConnectionError) {
          config.onConnectionError(err);
        } else if (config.debug) {
          console.error("WebSocket connection failed:", err);
        }
      });
      return mod;
    });
  }
  if (typeof window !== "undefined") {
    window.addEventListener("error", (event) => {
      if (config.debug)
        console.error("Runtime error:", event.error);
    });
  }
}
var GoSPA = {
  get config() {
    return config;
  },
  components,
  globalState,
  init,
  createComponent,
  destroyComponent,
  getComponent,
  getState,
  setState,
  callAction,
  bind,
  autoInit,
  remote,
  remoteAction,
  configureRemote,
  getRemotePrefix,
  get Rune() {
    return Rune;
  },
  get Derived() {
    return Derived;
  },
  get Effect() {
    return Effect;
  },
  get StateMap() {
    return StateMap;
  },
  batch,
  effect,
  watch,
  get on() {
    return on;
  },
  get offAll() {
    return offAll;
  },
  get debounce() {
    return debounce;
  },
  get throttle() {
    return throttle;
  },
  get sanitizeHtml() {
    return sanitizeHtml;
  },
  $state,
  $derived,
  $effect,
  createStore,
  getStore,
  createIsland,
  initWebSocket,
  getWebSocketClient,
  sendAction,
  navigate,
  back,
  prefetch
};
if (typeof window !== "undefined") {
  window.GoSPA = GoSPA;
  window.__GOSPA__ = GoSPA;
}
function createComponent(id, name) {
  if (components.has(id))
    return components.get(id);
  const instance = {
    id,
    name,
    states: new StateMap,
    elements: new Set,
    dispose: () => {
      instance.states.dispose();
      instance.elements.clear();
      components.delete(id);
    }
  };
  components.set(id, instance);
  return instance;
}
function destroyComponent(id) {
  const component = components.get(id);
  if (component)
    component.dispose();
}
function getComponent(id) {
  return components.get(id);
}
function getState(componentId, key) {
  const component = components.get(componentId);
  if (!component)
    return;
  const rune2 = component.states.get(key);
  return rune2 ? rune2.get() : undefined;
}
function setState(componentId, key, value) {
  const component = components.get(componentId);
  if (component) {
    component.states.set(key, value);
  }
}
function callAction(name, input) {
  return remote(name, input);
}
function bind(componentId, element, property, key, options = {}) {
  const component = components.get(componentId);
  if (!component)
    return () => {};
  component.elements.add(element);
  let rune2 = component.states.get(key);
  if (!rune2) {
    const container = element.closest("[data-gospa-state]");
    if (container) {
      try {
        const initialState = JSON.parse(container.getAttribute("data-gospa-state") || "{}");
        if (initialState[key] !== undefined) {
          rune2 = component.states.set(key, initialState[key]);
        }
      } catch (e) {}
    }
    if (!rune2)
      rune2 = component.states.set(key, undefined);
  }
  if (options.twoWay) {
    return bindTwoWay(element, rune2);
  }
  return bindElement(element, rune2, {
    type: property,
    transform: options.transformer
  });
}
function createIsland(id, name) {
  const instance = createComponent(id, name);
  const root = document.querySelector(`[data-gospa-component="${name}"][id="${id}"]`);
  if (root) {
    autoBindIsland(id, root);
  }
  return instance;
}
function autoBindIsland(componentId, root) {
  const elements = root.querySelectorAll("[data-gospa-bind], [data-model]");
  for (const el of elements) {
    const element = el;
    const bindAttr = element.getAttribute("data-gospa-bind");
    if (bindAttr) {
      const [prop, key2] = bindAttr.split(":");
      bind(componentId, element, prop, key2);
      continue;
    }
    const key = element.getAttribute("data-model");
    if (key)
      bind(componentId, element, "value", key, { twoWay: true });
  }
}
function handleServerMessage(message) {
  switch (message.type) {
    case "init":
      if (message.componentId && message.data) {
        const component = components.get(message.componentId);
        if (component)
          component.states.fromJSON(message.data);
      } else if (message.state) {
        const stateObj = message.state;
        for (const [scopedKey, value] of Object.entries(stateObj)) {
          const dotIndex = scopedKey.indexOf(".");
          if (dotIndex > 0) {
            const componentId = scopedKey.substring(0, dotIndex);
            const stateKey = scopedKey.substring(dotIndex + 1);
            const component = components.get(componentId);
            if (component)
              component.states.set(stateKey, value);
          } else {
            for (const component of components.values()) {
              if (component.states.get(scopedKey) !== undefined) {
                component.states.set(scopedKey, value);
              }
            }
            globalState.set(scopedKey, value);
          }
        }
      }
      break;
    case "patch":
      if (message.patch) {
        globalState.fromJSON(message.patch);
      }
      break;
    case "update":
      if (message.componentId && message.diff) {
        const component = components.get(message.componentId);
        if (component)
          component.states.fromJSON(message.diff);
      }
      break;
    case "sync":
      if (message.data) {
        globalState.fromJSON(message.data);
      } else if (message.key !== undefined && message.value !== undefined) {
        const scopedKey = message.key;
        const componentId = message.componentId;
        if (componentId) {
          const component = components.get(componentId);
          if (component)
            component.states.set(scopedKey, message.value);
        } else {
          globalState.set(scopedKey, message.value);
        }
      }
      break;
    case "error":
      if (config.debug)
        console.error("Server error:", message.error);
      break;
  }
}
function autoInit() {
  const componentRoots = document.querySelectorAll("[data-gospa-component]");
  componentRoots.forEach((root) => {
    const el = root;
    const name = el.getAttribute("data-gospa-component");
    const id = el.id || `c-${Math.random().toString(36).substring(2, 9)}`;
    if (!el.id)
      el.id = id;
    const instance = createComponent(id, name);
    const stateData = el.getAttribute("data-gospa-state");
    if (stateData) {
      try {
        instance.states.fromJSON(JSON.parse(stateData));
      } catch (e) {
        if (config.debug)
          console.error("Error parsing initial state for", name, e);
      }
    }
    autoBindIsland(id, el);
  });
  const islandRoots = document.querySelectorAll("[data-gospa-island]");
  islandRoots.forEach((root) => {
    const el = root;
    const name = el.getAttribute("data-gospa-island");
    if (!name)
      return;
    let setup = setupFunctions.get(name);
    if (!setup) {
      const globalSetups = window.__GOSPA_SETUPS__;
      if (globalSetups && typeof globalSetups[name] === "function") {
        setup = globalSetups[name];
      }
    }
    if (setup) {
      try {
        let stateData = {};
        const stateAttr = el.getAttribute("data-gospa-state");
        if (stateAttr) {
          try {
            stateData = JSON.parse(stateAttr);
          } catch {}
        }
        let propsData = {};
        const propsAttr = el.getAttribute("data-gospa-props");
        if (propsAttr) {
          try {
            propsData = JSON.parse(propsAttr);
          } catch {}
        }
        setup(el, propsData, stateData);
      } catch (e) {
        if (config.debug)
          console.error("Error initializing island", name, e);
      }
    } else if (config.debug) {
      console.warn("No setup function registered for island:", name);
    }
  });
}
async function getFrameworkFeatures() {
  if (!featuresModule)
    featuresModule = import("./framework-features-00yexptx.js");
  return featuresModule;
}
async function getWebSocket() {
  return getFrameworkFeatures();
}
async function getNavigation() {
  return getFrameworkFeatures();
}
async function getTransitions() {
  return getFrameworkFeatures();
}
if (typeof document !== "undefined") {
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => {
      if (document.documentElement.hasAttribute("data-gospa-auto"))
        autoInit();
    });
  } else if (document.documentElement.hasAttribute("data-gospa-auto")) {
    autoInit();
  }
}
function registerNavigationCleanup() {
  if (typeof window === "undefined")
    return;
  getFrameworkFeatures().then((mod) => {
    mod.onBeforeNavigate(() => {
      for (const [id] of components) {
        destroyComponent(id);
      }
      globalState.clear();
      mod.getIslandManager()?.destroy();
    });
    document.addEventListener("gospa:navigated", () => {
      mod.getIslandManager()?.discoverIslands();
    });
  }).catch(() => {});
}
if (typeof window !== "undefined") {
  registerNavigationCleanup();
}

export { sanitizeHtml, cancelPendingDOMUpdates, flushDOMUpdatesNow, setSanitizer, registerBinding, unregisterBinding, bindElement, bindDerived, bindTwoWay, querySelectorAll, find, findAll, addClass, removeClass, hasClass, attr, data, createElement, renderIf, renderList, parseEventString, on, offAll, debounce, throttle, bindEvent, transformers, delegate, onKey, keys, setupEventDelegation, onBeforeNavigate, onAfterNavigate, setNavigationOptions, navigate, back, forward, go, getCurrentPath, isNavigating, initNavigation, destroyNavigation, prefetch, createNavigationState, configureRemote, getRemotePrefix, remote, remoteAction, reactive, $state, derived, $derived, effect2 as effect, $effect, watchProp, toRaw, isReactive, reactiveArray, SharedStore, createStore, getStore, onComponentError, withErrorBoundary, createErrorFallback, getErrorBoundaryState, clearAllErrorBoundaries, isInErrorState, getSetup, init, createComponent, destroyComponent, getComponent, getState, setState, callAction, bind, autoInit, getFrameworkFeatures, getWebSocket, getNavigation, getTransitions };
