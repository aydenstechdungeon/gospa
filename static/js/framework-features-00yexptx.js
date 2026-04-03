import {
  PURIFY_CONFIG,
  domPurifySanitizer,
  isSanitizerReady,
  preloadSanitizer,
  sanitize,
  sanitizeSync
} from "./runtime-e8vb8rnf.js";
import {
  $derived,
  $effect,
  $state,
  SharedStore,
  addClass,
  attr,
  back,
  bindDerived,
  bindElement,
  bindEvent,
  bindTwoWay,
  cancelPendingDOMUpdates,
  clearAllErrorBoundaries,
  configureRemote,
  createElement,
  createErrorFallback,
  createNavigationState,
  createStore,
  data,
  debounce,
  delegate,
  derived,
  destroyNavigation,
  effect,
  find,
  findAll,
  flushDOMUpdatesNow,
  forward,
  getCurrentPath,
  getErrorBoundaryState,
  getRemotePrefix,
  getSetup,
  getStore,
  go,
  hasClass,
  initNavigation,
  isInErrorState,
  isNavigating,
  isReactive,
  keys,
  navigate,
  offAll,
  on,
  onAfterNavigate,
  onBeforeNavigate,
  onComponentError,
  onKey,
  parseEventString,
  prefetch,
  querySelectorAll,
  reactive,
  reactiveArray,
  registerBinding,
  remote,
  remoteAction,
  removeClass,
  renderIf,
  renderList,
  sanitizeHtml,
  setNavigationOptions,
  setSanitizer,
  setupEventDelegation,
  throttle,
  toRaw,
  transformers,
  unregisterBinding,
  watchProp,
  withErrorBoundary
} from "./runtime-rq764jre.js";
import {
  Derived,
  Effect,
  Rune,
  WSClient,
  applyStateUpdate,
  batch,
  createDevToolsPanel,
  createInspector,
  debugLog,
  getWebSocketClient,
  initWebSocket,
  inspect,
  isDev,
  memoryUsage,
  rune,
  sendAction,
  syncBatch,
  syncedRune,
  timing,
  toggleDevTools,
  untrack,
  updateDevToolsPanel
} from "./websocket-g18v2mwh.js";
import"./runtime-3hqyeswk.js";
// client/src/transition.ts
var linear = (t) => t;
var cubicOut = (t) => {
  const f = t - 1;
  return f * f * f + 1;
};
var cubicInOut = (t) => {
  return t < 0.5 ? 4 * t * t * t : 0.5 * Math.pow(2 * t - 2, 3) + 1;
};
var elasticOut = (t) => {
  return Math.sin(-13 * (t + 1) * Math.PI / 2) * Math.pow(2, -10 * t) + 1;
};
var bounceOut = (t) => {
  const n1 = 7.5625;
  const d1 = 2.75;
  if (t < 1 / d1) {
    return n1 * t * t;
  } else if (t < 2 / d1) {
    return n1 * (t -= 1.5 / d1) * t + 0.75;
  } else if (t < 2.5 / d1) {
    return n1 * (t -= 2.25 / d1) * t + 0.9375;
  } else {
    return n1 * (t -= 2.625 / d1) * t + 0.984375;
  }
};
function fade(node, { delay = 0, duration = 400, _easing = linear } = {}) {
  const o = +getComputedStyle(node).opacity;
  return {
    delay,
    duration,
    easing: "linear",
    css: (t) => `opacity: ${t * o}`
  };
}
function fly(node, {
  delay = 0,
  duration = 400,
  _easing = cubicOut,
  x = 0,
  y = 0,
  opacity = 0
} = {}) {
  const style = getComputedStyle(node);
  const targetOpacity = +style.opacity;
  const transform = style.transform === "none" ? "" : style.transform;
  return {
    delay,
    duration,
    easing: "ease-out",
    css: (t, u) => `
			transform: ${transform} translate(${(1 - t) * x}px, ${(1 - t) * y}px);
			opacity: ${targetOpacity - (targetOpacity - opacity) * u}
		`
  };
}
function slide(node, { delay = 0, duration = 400, _easing = cubicOut } = {}) {
  const style = getComputedStyle(node);
  const opacity = +style.opacity;
  const height = parseFloat(style.height);
  const paddingTop = parseFloat(style.paddingTop);
  const paddingBottom = parseFloat(style.paddingBottom);
  const marginTop = parseFloat(style.marginTop);
  const marginBottom = parseFloat(style.marginBottom);
  const borderTopWidth = parseFloat(style.borderTopWidth);
  const borderBottomWidth = parseFloat(style.borderBottomWidth);
  return {
    delay,
    duration,
    easing: "ease-out",
    css: (t) => `
			overflow: hidden;
			opacity: ${Math.min(t * 20, 1) * opacity};
			height: ${t * height}px;
			padding-top: ${t * paddingTop}px;
			padding-bottom: ${t * paddingBottom}px;
			margin-top: ${t * marginTop}px;
			margin-bottom: ${t * marginBottom}px;
			border-top-width: ${t * borderTopWidth}px;
			border-bottom-width: ${t * borderBottomWidth}px;
		`
  };
}
function scale(node, {
  delay = 0,
  duration = 400,
  _easing = cubicOut,
  start = 0,
  opacity = 0
} = {}) {
  const style = getComputedStyle(node);
  const targetOpacity = +style.opacity;
  const transform = style.transform === "none" ? "" : style.transform;
  const sd = 1 - start;
  return {
    delay,
    duration,
    easing: "ease-out",
    css: (t, u) => `
            transform: ${transform} scale(${1 - sd * u});
            opacity: ${targetOpacity - (targetOpacity - opacity) * u}
        `
  };
}
function blur(node, {
  delay = 0,
  duration = 400,
  _easing = cubicInOut,
  amount = 5,
  opacity = 0
} = {}) {
  const style = getComputedStyle(node);
  const targetOpacity = +style.opacity;
  return {
    delay,
    duration,
    easing: "ease-in-out",
    css: (t, u) => `
            opacity: ${targetOpacity - (targetOpacity - opacity) * u};
            filter: blur(${u * amount}px);
        `
  };
}
function crossfade(node, { delay = 0, duration = 400, _easing = linear } = {}) {
  return {
    delay,
    duration,
    easing: "linear",
    css: (t, _u) => `
            opacity: ${t};
            position: absolute;
        `
  };
}
var activeTransitions = new Set;
function transitionIn(node, fn, params) {
  if (activeTransitions.has(node))
    return;
  activeTransitions.add(node);
  const config = fn(node, params);
  const duration = config.duration ?? 400;
  const delay = config.delay || 0;
  const css = config.css || (() => "");
  if (duration === 0 && delay === 0) {
    activeTransitions.delete(node);
    return;
  }
  const originalStyle = node.getAttribute("style") || "";
  const name = `gospa-transition-${Math.random().toString(36).substring(2, 9)}`;
  const keyframes = `
		@keyframes ${name} {
			0% { ${css(0, 1)} }
			100% { ${css(1, 0)} }
		}
	`;
  const styleSheet = document.createElement("style");
  styleSheet.textContent = keyframes;
  document.head.appendChild(styleSheet);
  node.style.animation = `${name} ${duration}ms ${config.easing || "linear"} ${delay}ms both`;
  setTimeout(() => {
    node.setAttribute("style", originalStyle);
    node.style.animation = "";
    styleSheet.remove();
    activeTransitions.delete(node);
  }, duration + delay);
}
function transitionOut(node, fn, params, onComplete) {
  if (activeTransitions.has(node))
    return;
  activeTransitions.add(node);
  const config = fn(node, params);
  const duration = config.duration ?? 400;
  const delay = config.delay || 0;
  const css = config.css || (() => "");
  if (duration === 0 && delay === 0) {
    activeTransitions.delete(node);
    onComplete();
    return;
  }
  const name = `gospa-transition-${Math.random().toString(36).substring(2, 9)}`;
  const keyframes = `
		@keyframes ${name} {
			0% { ${css(1, 0)} }
			100% { ${css(0, 1)} }
		}
	`;
  const styleSheet = document.createElement("style");
  styleSheet.textContent = keyframes;
  document.head.appendChild(styleSheet);
  node.style.animation = `${name} ${duration}ms ${config.easing || "linear"} ${delay}ms both`;
  setTimeout(() => {
    styleSheet.remove();
    activeTransitions.delete(node);
    onComplete();
  }, duration + delay);
}
function setupTransitions(root = document.body) {
  const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      if (mutation.type === "childList") {
        mutation.addedNodes.forEach((node) => {
          if (node.nodeType === Node.ELEMENT_NODE) {
            const el = node;
            if (el.closest("[data-gospa-static]"))
              return;
            const transitionType = el.getAttribute("data-transition-in") || el.getAttribute("data-transition");
            if (transitionType) {
              const fn = getTransitionFn(transitionType);
              if (fn)
                transitionIn(el, fn, getTransitionParams(el));
            }
          }
        });
        mutation.removedNodes.forEach((node) => {
          if (node.nodeType === Node.ELEMENT_NODE) {
            const el = node;
            if (el.closest("[data-gospa-static]"))
              return;
            const transitionType = el.getAttribute("data-transition-out") || el.getAttribute("data-transition");
            if (transitionType) {
              const fn = getTransitionFn(transitionType);
              if (fn && !activeTransitions.has(el)) {
                const clone = el.cloneNode(true);
                clone.querySelectorAll("[data-bind]").forEach((n) => n.removeAttribute("data-bind"));
                clone.removeAttribute("data-bind");
                if (mutation.previousSibling && mutation.previousSibling.parentNode) {
                  mutation.previousSibling.parentNode.insertBefore(clone, mutation.previousSibling.nextSibling);
                } else if (mutation.target) {
                  mutation.target.appendChild(clone);
                }
                transitionOut(clone, fn, getTransitionParams(el), () => clone.remove());
              }
            }
          }
        });
      }
    });
  });
  observer.observe(root, { childList: true, subtree: true });
}
function getTransitionFn(name) {
  if (name.startsWith("fade"))
    return fade;
  if (name.startsWith("fly"))
    return fly;
  if (name.startsWith("slide"))
    return slide;
  if (name.startsWith("scale"))
    return scale;
  if (name.startsWith("blur"))
    return blur;
  if (name.startsWith("crossfade"))
    return crossfade;
  return null;
}
function getTransitionParams(node) {
  const paramStr = node.getAttribute("data-transition-params");
  if (!paramStr)
    return {};
  try {
    return JSON.parse(paramStr);
  } catch (e) {
    console.warn("Invalid transition parameters:", paramStr);
    return {};
  }
}
// client/src/island.ts
var PRIORITY_MAP = {
  critical: 100,
  high: 75,
  normal: 50,
  low: 25,
  deferred: 10
};

class IslandManager {
  islands = new Map;
  hydrated = new Set;
  pending = new Map;
  queue = {
    critical: [],
    high: [],
    normal: [],
    low: [],
    deferred: []
  };
  processing = false;
  moduleLoader;
  moduleBasePath;
  defaultTimeout;
  debug;
  observers = [];
  idleCallbacks = new Map;
  interactionListeners = new Map;
  constructor(config = {}) {
    this.moduleLoader = config.moduleLoader ?? this.defaultModuleLoader;
    this.moduleBasePath = config.moduleBasePath ?? "/islands";
    this.defaultTimeout = config.defaultTimeout ?? 30000;
    this.debug = config.debug ?? false;
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", () => this.discoverIslands());
    } else {
      this.discoverIslands();
    }
    const root = document.getElementById("app") || document.body;
    setupEventDelegation(root);
  }
  discoverIslands() {
    const elements = document.querySelectorAll("[data-gospa-island]");
    const discovered = [];
    elements.forEach((element) => {
      const data2 = this.parseIslandElement(element);
      if (data2 && !this.islands.has(data2.id)) {
        this.islands.set(data2.id, data2);
        discovered.push(data2);
        this.log("Discovered island:", data2.name, data2.id);
      }
    });
    this.scheduleHydration(discovered);
    return discovered;
  }
  parseIslandElement(element) {
    const id = element.id || this.generateId();
    const name = element.getAttribute("data-gospa-island");
    if (!name)
      return null;
    const mode = element.getAttribute("data-gospa-mode") || "immediate";
    const priority = element.getAttribute("data-gospa-priority") || "normal";
    let props;
    let state;
    const registry = window.__GOSPA_DATA__;
    if (Array.isArray(registry)) {
      const islandData = registry.find((d) => d.id === id || d.id === name);
      if (islandData) {
        props = islandData.props;
        state = islandData.state;
      }
    }
    if (!props) {
      const propsAttr = element.getAttribute("data-gospa-props");
      if (propsAttr) {
        try {
          props = JSON.parse(propsAttr);
        } catch (e) {
          this.log("Failed to parse props for island:", name, e);
        }
      }
    }
    if (!state) {
      const stateAttr = element.getAttribute("data-gospa-state");
      if (stateAttr) {
        try {
          state = JSON.parse(stateAttr);
        } catch (e) {
          this.log("Failed to parse state for island:", name, e);
        }
      }
    }
    const thresholdAttr = element.getAttribute("data-gospa-threshold");
    const deferAttr = element.getAttribute("data-gospa-defer");
    const threshold = thresholdAttr ? parseInt(thresholdAttr, 10) : undefined;
    const defer = deferAttr ? parseInt(deferAttr, 10) : undefined;
    return {
      id,
      name,
      mode,
      priority,
      props,
      state,
      threshold: threshold !== undefined && Number.isFinite(threshold) && threshold >= 0 ? threshold : undefined,
      defer: defer !== undefined && Number.isFinite(defer) && defer >= 0 ? defer : undefined,
      clientOnly: element.getAttribute("data-gospa-client-only") === "true",
      serverOnly: element.getAttribute("data-gospa-server-only") === "true",
      element
    };
  }
  scheduleHydration(islands) {
    for (const island of islands) {
      if (this.hydrated.has(island.id) || this.pending.has(island.id)) {
        continue;
      }
      switch (island.mode) {
        case "immediate":
          this.queueHydration(island);
          break;
        case "visible":
          this.scheduleVisibleHydration(island);
          break;
        case "idle":
          this.scheduleIdleHydration(island);
          break;
        case "interaction":
          this.scheduleInteractionHydration(island);
          break;
        case "lazy":
          break;
      }
    }
    this.processQueue();
  }
  queueHydration(island) {
    if (this.pending.has(island.id)) {
      return this.pending.get(island.id);
    }
    const promise = new Promise((resolve, reject) => {
      this.queue[island.priority].push({
        island,
        resolve,
        reject
      });
    });
    this.pending.set(island.id, promise);
    return promise;
  }
  async processQueue() {
    if (this.processing)
      return;
    this.processing = true;
    while (this.queue.critical.length > 0 || this.queue.high.length > 0 || this.queue.normal.length > 0 || this.queue.low.length > 0 || this.queue.deferred.length > 0) {
      const item = this.queue.critical.shift() ?? this.queue.high.shift() ?? this.queue.normal.shift() ?? this.queue.low.shift() ?? this.queue.deferred.shift();
      if (!item)
        break;
      try {
        const result = await this.hydrateIsland(item.island);
        item.resolve(result);
      } catch (error) {
        item.reject(error);
      }
    }
    this.processing = false;
  }
  async hydrateIsland(island) {
    if (this.hydrated.has(island.id)) {
      return { id: island.id, name: island.name, success: true };
    }
    if (island.serverOnly) {
      this.log("Skipping server-only island:", island.name);
      return { id: island.id, name: island.name, success: true };
    }
    this.log("Hydrating island:", island.name, island.id);
    try {
      const setupFn = getSetup(island.name);
      if (setupFn) {
        await setupFn(island.element, island.props ?? {}, island.state ?? {});
        this.hydrated.add(island.id);
        this.log("Hydrated island from registry:", island.name);
        return { id: island.id, name: island.name, success: true };
      }
      const module = await this.moduleLoader(island.name);
      if (!module) {
        throw new Error(`Island module not found: ${island.name}`);
      }
      const hydrateFn = module.hydrate ?? module.default?.hydrate ?? module.mount ?? module.default?.mount;
      if (!hydrateFn) {
        throw new Error(`No hydrate or mount function found for island: ${island.name}`);
      }
      await hydrateFn(island.element, island.props ?? {}, island.state ?? {});
      this.hydrated.add(island.id);
      this.log("Hydrated island:", island.name);
      return { id: island.id, name: island.name, success: true };
    } catch (error) {
      this.log("Failed to hydrate island:", island.name, error);
      this.hydrated.add(island.id);
      throw error;
    }
  }
  scheduleVisibleHydration(island) {
    if (!("IntersectionObserver" in window)) {
      this.queueHydration(island);
      this.processQueue();
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          this.queueHydration(island);
          this.processQueue();
          observer.disconnect();
          this.observers = this.observers.filter((o) => o !== observer);
        }
      }
    }, {
      rootMargin: `${island.threshold ?? 200}px`
    });
    observer.observe(island.element);
    this.observers.push(observer);
  }
  scheduleIdleHydration(island) {
    if (typeof requestIdleCallback !== "undefined") {
      const callbackId = requestIdleCallback(() => {
        this.queueHydration(island);
        this.processQueue();
        this.idleCallbacks.delete(island.id);
      }, { timeout: island.defer ?? 2000 });
      this.idleCallbacks.set(island.id, callbackId);
    } else {
      const timeoutId = setTimeout(() => {
        this.queueHydration(island);
        this.processQueue();
        this.idleCallbacks.delete(island.id);
      }, island.defer ?? 2000);
      this.idleCallbacks.set(island.id, timeoutId);
    }
  }
  scheduleInteractionHydration(island) {
    const events = ["mouseenter", "touchstart", "focusin", "click"];
    const hydrateOnInteraction = () => {
      this.queueHydration(island);
      this.processQueue();
      for (const event of events) {
        island.element.removeEventListener(event, hydrateOnInteraction);
      }
      this.interactionListeners.delete(island.id);
    };
    for (const event of events) {
      island.element.addEventListener(event, hydrateOnInteraction, {
        passive: true,
        once: true
      });
    }
    this.interactionListeners.set(island.id, hydrateOnInteraction);
  }
  defaultModuleLoader = async (name) => {
    try {
      const module = await import(`${this.moduleBasePath}/${name}.js`);
      return module;
    } catch (error) {
      this.log("Failed to load island module:", name, error);
      return null;
    }
  };
  generateId() {
    return `gospa-island-${Math.random().toString(36).substring(2, 11)}`;
  }
  log(...args) {
    if (this.debug) {
      console.log("[GoSPA Islands]", ...args);
    }
  }
  getIslands() {
    return Array.from(this.islands.values());
  }
  getIsland(id) {
    return this.islands.get(id);
  }
  isHydrated(id) {
    return this.hydrated.has(id);
  }
  async hydrate(idOrName) {
    let island = this.islands.get(idOrName);
    if (!island) {
      island = Array.from(this.islands.values()).find((i) => i.name === idOrName);
    }
    if (!island) {
      return null;
    }
    return this.hydrateIsland(island);
  }
  destroy() {
    for (const observer of this.observers) {
      observer.disconnect();
    }
    this.observers = [];
    for (const [, callbackId] of this.idleCallbacks) {
      if ("cancelIdleCallback" in window) {
        window.cancelIdleCallback(callbackId);
      } else {
        clearTimeout(callbackId);
      }
    }
    this.idleCallbacks.clear();
    for (const [_id, listener] of this.interactionListeners) {
      const island = this.islands.get(_id);
      if (island) {
        const events = ["mouseenter", "touchstart", "focusin", "click"];
        for (const event of events) {
          island.element.removeEventListener(event, listener);
        }
      }
    }
    this.interactionListeners.clear();
    this.islands.clear();
    this.hydrated.clear();
    this.pending.clear();
    this.queue.critical = [];
    this.queue.high = [];
    this.queue.normal = [];
    this.queue.low = [];
    this.queue.deferred = [];
  }
}
var globalManager = null;
function initIslands(config) {
  if (globalManager) {
    return globalManager;
  }
  globalManager = new IslandManager(config);
  return globalManager;
}
function getIslandManager() {
  return globalManager;
}
async function hydrateIsland(idOrName) {
  if (!globalManager) {
    console.warn("Island manager not initialized. Call initIslands() first.");
    return null;
  }
  return globalManager.hydrate(idOrName);
}
if (typeof document !== "undefined") {
  initIslands();
}
if (typeof window !== "undefined") {
  window.__GOSPA_ISLAND_MANAGER__ = {
    init: initIslands,
    get: getIslandManager,
    hydrate: hydrateIsland,
    IslandManager
  };
}
// client/src/priority.ts
var PRIORITY_CRITICAL = 100;
var PRIORITY_HIGH = 75;
var PRIORITY_NORMAL = 50;
var PRIORITY_LOW = 25;
var PRIORITY_DEFERRED = 10;
var DEFAULT_CONFIG = {
  maxConcurrent: 3,
  idleTimeout: 2000,
  intersectionThreshold: 0.1,
  intersectionRootMargin: "50px",
  enablePreload: true
};

class PriorityScheduler {
  config;
  islands = new Map;
  hydrationQueue = [];
  activeHydrations = 0;
  observers = new Map;
  idleCallbacks = new Map;
  interactionHandlers = new Map;
  constructor(config = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }
  registerPlan(plan) {
    if (this.config.enablePreload) {
      this.preloadScripts(plan.preload);
    }
    for (const island of plan.immediate) {
      this.registerIsland(island, "immediate");
    }
    for (const island of plan.idle) {
      this.registerIsland(island, "idle");
    }
    for (const island of plan.visible) {
      this.registerIsland(island, "visible");
    }
    for (const island of plan.interaction) {
      this.registerIsland(island, "interaction");
    }
    for (const island of plan.lazy) {
      this.registerIsland(island, "lazy");
    }
    this.processQueue();
  }
  registerIsland(island, mode) {
    const tracked = {
      ...island,
      state: "pending",
      mode
    };
    this.islands.set(island.id, tracked);
    const element = document.querySelector(`[data-island-id="${island.id}"]`);
    if (element) {
      tracked.element = element;
    }
    this.setupHydrationTrigger(tracked);
  }
  setupHydrationTrigger(island) {
    switch (island.mode) {
      case "immediate":
        this.hydrationQueue.push(island);
        break;
      case "idle":
        this.setupIdleHydration(island);
        break;
      case "visible":
        this.setupVisibleHydration(island);
        break;
      case "interaction":
        this.setupInteractionHydration(island);
        break;
      case "lazy":
        break;
    }
  }
  setupIdleHydration(island) {
    if ("requestIdleCallback" in window) {
      const callbackId = requestIdleCallback(() => {
        this.hydrationQueue.push(island);
        this.processQueue();
      }, { timeout: this.config.idleTimeout });
      this.idleCallbacks.set(island.id, callbackId);
    } else {
      setTimeout(() => {
        this.hydrationQueue.push(island);
        this.processQueue();
      }, this.config.idleTimeout);
    }
  }
  setupVisibleHydration(island) {
    if (!island.element) {
      this.hydrationQueue.push(island);
      this.processQueue();
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          this.hydrationQueue.push(island);
          this.processQueue();
          observer.disconnect();
          this.observers.delete(island.id);
        }
      }
    }, {
      threshold: this.config.intersectionThreshold,
      rootMargin: this.config.intersectionRootMargin
    });
    observer.observe(island.element);
    this.observers.set(island.id, observer);
  }
  setupInteractionHydration(island) {
    if (!island.element) {
      this.hydrationQueue.push(island);
      this.processQueue();
      return;
    }
    const events = ["click", "focus", "mouseenter", "touchstart"];
    const handlers = [];
    const hydrateOnInteraction = (_event) => {
      for (let i = 0;i < events.length; i++) {
        island.element.removeEventListener(events[i], handlers[i]);
      }
      this.hydrationQueue.push(island);
      this.processQueue();
    };
    for (const eventType of events) {
      const handler = hydrateOnInteraction;
      handlers.push(handler);
      island.element.addEventListener(eventType, handler, {
        passive: true,
        once: true
      });
    }
    this.interactionHandlers.set(island.id, handlers);
  }
  processQueue() {
    this.hydrationQueue.sort((a, b) => {
      if (a.priority !== b.priority) {
        return b.priority - a.priority;
      }
      return a.position - b.position;
    });
    while (this.activeHydrations < this.config.maxConcurrent && this.hydrationQueue.length > 0) {
      const island = this.hydrationQueue.shift();
      if (island && island.state === "pending") {
        this.hydrateIsland(island);
      }
    }
  }
  async hydrateIsland(island) {
    island.state = "hydrating";
    this.activeHydrations++;
    try {
      await this.waitForDependencies(island);
      const event = new CustomEvent("gospa:hydrate", {
        detail: {
          id: island.id,
          name: island.name,
          state: island.state
        }
      });
      document.dispatchEvent(event);
      island.state = "hydrated";
      const hydratedEvent = new CustomEvent("gospa:hydrated", {
        detail: { id: island.id, name: island.name }
      });
      document.dispatchEvent(hydratedEvent);
    } catch (error) {
      island.state = "error";
      island.error = error instanceof Error ? error : new Error(String(error));
      const errorEvent = new CustomEvent("gospa:hydration-error", {
        detail: { id: island.id, error: island.error }
      });
      document.dispatchEvent(errorEvent);
    } finally {
      this.activeHydrations--;
      this.processQueue();
    }
  }
  async waitForDependencies(island) {
    if (!island.dependencies || island.dependencies.length === 0) {
      return;
    }
    const promises = island.dependencies.map((depId) => {
      return new Promise((resolve) => {
        const dep = this.islands.get(depId);
        if (!dep || dep.state === "hydrated") {
          resolve();
          return;
        }
        const handler = (event) => {
          const customEvent = event;
          if (customEvent.detail.id === depId) {
            document.removeEventListener("gospa:hydrated", handler);
            resolve();
          }
        };
        document.addEventListener("gospa:hydrated", handler);
      });
    });
    await Promise.all(promises);
  }
  preloadScripts(scripts) {
    for (const src of scripts) {
      const link = document.createElement("link");
      link.rel = "preload";
      link.as = "script";
      link.href = src;
      document.head.appendChild(link);
    }
  }
  forceHydrate(id) {
    const island = this.islands.get(id);
    if (island && island.state === "pending") {
      this.cancelTriggers(id);
      this.hydrationQueue.push(island);
      this.processQueue();
    }
  }
  cancelTriggers(id) {
    const idleCallback = this.idleCallbacks.get(id);
    if (idleCallback !== undefined) {
      cancelIdleCallback(idleCallback);
      this.idleCallbacks.delete(id);
    }
    const observer = this.observers.get(id);
    if (observer) {
      observer.disconnect();
      this.observers.delete(id);
    }
    const handlers = this.interactionHandlers.get(id);
    if (handlers) {
      const island = this.islands.get(id);
      if (island?.element) {
        const events = ["click", "focus", "mouseenter", "touchstart"];
        for (let i = 0;i < events.length; i++) {
          island.element.removeEventListener(events[i], handlers[i]);
        }
      }
      this.interactionHandlers.delete(id);
    }
  }
  getIslandState(id) {
    return this.islands.get(id)?.state;
  }
  getPendingIslands() {
    return Array.from(this.islands.values()).filter((i) => i.state === "pending");
  }
  getHydratedIslands() {
    return Array.from(this.islands.values()).filter((i) => i.state === "hydrated");
  }
  getStats() {
    const islands = Array.from(this.islands.values());
    return {
      total: islands.length,
      pending: islands.filter((i) => i.state === "pending").length,
      hydrating: islands.filter((i) => i.state === "hydrating").length,
      hydrated: islands.filter((i) => i.state === "hydrated").length,
      errors: islands.filter((i) => i.state === "error").length
    };
  }
  destroy() {
    for (const callbackId of this.idleCallbacks.values()) {
      cancelIdleCallback(callbackId);
    }
    this.idleCallbacks.clear();
    for (const observer of this.observers.values()) {
      observer.disconnect();
    }
    this.observers.clear();
    this.interactionHandlers.clear();
    this.islands.clear();
    this.hydrationQueue = [];
  }
}
var globalScheduler = null;
function getPriorityScheduler(config) {
  if (!globalScheduler) {
    globalScheduler = new PriorityScheduler(config);
  }
  return globalScheduler;
}
function initPriorityHydration(plan) {
  const scheduler = getPriorityScheduler();
  scheduler.registerPlan(plan);
  return scheduler;
}
// client/src/streaming.ts
class StreamingManager {
  islands = [];
  hydrationQueue = [];
  hydratedIslands = new Set;
  isHydrating = false;
  options;
  constructor(options = {}) {
    this.options = {
      enableLogging: false,
      hydrationTimeout: 30000,
      ...options
    };
    this.setupStreamHandler();
  }
  setupStreamHandler() {
    const existingHandler = globalThis.__GOSPA_STREAM__;
    globalThis.__GOSPA_STREAM__ = (chunk) => {
      if (typeof existingHandler === "function") {
        existingHandler(chunk);
      }
      this.processChunk(chunk);
    };
  }
  processChunk(chunk) {
    if (this.options.enableLogging) {
      console.log("[GoSPA Stream]", chunk.type, chunk.id || "", chunk);
    }
    switch (chunk.type) {
      case "html":
        this.handleHtmlChunk(chunk);
        break;
      case "island":
        this.handleIslandChunk(chunk);
        break;
      case "script":
        this.handleScriptChunk(chunk);
        break;
      case "state":
        this.handleStateChunk(chunk);
        break;
      case "error":
        this.handleErrorChunk(chunk);
        break;
    }
  }
  handleHtmlChunk(chunk) {
    const element = document.getElementById(chunk.id);
    if (element) {
      const sanitized = sanitizeHtml(chunk.content);
      if (sanitized instanceof Promise) {
        sanitized.then((result) => {
          element.innerHTML = result;
        });
      } else {
        element.innerHTML = sanitized;
      }
      element.dispatchEvent(new CustomEvent("gospa:html-update", {
        detail: { id: chunk.id, content: chunk.content }
      }));
    }
  }
  handleIslandChunk(chunk) {
    const islandData = chunk.data;
    if (!islandData || !islandData.id) {
      console.error("[GoSPA Stream] Invalid island data:", chunk);
      return;
    }
    this.islands.push(islandData);
    this.queueHydration(islandData);
  }
  handleScriptChunk(chunk) {
    const script = document.createElement("script");
    script.textContent = chunk.content;
    document.head.appendChild(script);
  }
  handleStateChunk(chunk) {
    const gospaState = globalThis.__GOSPA_STATE__ ||= {};
    gospaState[chunk.id] = chunk.data;
    document.dispatchEvent(new CustomEvent("gospa:state-update", {
      detail: { id: chunk.id, state: chunk.data }
    }));
  }
  handleErrorChunk(chunk) {
    console.error("[GoSPA Stream Error]", chunk.content);
    document.dispatchEvent(new CustomEvent("gospa:stream-error", {
      detail: { error: chunk.content }
    }));
  }
  queueHydration(island) {
    switch (island.mode) {
      case "immediate":
        this.hydrateImmediate(island);
        break;
      case "visible":
        this.hydrateOnVisible(island);
        break;
      case "idle":
        this.hydrateOnIdle(island);
        break;
      case "interaction":
        this.hydrateOnInteraction(island);
        break;
      case "lazy":
        this.hydrateLazy(island);
        break;
      default:
        this.hydrateImmediate(island);
    }
  }
  hydrateImmediate(island) {
    this.addToHydrationQueue(island, "high");
  }
  hydrateOnVisible(island) {
    const element = document.querySelector(`[data-gospa-island="${island.id}"]`);
    if (!element) {
      this.hydrateImmediate(island);
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          observer.disconnect();
          this.addToHydrationQueue(island, "normal");
        }
      }
    }, { rootMargin: "100px" });
    observer.observe(element);
  }
  hydrateOnIdle(island) {
    if ("requestIdleCallback" in globalThis) {
      globalThis.requestIdleCallback(() => {
        this.addToHydrationQueue(island, "low");
      });
    } else {
      setTimeout(() => {
        this.addToHydrationQueue(island, "low");
      }, 100);
    }
  }
  hydrateOnInteraction(island) {
    const element = document.querySelector(`[data-gospa-island="${island.id}"]`);
    if (!element) {
      this.hydrateImmediate(island);
      return;
    }
    const events = ["mouseenter", "touchstart", "focusin", "click"];
    const handler = () => {
      events.forEach((event) => element.removeEventListener(event, handler));
      this.addToHydrationQueue(island, "high");
    };
    events.forEach((event) => {
      element.addEventListener(event, handler, { once: true, passive: true });
    });
  }
  hydrateLazy(island) {
    if (document.readyState === "complete") {
      this.hydrateOnIdle(island);
    } else {
      globalThis.addEventListener("load", () => {
        setTimeout(() => {
          this.hydrateOnIdle(island);
        }, 500);
      });
    }
  }
  addToHydrationQueue(island, priority) {
    if (this.hydratedIslands.has(island.id)) {
      return;
    }
    const queueItem = {
      island,
      resolve: () => {},
      reject: () => {}
    };
    if (priority === "high") {
      this.hydrationQueue.unshift(queueItem);
    } else {
      this.hydrationQueue.push(queueItem);
    }
    this.processQueue();
  }
  processQueue() {
    if (this.isHydrating || this.hydrationQueue.length === 0) {
      return;
    }
    this.isHydrating = true;
    const item = this.hydrationQueue.shift();
    if (item) {
      this.hydrateIsland(item.island).then(() => {
        this.hydratedIslands.add(item.island.id);
        this.isHydrating = false;
        this.processQueue();
      }).catch((error) => {
        console.error("[GoSPA] Hydration error:", error);
        this.isHydrating = false;
        this.processQueue();
      });
    }
  }
  async hydrateIsland(island) {
    const element = document.querySelector(`[data-gospa-island="${island.id}"]`);
    if (!element) {
      if (this.options.enableLogging) {
        console.warn("[GoSPA] Island element not found:", island.id);
      }
      return;
    }
    const islandManager = globalThis.__GOSPA_ISLAND_MANAGER__;
    if (islandManager && typeof islandManager.hydrate === "function") {
      await islandManager.hydrate(island.id, island);
    }
    element.dispatchEvent(new CustomEvent("gospa:hydrated", {
      detail: { island }
    }));
    if (this.options.enableLogging) {
      console.log("[GoSPA] Hydrated island:", island.id, island.name);
    }
  }
  getIslands() {
    return [...this.islands];
  }
  getHydratedIslands() {
    return new Set(this.hydratedIslands);
  }
  isHydrated(islandId) {
    return this.hydratedIslands.has(islandId);
  }
  async hydrate(islandId) {
    const island = this.islands.find((i) => i.id === islandId);
    if (island) {
      await this.hydrateIsland(island);
    }
  }
}
var streamingManager = null;
function initStreaming(options) {
  if (!streamingManager) {
    streamingManager = new StreamingManager(options);
  }
  return streamingManager;
}
function getStreamingManager() {
  return streamingManager;
}
if (typeof window !== "undefined") {
  setTimeout(() => {
    if (!streamingManager) {
      initStreaming();
    }
  }, 0);
}
// client/src/resource.ts
class Resource {
  _status;
  _data;
  _error;
  _fetcher;
  constructor(fetcher) {
    this._fetcher = fetcher;
    this._status = rune("idle");
    this._data = rune(undefined);
    this._error = rune(undefined);
  }
  get status() {
    return this._status.get();
  }
  get data() {
    return this._data.get();
  }
  get error() {
    return this._error.get();
  }
  get isPending() {
    return this.status === "pending";
  }
  get isSuccess() {
    return this.status === "success";
  }
  get isError() {
    return this.status === "error";
  }
  async fetch() {
    if (this._status.peek() === "pending")
      return;
    this._status.set("pending");
    this._error.set(undefined);
    try {
      const result = await this._fetcher();
      this._data.set(result);
      this._status.set("success");
      return result;
    } catch (err) {
      this._error.set(err);
      this._status.set("error");
      throw err;
    }
  }
  async refetch() {
    return this.fetch();
  }
  reset() {
    this._status.set("idle");
    this._data.set(undefined);
    this._error.set(undefined);
  }
}
function resourceReactive(fetcher) {
  const r = new Resource(fetcher);
  return r;
}
// client/src/ws-tab-sync.ts
class WSTabSync {
  channel = null;
  tabId;
  tabs = new Map;
  isLeader = false;
  pingTimer = null;
  config;
  stateRunes = new Map;
  onStateUpdate = null;
  onAction = null;
  constructor(config = {}) {
    this.tabId = `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    this.config = {
      channelName: config.channelName ?? "gospa-ws-sync",
      enabled: config.enabled ?? true,
      pingInterval: config.pingInterval ?? 5000,
      tabTimeout: config.tabTimeout ?? 1e4
    };
    if (this.config.enabled && typeof BroadcastChannel !== "undefined") {
      this.init();
    }
  }
  init() {
    try {
      this.channel = new BroadcastChannel(this.config.channelName);
      this.channel.onmessage = (event) => this.handleMessage(event.data);
      this.broadcast({
        type: "ping",
        tabId: this.tabId,
        timestamp: Date.now()
      });
      this.pingTimer = setInterval(() => {
        this.broadcast({
          type: "ping",
          tabId: this.tabId,
          timestamp: Date.now()
        });
        this.cleanupDeadTabs();
      }, this.config.pingInterval);
      window.addEventListener("beforeunload", () => {
        this.broadcast({
          type: "ws-disconnected",
          tabId: this.tabId,
          timestamp: Date.now()
        });
      });
      console.log(`[GoSPA Tab Sync] Initialized with tab ID: ${this.tabId}`);
    } catch (error) {
      console.warn("[GoSPA Tab Sync] BroadcastChannel not available:", error);
    }
  }
  handleMessage(message) {
    if (message.tabId === this.tabId)
      return;
    this.tabs.set(message.tabId, {
      id: message.tabId,
      lastSeen: Date.now(),
      isLeader: false
    });
    switch (message.type) {
      case "ping":
        this.broadcast({
          type: "pong",
          tabId: this.tabId,
          timestamp: Date.now()
        });
        this.electLeader();
        break;
      case "pong":
        this.electLeader();
        break;
      case "state-update":
        if (message.payload && typeof message.payload === "object") {
          const { key, value } = message.payload;
          const rune2 = this.stateRunes.get(key);
          if (rune2) {
            batch(() => {
              rune2.set(value);
            });
          }
          this.onStateUpdate?.(key, value);
        }
        break;
      case "state-sync":
        if (message.payload && typeof message.payload === "object") {
          const state = message.payload;
          batch(() => {
            for (const [key, value] of Object.entries(state)) {
              const rune2 = this.stateRunes.get(key);
              if (rune2) {
                rune2.set(value);
              }
            }
          });
        }
        break;
      case "action":
        if (message.payload && typeof message.payload === "object") {
          const { action, payload } = message.payload;
          this.onAction?.(action, payload);
        }
        break;
      case "ws-connected":
        console.log(`[GoSPA Tab Sync] Tab ${message.tabId} connected`);
        this.electLeader();
        break;
      case "ws-disconnected":
        this.tabs.delete(message.tabId);
        this.electLeader();
        break;
    }
  }
  broadcast(message) {
    if (this.channel) {
      try {
        this.channel.postMessage(message);
      } catch (error) {
        console.warn("[GoSPA Tab Sync] Failed to broadcast:", error);
      }
    }
  }
  electLeader() {
    const now = Date.now();
    let oldestTab = null;
    const allTabs = [
      { id: this.tabId, lastSeen: now, isLeader: false },
      ...Array.from(this.tabs.values())
    ];
    for (const tab of allTabs) {
      if (!oldestTab || tab.lastSeen < oldestTab.lastSeen) {
        oldestTab = tab;
      }
    }
    const wasLeader = this.isLeader;
    this.isLeader = oldestTab?.id === this.tabId;
    if (this.isLeader && !wasLeader) {
      console.log("[GoSPA Tab Sync] This tab is now the leader");
      this.syncStateToTabs();
    }
  }
  cleanupDeadTabs() {
    const now = Date.now();
    for (const [tabId, tab] of this.tabs) {
      if (now - tab.lastSeen > this.config.tabTimeout) {
        this.tabs.delete(tabId);
        console.log(`[GoSPA Tab Sync] Removed dead tab: ${tabId}`);
      }
    }
    this.electLeader();
  }
  syncStateToTabs() {
    const state = {};
    for (const [key, rune2] of this.stateRunes) {
      state[key] = rune2.get();
    }
    this.broadcast({
      type: "state-sync",
      tabId: this.tabId,
      timestamp: Date.now(),
      payload: state
    });
  }
  registerState(key, rune2) {
    this.stateRunes.set(key, rune2);
    rune2.subscribe((value) => {
      if (!this.isLeader)
        return;
      this.broadcast({
        type: "state-update",
        tabId: this.tabId,
        timestamp: Date.now(),
        payload: { key, value }
      });
    });
  }
  unregisterState(key) {
    this.stateRunes.delete(key);
  }
  onStateChange(callback) {
    this.onStateUpdate = callback;
  }
  onActionReceived(callback) {
    this.onAction = callback;
  }
  broadcastAction(action, payload = {}) {
    this.broadcast({
      type: "action",
      tabId: this.tabId,
      timestamp: Date.now(),
      payload: { action, payload }
    });
  }
  getIsLeader() {
    return this.isLeader;
  }
  getTabId() {
    return this.tabId;
  }
  getActiveTabCount() {
    return this.tabs.size + 1;
  }
  destroy() {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
    if (this.channel) {
      this.broadcast({
        type: "ws-disconnected",
        tabId: this.tabId,
        timestamp: Date.now()
      });
      this.channel.close();
      this.channel = null;
    }
    this.tabs.clear();
    this.stateRunes.clear();
  }
}
function createTabSync(config) {
  return new WSTabSync(config);
}
var globalTabSync = null;
function getTabSync(config) {
  if (!globalTabSync) {
    globalTabSync = new WSTabSync(config);
  }
  return globalTabSync;
}
function destroyTabSync() {
  if (globalTabSync) {
    globalTabSync.destroy();
    globalTabSync = null;
  }
}
// client/src/indexeddb.ts
class IndexedDBPersistence {
  db = null;
  config;
  initPromise = null;
  constructor(config = {}) {
    this.config = {
      dbName: config.dbName ?? "gospa-state",
      version: config.version ?? 1,
      storeName: config.storeName ?? "state",
      autoCleanup: config.autoCleanup ?? true,
      maxAge: config.maxAge ?? 7 * 24 * 60 * 60 * 1000
    };
  }
  init() {
    if (this.initPromise)
      return this.initPromise;
    this.initPromise = new Promise((resolve, reject) => {
      if (typeof indexedDB === "undefined") {
        reject(new Error("IndexedDB not available"));
        return;
      }
      const request = indexedDB.open(this.config.dbName, this.config.version);
      request.onerror = () => {
        reject(new Error(`Failed to open IndexedDB: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        this.db = request.result;
        if (typeof process !== "undefined" && process.env?.NODE_ENV !== "production") {
          console.log(`[GoSPA IndexedDB] Database opened: ${this.config.dbName}`);
        }
        if (this.config.autoCleanup) {
          this.cleanup().catch(console.error);
        }
        resolve();
      };
      request.onupgradeneeded = (event) => {
        const db = event.target.result;
        if (!db.objectStoreNames.contains(this.config.storeName)) {
          const store = db.createObjectStore(this.config.storeName, {
            keyPath: "key"
          });
          store.createIndex("timestamp", "timestamp", { unique: false });
          store.createIndex("expiresAt", "expiresAt", { unique: false });
          if (typeof process !== "undefined" && process.env?.NODE_ENV !== "production") {
            console.log(`[GoSPA IndexedDB] Created store: ${this.config.storeName}`);
          }
        }
      };
    });
    return this.initPromise;
  }
  async get(key) {
    await this.init();
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }
      const transaction = this.db.transaction(this.config.storeName, "readonly");
      const store = transaction.objectStore(this.config.storeName);
      const request = store.get(key);
      request.onerror = () => {
        reject(new Error(`Failed to get key ${key}: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        const entry = request.result;
        if (!entry) {
          resolve(null);
          return;
        }
        if (entry.expiresAt && Date.now() > entry.expiresAt) {
          this.delete(key).catch(console.error);
          resolve(null);
          return;
        }
        resolve(entry.value);
      };
    });
  }
  async set(key, value, ttl) {
    await this.init();
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }
      const entry = {
        key,
        value,
        timestamp: Date.now(),
        expiresAt: ttl ? Date.now() + ttl : undefined
      };
      const transaction = this.db.transaction(this.config.storeName, "readwrite");
      const store = transaction.objectStore(this.config.storeName);
      const request = store.put(entry);
      request.onerror = () => {
        reject(new Error(`Failed to set key ${key}: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        resolve();
      };
    });
  }
  async delete(key) {
    await this.init();
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }
      const transaction = this.db.transaction(this.config.storeName, "readwrite");
      const store = transaction.objectStore(this.config.storeName);
      const request = store.delete(key);
      request.onerror = () => {
        reject(new Error(`Failed to delete key ${key}: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        resolve();
      };
    });
  }
  async keys() {
    await this.init();
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }
      const transaction = this.db.transaction(this.config.storeName, "readonly");
      const store = transaction.objectStore(this.config.storeName);
      const request = store.getAllKeys();
      request.onerror = () => {
        reject(new Error(`Failed to get keys: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        resolve(request.result);
      };
    });
  }
  async clear() {
    await this.init();
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }
      const transaction = this.db.transaction(this.config.storeName, "readwrite");
      const store = transaction.objectStore(this.config.storeName);
      const request = store.clear();
      request.onerror = () => {
        reject(new Error(`Failed to clear store: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        if (typeof process !== "undefined" && process.env?.NODE_ENV !== "production") {
          console.log(`[GoSPA IndexedDB] Cleared store: ${this.config.storeName}`);
        }
        resolve();
      };
    });
  }
  async cleanup() {
    await this.init();
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }
      const transaction = this.db.transaction(this.config.storeName, "readwrite");
      const store = transaction.objectStore(this.config.storeName);
      const index = store.index("expiresAt");
      const now = Date.now();
      let deletedCount = 0;
      const request = index.openCursor(IDBKeyRange.upperBound(now));
      request.onerror = () => {
        reject(new Error(`Failed to cleanup: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        const cursor = request.result;
        if (cursor) {
          cursor.delete();
          deletedCount++;
          cursor.continue();
        } else {
          if (deletedCount > 0 && typeof process !== "undefined" && process.env?.NODE_ENV !== "production") {
            console.log(`[GoSPA IndexedDB] Cleaned up ${deletedCount} expired entries`);
          }
          resolve(deletedCount);
        }
      };
    });
  }
  async getSize() {
    await this.init();
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }
      const transaction = this.db.transaction(this.config.storeName, "readonly");
      const store = transaction.objectStore(this.config.storeName);
      const countRequest = store.count();
      let entries = 0;
      countRequest.onerror = () => {
        reject(new Error(`Failed to count entries: ${countRequest.error?.message}`));
      };
      countRequest.onsuccess = () => {
        entries = countRequest.result;
        const getAllRequest = store.getAll();
        getAllRequest.onerror = () => {
          resolve({ entries, bytes: 0 });
        };
        getAllRequest.onsuccess = () => {
          const data2 = getAllRequest.result;
          const bytes = new Blob([JSON.stringify(data2)]).size;
          resolve({ entries, bytes });
        };
      };
    });
  }
  close() {
    if (this.db) {
      this.db.close();
      this.db = null;
      this.initPromise = null;
      if (typeof process !== "undefined" && process.env?.NODE_ENV !== "production") {
        console.log(`[GoSPA IndexedDB] Database closed: ${this.config.dbName}`);
      }
    }
  }
  async deleteDatabase() {
    this.close();
    return new Promise((resolve, reject) => {
      const request = indexedDB.deleteDatabase(this.config.dbName);
      request.onerror = () => {
        reject(new Error(`Failed to delete database: ${request.error?.message}`));
      };
      request.onsuccess = () => {
        if (typeof process !== "undefined" && process.env?.NODE_ENV !== "production") {
          console.log(`[GoSPA IndexedDB] Database deleted: ${this.config.dbName}`);
        }
        resolve();
      };
    });
  }
}
function createIndexedDBPersistence(config) {
  return new IndexedDBPersistence(config);
}
var globalPersistence = null;
function getIndexedDBPersistence(config) {
  if (!globalPersistence) {
    globalPersistence = new IndexedDBPersistence(config);
  }
  return globalPersistence;
}
function destroyIndexedDBPersistence() {
  if (globalPersistence) {
    globalPersistence.close();
    globalPersistence = null;
  }
}
// client/src/a11y.ts
class ScreenReaderAnnouncer {
  container = null;
  config;
  announceTimer = null;
  pendingAnnouncements = [];
  constructor(config = {}) {
    this.config = {
      announceNavigation: config.announceNavigation ?? true,
      announceStateChanges: config.announceStateChanges ?? false,
      politeness: config.politeness ?? "polite"
    };
    if (typeof document !== "undefined") {
      this.init();
    }
  }
  init() {
    this.container = document.getElementById("gospa-announcer");
    if (!this.container) {
      this.container = document.createElement("div");
      this.container.id = "gospa-announcer";
      this.container.setAttribute("aria-live", this.config.politeness);
      this.container.setAttribute("aria-atomic", "true");
      this.container.setAttribute("role", "status");
      this.container.style.cssText = `
				position: absolute;
				width: 1px;
				height: 1px;
				padding: 0;
				margin: -1px;
				overflow: hidden;
				clip: rect(0, 0, 0, 0);
				white-space: nowrap;
				border: 0;
			`;
      document.body.appendChild(this.container);
    }
  }
  announce(message, priority) {
    if (!this.container) {
      this.init();
    }
    if (priority && priority !== this.config.politeness) {
      this.container?.setAttribute("aria-live", priority);
    }
    if (this.announceTimer) {
      clearTimeout(this.announceTimer);
    }
    this.pendingAnnouncements.push(message);
    this.announceTimer = setTimeout(() => {
      const announcement = this.pendingAnnouncements.join(". ");
      this.pendingAnnouncements = [];
      if (this.container) {
        this.container.textContent = "";
        requestAnimationFrame(() => {
          if (this.container) {
            this.container.textContent = announcement;
          }
        });
      }
      if (priority && priority !== this.config.politeness) {
        this.container?.setAttribute("aria-live", this.config.politeness);
      }
    }, 100);
  }
  announceNavigation(path, title) {
    if (!this.config.announceNavigation)
      return;
    const message = title ? `Navigated to ${title}` : `Navigated to ${path}`;
    this.announce(message);
  }
  announceStateChange(key, value) {
    if (!this.config.announceStateChanges)
      return;
    const valueStr = typeof value === "object" ? JSON.stringify(value) : String(value);
    this.announce(`${key} changed to ${valueStr}`);
  }
  announceLoading(message = "Loading") {
    this.announce(message, "assertive");
  }
  announceError(message) {
    this.announce(`Error: ${message}`, "assertive");
  }
  announceSuccess(message) {
    this.announce(message);
  }
  destroy() {
    if (this.announceTimer) {
      clearTimeout(this.announceTimer);
    }
    if (this.container) {
      this.container.remove();
      this.container = null;
    }
    this.pendingAnnouncements = [];
  }
}
var aria = {
  setAttributes(element, attributes) {
    for (const [key, value] of Object.entries(attributes)) {
      if (value === null || value === false) {
        element.removeAttribute(key);
      } else if (value === true) {
        element.setAttribute(key, "");
      } else {
        element.setAttribute(key, String(value));
      }
    }
  },
  makeFocusable(element, tabIndex = 0) {
    element.setAttribute("tabindex", String(tabIndex));
  },
  label(element, label) {
    element.setAttribute("aria-label", label);
  },
  describe(element, descriptionId) {
    element.setAttribute("aria-describedby", descriptionId);
  },
  expanded(element, expanded) {
    element.setAttribute("aria-expanded", String(expanded));
  },
  hidden(element, hidden) {
    if (hidden) {
      element.setAttribute("aria-hidden", "true");
    } else {
      element.removeAttribute("aria-hidden");
    }
  },
  selected(element, selected) {
    element.setAttribute("aria-selected", String(selected));
  },
  checked(element, checked) {
    element.setAttribute("aria-checked", String(checked));
  },
  disabled(element, disabled) {
    element.setAttribute("aria-disabled", String(disabled));
  },
  busy(element, busy) {
    element.setAttribute("aria-busy", String(busy));
  },
  live(element, politeness) {
    element.setAttribute("aria-live", politeness);
  },
  createDescription(id, text) {
    const el = document.createElement("div");
    el.id = id;
    el.className = "gospa-sr-only";
    el.textContent = text;
    el.style.cssText = `
			position: absolute;
			width: 1px;
			height: 1px;
			padding: 0;
			margin: -1px;
			overflow: hidden;
			clip: rect(0, 0, 0, 0);
			white-space: nowrap;
			border: 0;
		`;
    return el;
  }
};
var focus = {
  trap(element) {
    const focusableSelectors = [
      "a[href]",
      "button:not([disabled])",
      "input:not([disabled])",
      "textarea:not([disabled])",
      "select:not([disabled])",
      '[tabindex]:not([tabindex="-1"])'
    ].join(", ");
    const focusableElements = Array.from(element.querySelectorAll(focusableSelectors));
    if (focusableElements.length === 0)
      return () => {};
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    const handleKeyDown = (event) => {
      const keyEvent = event;
      if (keyEvent.key !== "Tab")
        return;
      if (keyEvent.shiftKey) {
        if (document.activeElement === firstElement) {
          keyEvent.preventDefault();
          lastElement.focus();
        }
      } else {
        if (document.activeElement === lastElement) {
          keyEvent.preventDefault();
          firstElement.focus();
        }
      }
    };
    element.addEventListener("keydown", handleKeyDown);
    firstElement.focus();
    return () => {
      element.removeEventListener("keydown", handleKeyDown);
    };
  },
  restore(element) {
    if (element && element instanceof HTMLElement) {
      element.focus();
    }
  },
  save() {
    const activeElement = document.activeElement;
    return () => this.restore(activeElement);
  },
  moveTo(element) {
    if (element instanceof HTMLElement) {
      element.focus();
    }
  }
};
function createAnnouncer(config) {
  return new ScreenReaderAnnouncer(config);
}
var globalAnnouncer = null;
function getAnnouncer(config) {
  if (!globalAnnouncer) {
    globalAnnouncer = new ScreenReaderAnnouncer(config);
  }
  return globalAnnouncer;
}
function destroyAnnouncer() {
  if (globalAnnouncer) {
    globalAnnouncer.destroy();
    globalAnnouncer = null;
  }
}
function announce(message, priority) {
  getAnnouncer().announce(message, priority);
}
// client/src/performance.ts
class PerformanceMonitor {
  metrics = [];
  marks = new Map;
  config;
  observers = new Set;
  constructor(config = {}) {
    this.config = {
      enabled: config.enabled ?? (typeof process !== "undefined" && process.env?.NODE_ENV !== "production"),
      maxMetrics: config.maxMetrics ?? 1000,
      sampleRate: config.sampleRate ?? 1,
      enableConsoleLog: config.enableConsoleLog ?? false
    };
  }
  isEnabled() {
    if (!this.config.enabled)
      return false;
    if (this.config.sampleRate < 1 && Math.random() > this.config.sampleRate) {
      return false;
    }
    return true;
  }
  start(name) {
    if (!this.isEnabled())
      return;
    const markName = `gospa:${name}:start`;
    this.marks.set(name, performance.now());
    if (typeof performance !== "undefined" && performance.mark) {
      performance.mark(markName);
    }
  }
  end(name, metadata) {
    if (!this.isEnabled())
      return null;
    const startTime = this.marks.get(name);
    if (startTime === undefined) {
      console.warn(`[GoSPA Performance] No start mark found for: ${name}`);
      return null;
    }
    const endTime = performance.now();
    const duration = endTime - startTime;
    this.marks.delete(name);
    const metric = {
      name,
      duration,
      timestamp: Date.now(),
      metadata
    };
    this.addMetric(metric);
    if (typeof performance !== "undefined" && performance.measure) {
      try {
        const startMark = `gospa:${name}:start`;
        const endMark = `gospa:${name}:end`;
        performance.mark(endMark);
        performance.measure(`gospa:${name}`, startMark, endMark);
        performance.clearMarks(startMark);
        performance.clearMarks(endMark);
      } catch {}
    }
    return duration;
  }
  measure(name, fn, metadata) {
    if (!this.isEnabled()) {
      return fn();
    }
    this.start(name);
    try {
      const result = fn();
      this.end(name, metadata);
      return result;
    } catch (error) {
      this.end(name, { ...metadata, error: true });
      throw error;
    }
  }
  async measureAsync(name, fn, metadata) {
    if (!this.isEnabled()) {
      return fn();
    }
    this.start(name);
    try {
      const result = await fn();
      this.end(name, metadata);
      return result;
    } catch (error) {
      this.end(name, { ...metadata, error: true });
      throw error;
    }
  }
  addMetric(metric) {
    this.metrics.push(metric);
    if (this.metrics.length > this.config.maxMetrics) {
      this.metrics = this.metrics.slice(-this.config.maxMetrics);
    }
    for (const observer of this.observers) {
      try {
        observer(metric);
      } catch (error) {
        console.error("[GoSPA Performance] Observer error:", error);
      }
    }
    if (this.config.enableConsoleLog) {
      console.log(`[GoSPA Performance] ${metric.name}: ${metric.duration.toFixed(2)}ms`, metric.metadata);
    }
  }
  getMetrics() {
    return [...this.metrics];
  }
  getMetricsByName(name) {
    return this.metrics.filter((m) => m.name === name);
  }
  getAverageDuration(name) {
    const metrics = this.getMetricsByName(name);
    if (metrics.length === 0)
      return 0;
    const total = metrics.reduce((sum, m) => sum + m.duration, 0);
    return total / metrics.length;
  }
  getSummary() {
    const summary = {};
    for (const metric of this.metrics) {
      if (!summary[metric.name]) {
        summary[metric.name] = {
          count: 0,
          avg: 0,
          min: Infinity,
          max: -Infinity
        };
      }
      const s = summary[metric.name];
      s.count++;
      s.min = Math.min(s.min, metric.duration);
      s.max = Math.max(s.max, metric.duration);
    }
    for (const name of Object.keys(summary)) {
      const metrics = this.getMetricsByName(name);
      const total = metrics.reduce((sum, m) => sum + m.duration, 0);
      summary[name].avg = total / metrics.length;
    }
    return summary;
  }
  subscribe(observer) {
    this.observers.add(observer);
    return () => this.observers.delete(observer);
  }
  clear() {
    this.metrics = [];
    this.marks.clear();
  }
  getMemoryUsage() {
    if (typeof performance !== "undefined" && "memory" in performance) {
      const memory = performance.memory;
      return {
        used: memory.usedJSHeapSize,
        total: memory.totalJSHeapSize
      };
    }
    return null;
  }
  async getWebVitals() {
    const vitals = {};
    if (typeof performance !== "undefined" && performance.getEntriesByType) {
      const paintEntries = performance.getEntriesByType("paint");
      for (const entry of paintEntries) {
        if (entry.name === "first-contentful-paint") {
          vitals["FCP"] = entry.startTime;
        }
      }
      const lcpEntries = performance.getEntriesByType("largest-contentful-paint");
      if (lcpEntries.length > 0) {
        vitals["LCP"] = lcpEntries[lcpEntries.length - 1].startTime;
      }
      const fidEntries = performance.getEntriesByType("first-input");
      if (fidEntries.length > 0) {
        const fid = fidEntries[0];
        vitals["FID"] = fid.processingStart - fid.startTime;
      }
      const clsEntries = performance.getEntriesByType("layout-shift");
      let clsValue = 0;
      for (const entry of clsEntries) {
        if (!entry.hadRecentInput) {
          clsValue += entry.value;
        }
      }
      vitals["CLS"] = clsValue;
    }
    return vitals;
  }
}
function createPerformanceMonitor(config) {
  return new PerformanceMonitor(config);
}
var globalMonitor = null;
function getPerformanceMonitor(config) {
  if (!globalMonitor) {
    globalMonitor = new PerformanceMonitor(config);
  }
  return globalMonitor;
}
function destroyPerformanceMonitor() {
  if (globalMonitor) {
    globalMonitor.clear();
    globalMonitor = null;
  }
}
function measure(name, fn, metadata) {
  return getPerformanceMonitor().measure(name, fn, metadata);
}
function measureAsync(name, fn, metadata) {
  return getPerformanceMonitor().measureAsync(name, fn, metadata);
}
export {
  withErrorBoundary,
  watchProp,
  updateDevToolsPanel,
  untrack,
  unregisterBinding,
  transitionOut,
  transitionIn,
  transformers,
  toggleDevTools,
  toRaw,
  timing,
  throttle,
  syncedRune,
  syncBatch,
  slide,
  setupTransitions,
  setupEventDelegation,
  setSanitizer,
  setNavigationOptions,
  sendAction,
  scale,
  sanitizeSync,
  sanitizeHtml,
  sanitize,
  resourceReactive,
  renderList,
  renderIf,
  removeClass,
  remoteAction,
  remote,
  registerBinding,
  reactiveArray,
  reactive,
  querySelectorAll,
  preloadSanitizer,
  prefetch,
  parseEventString,
  onKey,
  onComponentError,
  onBeforeNavigate,
  onAfterNavigate,
  on,
  offAll,
  navigate,
  memoryUsage,
  measureAsync,
  measure,
  linear,
  keys,
  isSanitizerReady,
  isReactive,
  isNavigating,
  isInErrorState,
  isDev,
  inspect,
  initWebSocket,
  initStreaming,
  initPriorityHydration,
  initNavigation,
  initIslands,
  hydrateIsland,
  hasClass,
  go,
  getWebSocketClient,
  getTabSync,
  getStreamingManager,
  getStore,
  getRemotePrefix,
  getPriorityScheduler,
  getPerformanceMonitor,
  getIslandManager,
  getIndexedDBPersistence,
  getErrorBoundaryState,
  getCurrentPath,
  getAnnouncer,
  forward,
  focus,
  fly,
  flushDOMUpdatesNow,
  findAll,
  find,
  fade,
  elasticOut,
  effect,
  domPurifySanitizer,
  destroyTabSync,
  destroyPerformanceMonitor,
  destroyNavigation,
  destroyIndexedDBPersistence,
  destroyAnnouncer,
  derived,
  delegate,
  debugLog,
  debounce,
  data,
  cubicOut,
  cubicInOut,
  crossfade,
  createTabSync,
  createStore,
  createPerformanceMonitor,
  createNavigationState,
  createInspector,
  createIndexedDBPersistence,
  createErrorFallback,
  createElement,
  createDevToolsPanel,
  createAnnouncer,
  configureRemote,
  clearAllErrorBoundaries,
  cancelPendingDOMUpdates,
  bounceOut,
  blur,
  bindTwoWay,
  bindEvent,
  bindElement,
  bindDerived,
  batch,
  back,
  attr,
  aria,
  applyStateUpdate,
  announce,
  addClass,
  WSTabSync,
  WSClient,
  StreamingManager,
  SharedStore,
  ScreenReaderAnnouncer,
  Rune,
  Resource,
  PriorityScheduler,
  PerformanceMonitor,
  PURIFY_CONFIG,
  PRIORITY_NORMAL,
  PRIORITY_MAP,
  PRIORITY_LOW,
  PRIORITY_HIGH,
  PRIORITY_DEFERRED,
  PRIORITY_CRITICAL,
  IslandManager,
  IndexedDBPersistence,
  Effect,
  Derived,
  $state,
  $effect,
  $derived
};
