import { H as xX, I as AX, J as TX } from "./island-fs1jd09z.js";
import {
  $ as bJ,
  Aa as bX,
  K as S,
  L as RJ,
  M as BJ,
  N as xJ,
  O as AJ,
  P as TJ,
  Q as PJ,
  R as MJ,
  S as EJ,
  T as jJ,
  U as qJ,
  V as SJ,
  W as IJ,
  X as pJ,
  Y as vJ,
  Z as hJ,
  _ as kJ,
  ba as ZX,
  ca as $X,
  da as GX,
  ea as _X,
  fa as LX,
  ga as QX,
  ha as UX,
  ia as VX,
  ja as KX,
  ka as OX,
  la as zX,
  ma as HX,
  na as FX,
  pa as PX,
  qa as MX,
  ra as EX,
  sa as jX,
  ta as qX,
  ua as SX,
  va as IX,
  wa as pX,
  xa as vX,
  ya as hX,
  za as kX,
} from "./runtime-core.js";
import {
  Ba as fJ,
  Ca as yJ,
  Da as uJ,
  Ea as gJ,
  Fa as mJ,
  Ga as cJ,
} from "./websocket-3byh9n94.js";
import {
  $a as DJ,
  Ha as _J,
  Ia as LJ,
  Ja as QJ,
  Ka as UJ,
  La as VJ,
  Ma as KJ,
  Ra as D,
  Sa as OJ,
  Ta as C,
  Ua as zJ,
  Va as HJ,
  Wa as FJ,
  Xa as WJ,
  Ya as wJ,
  Za as NJ,
  _a as CJ,
} from "./runtime-wsmb0jca.js";
import {
  ab as dJ,
  bb as oJ,
  cb as lJ,
  db as iJ,
  eb as rJ,
  fb as sJ,
  gb as aJ,
  hb as tJ,
  ib as nJ,
  jb as eJ,
  kb as JX,
  lb as XX,
  mb as YX,
} from "./navigation-5sm41hnk.js";
import {
  nb as WX,
  ob as wX,
  pb as NX,
  qb as CX,
  rb as DX,
  sb as RX,
  tb as BX,
} from "./transition-nqs87scc.js";
import "./runtime-m2remsqk.js";
var h = {
  maxConcurrent: 3,
  idleTimeout: 2000,
  intersectionThreshold: 0.1,
  intersectionRootMargin: "50px",
  enablePreload: !0,
};
class j {
  config;
  islands = new Map();
  hydrationQueue = [];
  activeHydrations = 0;
  observers = new Map();
  idleCallbacks = new Map();
  interactionHandlers = new Map();
  constructor(J = {}) {
    this.config = { ...h, ...J };
  }
  registerPlan(J) {
    if (this.config.enablePreload) this.preloadScripts(J.preload);
    for (let X of J.immediate) this.registerIsland(X, "immediate");
    for (let X of J.idle) this.registerIsland(X, "idle");
    for (let X of J.visible) this.registerIsland(X, "visible");
    for (let X of J.interaction) this.registerIsland(X, "interaction");
    for (let X of J.lazy) this.registerIsland(X, "lazy");
    this.processQueue();
  }
  registerIsland(J, X) {
    let Y = { ...J, state: "pending", mode: X };
    this.islands.set(J.id, Y);
    let Z = document.querySelector(`[data-island-id="${J.id}"]`);
    if (Z) Y.element = Z;
    this.setupHydrationTrigger(Y);
  }
  setupHydrationTrigger(J) {
    switch (J.mode) {
      case "immediate":
        this.hydrationQueue.push(J);
        break;
      case "idle":
        this.setupIdleHydration(J);
        break;
      case "visible":
        this.setupVisibleHydration(J);
        break;
      case "interaction":
        this.setupInteractionHydration(J);
        break;
      case "lazy":
        break;
    }
  }
  setupIdleHydration(J) {
    if ("requestIdleCallback" in window) {
      let X = requestIdleCallback(
        () => {
          (this.hydrationQueue.push(J), this.processQueue());
        },
        { timeout: this.config.idleTimeout },
      );
      this.idleCallbacks.set(J.id, X);
    } else
      setTimeout(() => {
        (this.hydrationQueue.push(J), this.processQueue());
      }, this.config.idleTimeout);
  }
  setupVisibleHydration(J) {
    if (!J.element) {
      (this.hydrationQueue.push(J), this.processQueue());
      return;
    }
    let X = new IntersectionObserver(
      (Y) => {
        for (let Z of Y)
          if (Z.isIntersecting)
            (this.hydrationQueue.push(J),
              this.processQueue(),
              X.disconnect(),
              this.observers.delete(J.id));
      },
      {
        threshold: this.config.intersectionThreshold,
        rootMargin: this.config.intersectionRootMargin,
      },
    );
    (X.observe(J.element), this.observers.set(J.id, X));
  }
  setupInteractionHydration(J) {
    if (!J.element) {
      (this.hydrationQueue.push(J), this.processQueue());
      return;
    }
    let X = ["click", "focus", "mouseenter", "touchstart"],
      Y = [],
      Z = ($) => {
        for (let G = 0; G < X.length; G++)
          J.element.removeEventListener(X[G], Y[G]);
        (this.hydrationQueue.push(J), this.processQueue());
      };
    for (let $ of X) {
      let G = Z;
      (Y.push(G), J.element.addEventListener($, G, { passive: !0, once: !0 }));
    }
    this.interactionHandlers.set(J.id, Y);
  }
  processQueue() {
    this.hydrationQueue.sort((J, X) => {
      if (J.priority !== X.priority) return X.priority - J.priority;
      return J.position - X.position;
    });
    while (
      this.activeHydrations < this.config.maxConcurrent &&
      this.hydrationQueue.length > 0
    ) {
      let J = this.hydrationQueue.shift();
      if (J && J.state === "pending") this.hydrateIsland(J);
    }
  }
  async hydrateIsland(J) {
    ((J.state = "hydrating"), this.activeHydrations++);
    try {
      await this.waitForDependencies(J);
      let X = new CustomEvent("gospa:hydrate", {
        detail: { id: J.id, name: J.name, state: J.state },
      });
      (document.dispatchEvent(X), (J.state = "hydrated"));
      let Y = new CustomEvent("gospa:hydrated", {
        detail: { id: J.id, name: J.name },
      });
      document.dispatchEvent(Y);
    } catch (X) {
      ((J.state = "error"),
        (J.error = X instanceof Error ? X : Error(String(X))));
      let Y = new CustomEvent("gospa:hydration-error", {
        detail: { id: J.id, error: J.error },
      });
      document.dispatchEvent(Y);
    } finally {
      (this.activeHydrations--, this.processQueue());
    }
  }
  async waitForDependencies(J) {
    if (!J.dependencies || J.dependencies.length === 0) return;
    let X = J.dependencies.map((Y) => {
      return new Promise((Z) => {
        let $ = this.islands.get(Y);
        if (!$ || $.state === "hydrated") {
          Z();
          return;
        }
        let G = (_) => {
          if (_.detail.id === Y)
            (document.removeEventListener("gospa:hydrated", G), Z());
        };
        document.addEventListener("gospa:hydrated", G);
      });
    });
    await Promise.all(X);
  }
  preloadScripts(J) {
    for (let X of J) {
      let Y = document.createElement("link");
      ((Y.rel = "preload"),
        (Y.as = "script"),
        (Y.href = X),
        document.head.appendChild(Y));
    }
  }
  forceHydrate(J) {
    let X = this.islands.get(J);
    if (X && X.state === "pending")
      (this.cancelTriggers(J),
        this.hydrationQueue.push(X),
        this.processQueue());
  }
  cancelTriggers(J) {
    let X = this.idleCallbacks.get(J);
    if (X !== void 0) (cancelIdleCallback(X), this.idleCallbacks.delete(J));
    let Y = this.observers.get(J);
    if (Y) (Y.disconnect(), this.observers.delete(J));
    let Z = this.interactionHandlers.get(J);
    if (Z) {
      let $ = this.islands.get(J);
      if ($?.element) {
        let G = ["click", "focus", "mouseenter", "touchstart"];
        for (let _ = 0; _ < G.length; _++)
          $.element.removeEventListener(G[_], Z[_]);
      }
      this.interactionHandlers.delete(J);
    }
  }
  getIslandState(J) {
    return this.islands.get(J)?.state;
  }
  getPendingIslands() {
    return Array.from(this.islands.values()).filter(
      (J) => J.state === "pending",
    );
  }
  getHydratedIslands() {
    return Array.from(this.islands.values()).filter(
      (J) => J.state === "hydrated",
    );
  }
  getStats() {
    let J = Array.from(this.islands.values());
    return {
      total: J.length,
      pending: J.filter((X) => X.state === "pending").length,
      hydrating: J.filter((X) => X.state === "hydrating").length,
      hydrated: J.filter((X) => X.state === "hydrated").length,
      errors: J.filter((X) => X.state === "error").length,
    };
  }
  destroy() {
    for (let J of this.idleCallbacks.values()) cancelIdleCallback(J);
    this.idleCallbacks.clear();
    for (let J of this.observers.values()) J.disconnect();
    (this.observers.clear(),
      this.interactionHandlers.clear(),
      this.islands.clear(),
      (this.hydrationQueue = []));
  }
}
var T = null;
function q(J) {
  if (!T) T = new j(J);
  return T;
}
function k(J) {
  let X = q();
  return (X.registerPlan(J), X);
}
class I {
  islands = [];
  hydrationQueue = [];
  hydratedIslands = new Set();
  isHydrating = !1;
  options;
  constructor(J = {}) {
    ((this.options = { enableLogging: !1, hydrationTimeout: 30000, ...J }),
      this.setupStreamHandler());
  }
  setupStreamHandler() {
    let J = globalThis.__GOSPA_STREAM__;
    globalThis.__GOSPA_STREAM__ = (X) => {
      if (typeof J === "function") J(X);
      this.processChunk(X);
    };
  }
  processChunk(J) {
    if (this.options.enableLogging)
      console.log("[GoSPA Stream]", J.type, J.id || "", J);
    switch (J.type) {
      case "html":
        this.handleHtmlChunk(J);
        break;
      case "island":
        this.handleIslandChunk(J);
        break;
      case "script":
        this.handleScriptChunk(J);
        break;
      case "state":
        this.handleStateChunk(J);
        break;
      case "error":
        this.handleErrorChunk(J);
        break;
    }
  }
  handleHtmlChunk(J) {
    let X = document.getElementById(J.id);
    if (X) {
      let Y = S(J.content);
      if (Y instanceof Promise)
        Y.then((Z) => {
          X.innerHTML = Z;
        });
      else X.innerHTML = Y;
      X.dispatchEvent(
        new CustomEvent("gospa:html-update", {
          detail: { id: J.id, content: J.content },
        }),
      );
    }
  }
  handleIslandChunk(J) {
    let X = J.data;
    if (!X || !X.id) {
      console.error("[GoSPA Stream] Invalid island data:", J);
      return;
    }
    (this.islands.push(X), this.queueHydration(X));
  }
  handleScriptChunk(J) {
    let X = document.createElement("script");
    ((X.textContent = J.content), document.head.appendChild(X));
  }
  handleStateChunk(J) {
    let X = (globalThis.__GOSPA_STATE__ ||= {});
    ((X[J.id] = J.data),
      document.dispatchEvent(
        new CustomEvent("gospa:state-update", {
          detail: { id: J.id, state: J.data },
        }),
      ));
  }
  handleErrorChunk(J) {
    (console.error("[GoSPA Stream Error]", J.content),
      document.dispatchEvent(
        new CustomEvent("gospa:stream-error", { detail: { error: J.content } }),
      ));
  }
  queueHydration(J) {
    switch (J.mode) {
      case "immediate":
        this.hydrateImmediate(J);
        break;
      case "visible":
        this.hydrateOnVisible(J);
        break;
      case "idle":
        this.hydrateOnIdle(J);
        break;
      case "interaction":
        this.hydrateOnInteraction(J);
        break;
      case "lazy":
        this.hydrateLazy(J);
        break;
      default:
        this.hydrateImmediate(J);
    }
  }
  hydrateImmediate(J) {
    this.addToHydrationQueue(J, "high");
  }
  hydrateOnVisible(J) {
    let X = document.querySelector(`[data-gospa-island="${J.id}"]`);
    if (!X) {
      this.hydrateImmediate(J);
      return;
    }
    let Y = new IntersectionObserver(
      (Z) => {
        for (let $ of Z)
          if ($.isIntersecting)
            (Y.disconnect(), this.addToHydrationQueue(J, "normal"));
      },
      { rootMargin: "100px" },
    );
    Y.observe(X);
  }
  hydrateOnIdle(J) {
    if ("requestIdleCallback" in globalThis)
      globalThis.requestIdleCallback(() => {
        this.addToHydrationQueue(J, "low");
      });
    else
      setTimeout(() => {
        this.addToHydrationQueue(J, "low");
      }, 100);
  }
  hydrateOnInteraction(J) {
    let X = document.querySelector(`[data-gospa-island="${J.id}"]`);
    if (!X) {
      this.hydrateImmediate(J);
      return;
    }
    let Y = ["mouseenter", "touchstart", "focusin", "click"],
      Z = () => {
        (Y.forEach(($) => X.removeEventListener($, Z)),
          this.addToHydrationQueue(J, "high"));
      };
    Y.forEach(($) => {
      X.addEventListener($, Z, { once: !0, passive: !0 });
    });
  }
  hydrateLazy(J) {
    if (document.readyState === "complete") this.hydrateOnIdle(J);
    else
      globalThis.addEventListener("load", () => {
        setTimeout(() => {
          this.hydrateOnIdle(J);
        }, 500);
      });
  }
  addToHydrationQueue(J, X) {
    if (this.hydratedIslands.has(J.id)) return;
    let Y = { island: J, resolve: () => {}, reject: () => {} };
    if (X === "high") this.hydrationQueue.unshift(Y);
    else this.hydrationQueue.push(Y);
    this.processQueue();
  }
  processQueue() {
    if (this.isHydrating || this.hydrationQueue.length === 0) return;
    this.isHydrating = !0;
    let J = this.hydrationQueue.shift();
    if (J)
      this.hydrateIsland(J.island)
        .then(() => {
          (this.hydratedIslands.add(J.island.id),
            (this.isHydrating = !1),
            this.processQueue());
        })
        .catch((X) => {
          (console.error("[GoSPA] Hydration error:", X),
            (this.isHydrating = !1),
            this.processQueue());
        });
  }
  async hydrateIsland(J) {
    let X = document.querySelector(`[data-gospa-island="${J.id}"]`);
    if (!X) {
      if (this.options.enableLogging)
        console.warn("[GoSPA] Island element not found:", J.id);
      return;
    }
    let Y = globalThis.__GOSPA_ISLAND_MANAGER__;
    if (Y && typeof Y.hydrate === "function") await Y.hydrate(J.id, J);
    if (
      (X.dispatchEvent(
        new CustomEvent("gospa:hydrated", { detail: { island: J } }),
      ),
      this.options.enableLogging)
    )
      console.log("[GoSPA] Hydrated island:", J.id, J.name);
  }
  getIslands() {
    return [...this.islands];
  }
  getHydratedIslands() {
    return new Set(this.hydratedIslands);
  }
  isHydrated(J) {
    return this.hydratedIslands.has(J);
  }
  async hydrate(J) {
    let X = this.islands.find((Y) => Y.id === J);
    if (X) await this.hydrateIsland(X);
  }
}
var N = null;
function p(J) {
  if (!N) N = new I(J);
  return N;
}
function b() {
  return N;
}
if (typeof window < "u")
  setTimeout(() => {
    if (!N) p();
  }, 0);
class P {
  _status;
  _data;
  _error;
  _fetcher;
  constructor(J) {
    ((this._fetcher = J),
      (this._status = C("idle")),
      (this._data = C(void 0)),
      (this._error = C(void 0)));
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
    if (this._status.peek() === "pending") return;
    (this._status.set("pending"), this._error.set(void 0));
    try {
      let J = await this._fetcher();
      return (this._data.set(J), this._status.set("success"), J);
    } catch (J) {
      throw (this._error.set(J), this._status.set("error"), J);
    }
  }
  async refetch() {
    return this.fetch();
  }
  reset() {
    (this._status.set("idle"), this._data.set(void 0), this._error.set(void 0));
  }
}
function f(J) {
  return new P(J);
}
var z = new Map(),
  M = new Set();
function y(J) {
  return (M.add(J), () => M.delete(J));
}
function u(J, X) {
  if (!z.has(J)) z.set(J, { hasError: !1, error: null, retryCount: 0 });
  let Y = () => z.get(J),
    Z = (L) => {
      let U = Y();
      ((U.hasError = !0), (U.error = L), X.onError?.(L, J));
      for (let K of M)
        try {
          K(L, J);
        } catch (O) {
          console.error("[GoSPA] Error in error handler:", O);
        }
      let V = document.querySelector(`[data-gospa-component="${J}"]`);
      if (V) {
        let K =
          typeof X.fallback === "function"
            ? X.fallback(L, J)
            : X.fallback.cloneNode(!0);
        if (
          (V.replaceChildren(K),
          X.retryable && U.retryCount < (X.maxRetries ?? 3))
        ) {
          let O = document.createElement("button");
          ((O.textContent = "Retry"),
            (O.className = "gospa-retry-btn"),
            (O.onclick = () => {
              (U.retryCount++,
                (U.hasError = !1),
                (U.error = null),
                V.dispatchEvent(
                  new CustomEvent("gospa:retry", {
                    detail: { componentId: J },
                  }),
                ));
            }),
            V.appendChild(O));
        }
      }
    };
  return {
    wrapMount: (L) => {
      return () => {
        if (Y().hasError) return () => {};
        try {
          return L();
        } catch (V) {
          return (Z(V), () => {});
        }
      };
    },
    wrapDestroy: (L) => {
      return () => {
        try {
          L();
        } catch (U) {
          console.error(`[GoSPA] Error destroying component ${J}:`, U);
        }
      };
    },
    wrapAction: (L) => {
      return (...U) => {
        let V = Y();
        if (V.hasError)
          throw Error(`Component ${J} is in error state: ${V.error?.message}`);
        try {
          return L(...U);
        } catch (K) {
          throw (Z(K), K);
        }
      };
    },
    clearError: () => {
      let L = Y();
      ((L.hasError = !1), (L.error = null), (L.retryCount = 0));
    },
    getState: Y,
  };
}
function g(J) {
  let X = document.createElement("div");
  ((X.className = "gospa-error-fallback"), X.setAttribute("role", "alert"));
  let Y = document.createElement("div");
  Y.className = "gospa-error-content";
  let Z = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  (Z.setAttribute("class", "gospa-error-icon"),
    Z.setAttribute("viewBox", "0 0 24 24"),
    Z.setAttribute("fill", "none"),
    Z.setAttribute("stroke", "currentColor"),
    Z.setAttribute("stroke-width", "2"));
  let $ = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  ($.setAttribute("cx", "12"),
    $.setAttribute("cy", "12"),
    $.setAttribute("r", "10"));
  let G = document.createElementNS("http://www.w3.org/2000/svg", "line");
  (G.setAttribute("x1", "12"),
    G.setAttribute("y1", "8"),
    G.setAttribute("x2", "12"),
    G.setAttribute("y2", "12"));
  let _ = document.createElementNS("http://www.w3.org/2000/svg", "line");
  (_.setAttribute("x1", "12"),
    _.setAttribute("y1", "16"),
    _.setAttribute("x2", "12.01"),
    _.setAttribute("y2", "16"),
    Z.appendChild($),
    Z.appendChild(G),
    Z.appendChild(_));
  let Q = document.createElement("p");
  return (
    (Q.className = "gospa-error-message"),
    (Q.textContent = J || "Something went wrong"),
    Y.appendChild(Z),
    Y.appendChild(Q),
    X.appendChild(Y),
    X
  );
}
function m(J) {
  return z.get(J);
}
function c() {
  for (let J of z.values())
    ((J.hasError = !1), (J.error = null), (J.retryCount = 0));
}
function d(J) {
  return z.get(J)?.hasError ?? !1;
}
class R {
  channel = null;
  tabId;
  tabs = new Map();
  isLeader = !1;
  pingTimer = null;
  config;
  stateRunes = new Map();
  onStateUpdate = null;
  onAction = null;
  constructor(J = {}) {
    if (
      ((this.tabId = `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`),
      (this.config = {
        channelName: J.channelName ?? "gospa-ws-sync",
        enabled: J.enabled ?? !0,
        pingInterval: J.pingInterval ?? 5000,
        tabTimeout: J.tabTimeout ?? 1e4,
      }),
      this.config.enabled && typeof BroadcastChannel < "u")
    )
      this.init();
  }
  init() {
    try {
      ((this.channel = new BroadcastChannel(this.config.channelName)),
        (this.channel.onmessage = (J) => this.handleMessage(J.data)),
        this.broadcast({
          type: "ping",
          tabId: this.tabId,
          timestamp: Date.now(),
        }),
        (this.pingTimer = setInterval(() => {
          (this.broadcast({
            type: "ping",
            tabId: this.tabId,
            timestamp: Date.now(),
          }),
            this.cleanupDeadTabs());
        }, this.config.pingInterval)),
        window.addEventListener("beforeunload", () => {
          this.broadcast({
            type: "ws-disconnected",
            tabId: this.tabId,
            timestamp: Date.now(),
          });
        }),
        console.log(`[GoSPA Tab Sync] Initialized with tab ID: ${this.tabId}`));
    } catch (J) {
      console.warn("[GoSPA Tab Sync] BroadcastChannel not available:", J);
    }
  }
  handleMessage(J) {
    if (J.tabId === this.tabId) return;
    switch (
      (this.tabs.set(J.tabId, {
        id: J.tabId,
        lastSeen: Date.now(),
        isLeader: !1,
      }),
      J.type)
    ) {
      case "ping":
        (this.broadcast({
          type: "pong",
          tabId: this.tabId,
          timestamp: Date.now(),
        }),
          this.electLeader());
        break;
      case "pong":
        this.electLeader();
        break;
      case "state-update":
        if (J.payload && typeof J.payload === "object") {
          let { key: X, value: Y } = J.payload,
            Z = this.stateRunes.get(X);
          if (Z)
            D(() => {
              Z.set(Y);
            });
          this.onStateUpdate?.(X, Y);
        }
        break;
      case "state-sync":
        if (J.payload && typeof J.payload === "object") {
          let X = J.payload;
          D(() => {
            for (let [Y, Z] of Object.entries(X)) {
              let $ = this.stateRunes.get(Y);
              if ($) $.set(Z);
            }
          });
        }
        break;
      case "action":
        if (J.payload && typeof J.payload === "object") {
          let { action: X, payload: Y } = J.payload;
          this.onAction?.(X, Y);
        }
        break;
      case "ws-connected":
        (console.log(`[GoSPA Tab Sync] Tab ${J.tabId} connected`),
          this.electLeader());
        break;
      case "ws-disconnected":
        (this.tabs.delete(J.tabId), this.electLeader());
        break;
    }
  }
  broadcast(J) {
    if (this.channel)
      try {
        this.channel.postMessage(J);
      } catch (X) {
        console.warn("[GoSPA Tab Sync] Failed to broadcast:", X);
      }
  }
  electLeader() {
    let J = Date.now(),
      X = null,
      Y = [
        { id: this.tabId, lastSeen: J, isLeader: !1 },
        ...Array.from(this.tabs.values()),
      ];
    for (let $ of Y) if (!X || $.lastSeen < X.lastSeen) X = $;
    let Z = this.isLeader;
    if (((this.isLeader = X?.id === this.tabId), this.isLeader && !Z))
      (console.log("[GoSPA Tab Sync] This tab is now the leader"),
        this.syncStateToTabs());
  }
  cleanupDeadTabs() {
    let J = Date.now();
    for (let [X, Y] of this.tabs)
      if (J - Y.lastSeen > this.config.tabTimeout)
        (this.tabs.delete(X),
          console.log(`[GoSPA Tab Sync] Removed dead tab: ${X}`));
    this.electLeader();
  }
  syncStateToTabs() {
    let J = {};
    for (let [X, Y] of this.stateRunes) J[X] = Y.get();
    this.broadcast({
      type: "state-sync",
      tabId: this.tabId,
      timestamp: Date.now(),
      payload: J,
    });
  }
  registerState(J, X) {
    (this.stateRunes.set(J, X),
      X.subscribe((Y) => {
        if (!this.isLeader) return;
        this.broadcast({
          type: "state-update",
          tabId: this.tabId,
          timestamp: Date.now(),
          payload: { key: J, value: Y },
        });
      }));
  }
  unregisterState(J) {
    this.stateRunes.delete(J);
  }
  onStateChange(J) {
    this.onStateUpdate = J;
  }
  onActionReceived(J) {
    this.onAction = J;
  }
  broadcastAction(J, X = {}) {
    this.broadcast({
      type: "action",
      tabId: this.tabId,
      timestamp: Date.now(),
      payload: { action: J, payload: X },
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
    if (this.pingTimer)
      (clearInterval(this.pingTimer), (this.pingTimer = null));
    if (this.channel)
      (this.broadcast({
        type: "ws-disconnected",
        tabId: this.tabId,
        timestamp: Date.now(),
      }),
        this.channel.close(),
        (this.channel = null));
    (this.tabs.clear(), this.stateRunes.clear());
  }
}
function o(J) {
  return new R(J);
}
var H = null;
function l(J) {
  if (!H) H = new R(J);
  return H;
}
function i() {
  if (H) (H.destroy(), (H = null));
}
class B {
  db = null;
  config;
  initPromise = null;
  constructor(J = {}) {
    this.config = {
      dbName: J.dbName ?? "gospa-state",
      version: J.version ?? 1,
      storeName: J.storeName ?? "state",
      autoCleanup: J.autoCleanup ?? !0,
      maxAge: J.maxAge ?? 604800000,
    };
  }
  init() {
    if (this.initPromise) return this.initPromise;
    return (
      (this.initPromise = new Promise((J, X) => {
        if (typeof indexedDB > "u") {
          X(Error("IndexedDB not available"));
          return;
        }
        let Y = indexedDB.open(this.config.dbName, this.config.version);
        ((Y.onerror = () => {
          X(Error(`Failed to open IndexedDB: ${Y.error?.message}`));
        }),
          (Y.onsuccess = () => {
            if (
              ((this.db = Y.result),
              typeof process < "u" && process.env?.NODE_ENV !== "production")
            )
              console.log(
                `[GoSPA IndexedDB] Database opened: ${this.config.dbName}`,
              );
            if (this.config.autoCleanup) this.cleanup().catch(console.error);
            J();
          }),
          (Y.onupgradeneeded = (Z) => {
            let $ = Z.target.result;
            if (!$.objectStoreNames.contains(this.config.storeName)) {
              let G = $.createObjectStore(this.config.storeName, {
                keyPath: "key",
              });
              if (
                (G.createIndex("timestamp", "timestamp", { unique: !1 }),
                G.createIndex("expiresAt", "expiresAt", { unique: !1 }),
                typeof process < "u" && process.env?.NODE_ENV !== "production")
              )
                console.log(
                  `[GoSPA IndexedDB] Created store: ${this.config.storeName}`,
                );
            }
          }));
      })),
      this.initPromise
    );
  }
  async get(J) {
    return (
      await this.init(),
      new Promise((X, Y) => {
        if (!this.db) {
          Y(Error("Database not initialized"));
          return;
        }
        let G = this.db
          .transaction(this.config.storeName, "readonly")
          .objectStore(this.config.storeName)
          .get(J);
        ((G.onerror = () => {
          Y(Error(`Failed to get key ${J}: ${G.error?.message}`));
        }),
          (G.onsuccess = () => {
            let _ = G.result;
            if (!_) {
              X(null);
              return;
            }
            if (_.expiresAt && Date.now() > _.expiresAt) {
              (this.delete(J).catch(console.error), X(null));
              return;
            }
            X(_.value);
          }));
      })
    );
  }
  async set(J, X, Y) {
    return (
      await this.init(),
      new Promise((Z, $) => {
        if (!this.db) {
          $(Error("Database not initialized"));
          return;
        }
        let G = {
            key: J,
            value: X,
            timestamp: Date.now(),
            expiresAt: Y ? Date.now() + Y : void 0,
          },
          L = this.db
            .transaction(this.config.storeName, "readwrite")
            .objectStore(this.config.storeName)
            .put(G);
        ((L.onerror = () => {
          $(Error(`Failed to set key ${J}: ${L.error?.message}`));
        }),
          (L.onsuccess = () => {
            Z();
          }));
      })
    );
  }
  async delete(J) {
    return (
      await this.init(),
      new Promise((X, Y) => {
        if (!this.db) {
          Y(Error("Database not initialized"));
          return;
        }
        let G = this.db
          .transaction(this.config.storeName, "readwrite")
          .objectStore(this.config.storeName)
          .delete(J);
        ((G.onerror = () => {
          Y(Error(`Failed to delete key ${J}: ${G.error?.message}`));
        }),
          (G.onsuccess = () => {
            X();
          }));
      })
    );
  }
  async keys() {
    return (
      await this.init(),
      new Promise((J, X) => {
        if (!this.db) {
          X(Error("Database not initialized"));
          return;
        }
        let $ = this.db
          .transaction(this.config.storeName, "readonly")
          .objectStore(this.config.storeName)
          .getAllKeys();
        (($.onerror = () => {
          X(Error(`Failed to get keys: ${$.error?.message}`));
        }),
          ($.onsuccess = () => {
            J($.result);
          }));
      })
    );
  }
  async clear() {
    return (
      await this.init(),
      new Promise((J, X) => {
        if (!this.db) {
          X(Error("Database not initialized"));
          return;
        }
        let $ = this.db
          .transaction(this.config.storeName, "readwrite")
          .objectStore(this.config.storeName)
          .clear();
        (($.onerror = () => {
          X(Error(`Failed to clear store: ${$.error?.message}`));
        }),
          ($.onsuccess = () => {
            if (typeof process < "u" && process.env?.NODE_ENV !== "production")
              console.log(
                `[GoSPA IndexedDB] Cleared store: ${this.config.storeName}`,
              );
            J();
          }));
      })
    );
  }
  async cleanup() {
    return (
      await this.init(),
      new Promise((J, X) => {
        if (!this.db) {
          X(Error("Database not initialized"));
          return;
        }
        let $ = this.db
            .transaction(this.config.storeName, "readwrite")
            .objectStore(this.config.storeName)
            .index("expiresAt"),
          G = Date.now(),
          _ = 0,
          Q = $.openCursor(IDBKeyRange.upperBound(G));
        ((Q.onerror = () => {
          X(Error(`Failed to cleanup: ${Q.error?.message}`));
        }),
          (Q.onsuccess = () => {
            let L = Q.result;
            if (L) (L.delete(), _++, L.continue());
            else {
              if (
                _ > 0 &&
                typeof process < "u" &&
                process.env?.NODE_ENV !== "production"
              )
                console.log(
                  `[GoSPA IndexedDB] Cleaned up ${_} expired entries`,
                );
              J(_);
            }
          }));
      })
    );
  }
  async getSize() {
    return (
      await this.init(),
      new Promise((J, X) => {
        if (!this.db) {
          X(Error("Database not initialized"));
          return;
        }
        let Z = this.db
            .transaction(this.config.storeName, "readonly")
            .objectStore(this.config.storeName),
          $ = Z.count(),
          G = 0;
        (($.onerror = () => {
          X(Error(`Failed to count entries: ${$.error?.message}`));
        }),
          ($.onsuccess = () => {
            G = $.result;
            let _ = Z.getAll();
            ((_.onerror = () => {
              J({ entries: G, bytes: 0 });
            }),
              (_.onsuccess = () => {
                let Q = _.result,
                  L = new Blob([JSON.stringify(Q)]).size;
                J({ entries: G, bytes: L });
              }));
          }));
      })
    );
  }
  close() {
    if (this.db) {
      if (
        (this.db.close(),
        (this.db = null),
        (this.initPromise = null),
        typeof process < "u" && process.env?.NODE_ENV !== "production")
      )
        console.log(`[GoSPA IndexedDB] Database closed: ${this.config.dbName}`);
    }
  }
  async deleteDatabase() {
    return (
      this.close(),
      new Promise((J, X) => {
        let Y = indexedDB.deleteDatabase(this.config.dbName);
        ((Y.onerror = () => {
          X(Error(`Failed to delete database: ${Y.error?.message}`));
        }),
          (Y.onsuccess = () => {
            if (typeof process < "u" && process.env?.NODE_ENV !== "production")
              console.log(
                `[GoSPA IndexedDB] Database deleted: ${this.config.dbName}`,
              );
            J();
          }));
      })
    );
  }
}
function r(J) {
  return new B(J);
}
var F = null;
function s(J) {
  if (!F) F = new B(J);
  return F;
}
function a() {
  if (F) (F.close(), (F = null));
}
class x {
  container = null;
  config;
  announceTimer = null;
  pendingAnnouncements = [];
  constructor(J = {}) {
    if (
      ((this.config = {
        announceNavigation: J.announceNavigation ?? !0,
        announceStateChanges: J.announceStateChanges ?? !1,
        politeness: J.politeness ?? "polite",
      }),
      typeof document < "u")
    )
      this.init();
  }
  init() {
    if (
      ((this.container = document.getElementById("gospa-announcer")),
      !this.container)
    )
      ((this.container = document.createElement("div")),
        (this.container.id = "gospa-announcer"),
        this.container.setAttribute("aria-live", this.config.politeness),
        this.container.setAttribute("aria-atomic", "true"),
        this.container.setAttribute("role", "status"),
        (this.container.style.cssText = `
				position: absolute;
				width: 1px;
				height: 1px;
				padding: 0;
				margin: -1px;
				overflow: hidden;
				clip: rect(0, 0, 0, 0);
				white-space: nowrap;
				border: 0;
			`),
        document.body.appendChild(this.container));
  }
  announce(J, X) {
    if (!this.container) this.init();
    if (X && X !== this.config.politeness)
      this.container?.setAttribute("aria-live", X);
    if (this.announceTimer) clearTimeout(this.announceTimer);
    (this.pendingAnnouncements.push(J),
      (this.announceTimer = setTimeout(() => {
        let Y = this.pendingAnnouncements.join(". ");
        if (((this.pendingAnnouncements = []), this.container))
          ((this.container.textContent = ""),
            requestAnimationFrame(() => {
              if (this.container) this.container.textContent = Y;
            }));
        if (X && X !== this.config.politeness)
          this.container?.setAttribute("aria-live", this.config.politeness);
      }, 100)));
  }
  announceNavigation(J, X) {
    if (!this.config.announceNavigation) return;
    let Y = X ? `Navigated to ${X}` : `Navigated to ${J}`;
    this.announce(Y);
  }
  announceStateChange(J, X) {
    if (!this.config.announceStateChanges) return;
    let Y = typeof X === "object" ? JSON.stringify(X) : String(X);
    this.announce(`${J} changed to ${Y}`);
  }
  announceLoading(J = "Loading") {
    this.announce(J, "assertive");
  }
  announceError(J) {
    this.announce(`Error: ${J}`, "assertive");
  }
  announceSuccess(J) {
    this.announce(J);
  }
  destroy() {
    if (this.announceTimer) clearTimeout(this.announceTimer);
    if (this.container) (this.container.remove(), (this.container = null));
    this.pendingAnnouncements = [];
  }
}
var t = {
    setAttributes(J, X) {
      for (let [Y, Z] of Object.entries(X))
        if (Z === null || Z === !1) J.removeAttribute(Y);
        else if (Z === !0) J.setAttribute(Y, "");
        else J.setAttribute(Y, String(Z));
    },
    makeFocusable(J, X = 0) {
      J.setAttribute("tabindex", String(X));
    },
    label(J, X) {
      J.setAttribute("aria-label", X);
    },
    describe(J, X) {
      J.setAttribute("aria-describedby", X);
    },
    expanded(J, X) {
      J.setAttribute("aria-expanded", String(X));
    },
    hidden(J, X) {
      if (X) J.setAttribute("aria-hidden", "true");
      else J.removeAttribute("aria-hidden");
    },
    selected(J, X) {
      J.setAttribute("aria-selected", String(X));
    },
    checked(J, X) {
      J.setAttribute("aria-checked", String(X));
    },
    disabled(J, X) {
      J.setAttribute("aria-disabled", String(X));
    },
    busy(J, X) {
      J.setAttribute("aria-busy", String(X));
    },
    live(J, X) {
      J.setAttribute("aria-live", X);
    },
    createDescription(J, X) {
      let Y = document.createElement("div");
      return (
        (Y.id = J),
        (Y.className = "gospa-sr-only"),
        (Y.textContent = X),
        (Y.style.cssText = `
			position: absolute;
			width: 1px;
			height: 1px;
			padding: 0;
			margin: -1px;
			overflow: hidden;
			clip: rect(0, 0, 0, 0);
			white-space: nowrap;
			border: 0;
		`),
        Y
      );
    },
  },
  n = {
    trap(J) {
      let X = [
          "a[href]",
          "button:not([disabled])",
          "input:not([disabled])",
          "textarea:not([disabled])",
          "select:not([disabled])",
          '[tabindex]:not([tabindex="-1"])',
        ].join(", "),
        Y = Array.from(J.querySelectorAll(X));
      if (Y.length === 0) return () => {};
      let Z = Y[0],
        $ = Y[Y.length - 1],
        G = (_) => {
          let Q = _;
          if (Q.key !== "Tab") return;
          if (Q.shiftKey) {
            if (document.activeElement === Z) (Q.preventDefault(), $.focus());
          } else if (document.activeElement === $)
            (Q.preventDefault(), Z.focus());
        };
      return (
        J.addEventListener("keydown", G),
        Z.focus(),
        () => {
          J.removeEventListener("keydown", G);
        }
      );
    },
    restore(J) {
      if (J && J instanceof HTMLElement) J.focus();
    },
    save() {
      let J = document.activeElement;
      return () => this.restore(J);
    },
    moveTo(J) {
      if (J instanceof HTMLElement) J.focus();
    },
  };
function e(J) {
  return new x(J);
}
var W = null;
function v(J) {
  if (!W) W = new x(J);
  return W;
}
function JJ() {
  if (W) (W.destroy(), (W = null));
}
function XJ(J, X) {
  v().announce(J, X);
}
class A {
  metrics = [];
  marks = new Map();
  config;
  observers = new Set();
  constructor(J = {}) {
    this.config = {
      enabled:
        J.enabled ??
        (typeof process < "u" && process.env?.NODE_ENV !== "production"),
      maxMetrics: J.maxMetrics ?? 1000,
      sampleRate: J.sampleRate ?? 1,
      enableConsoleLog: J.enableConsoleLog ?? !1,
    };
  }
  isEnabled() {
    if (!this.config.enabled) return !1;
    if (this.config.sampleRate < 1 && Math.random() > this.config.sampleRate)
      return !1;
    return !0;
  }
  start(J) {
    if (!this.isEnabled()) return;
    let X = `gospa:${J}:start`;
    if (
      (this.marks.set(J, performance.now()),
      typeof performance < "u" && performance.mark)
    )
      performance.mark(X);
  }
  end(J, X) {
    if (!this.isEnabled()) return null;
    let Y = this.marks.get(J);
    if (Y === void 0)
      return (
        console.warn(`[GoSPA Performance] No start mark found for: ${J}`),
        null
      );
    let $ = performance.now() - Y;
    this.marks.delete(J);
    let G = { name: J, duration: $, timestamp: Date.now(), metadata: X };
    if ((this.addMetric(G), typeof performance < "u" && performance.measure))
      try {
        let _ = `gospa:${J}:start`,
          Q = `gospa:${J}:end`;
        (performance.mark(Q),
          performance.measure(`gospa:${J}`, _, Q),
          performance.clearMarks(_),
          performance.clearMarks(Q));
      } catch {}
    return $;
  }
  measure(J, X, Y) {
    if (!this.isEnabled()) return X();
    this.start(J);
    try {
      let Z = X();
      return (this.end(J, Y), Z);
    } catch (Z) {
      throw (this.end(J, { ...Y, error: !0 }), Z);
    }
  }
  async measureAsync(J, X, Y) {
    if (!this.isEnabled()) return X();
    this.start(J);
    try {
      let Z = await X();
      return (this.end(J, Y), Z);
    } catch (Z) {
      throw (this.end(J, { ...Y, error: !0 }), Z);
    }
  }
  addMetric(J) {
    if ((this.metrics.push(J), this.metrics.length > this.config.maxMetrics))
      this.metrics = this.metrics.slice(-this.config.maxMetrics);
    for (let X of this.observers)
      try {
        X(J);
      } catch (Y) {
        console.error("[GoSPA Performance] Observer error:", Y);
      }
    if (this.config.enableConsoleLog)
      console.log(
        `[GoSPA Performance] ${J.name}: ${J.duration.toFixed(2)}ms`,
        J.metadata,
      );
  }
  getMetrics() {
    return [...this.metrics];
  }
  getMetricsByName(J) {
    return this.metrics.filter((X) => X.name === J);
  }
  getAverageDuration(J) {
    let X = this.getMetricsByName(J);
    if (X.length === 0) return 0;
    return X.reduce((Z, $) => Z + $.duration, 0) / X.length;
  }
  getSummary() {
    let J = {};
    for (let X of this.metrics) {
      if (!J[X.name]) J[X.name] = { count: 0, avg: 0, min: 1 / 0, max: -1 / 0 };
      let Y = J[X.name];
      (Y.count++,
        (Y.min = Math.min(Y.min, X.duration)),
        (Y.max = Math.max(Y.max, X.duration)));
    }
    for (let X of Object.keys(J)) {
      let Y = this.getMetricsByName(X),
        Z = Y.reduce(($, G) => $ + G.duration, 0);
      J[X].avg = Z / Y.length;
    }
    return J;
  }
  subscribe(J) {
    return (this.observers.add(J), () => this.observers.delete(J));
  }
  clear() {
    ((this.metrics = []), this.marks.clear());
  }
  getMemoryUsage() {
    if (typeof performance < "u" && "memory" in performance) {
      let J = performance.memory;
      return { used: J.usedJSHeapSize, total: J.totalJSHeapSize };
    }
    return null;
  }
  async getWebVitals() {
    let J = {};
    if (typeof performance < "u" && performance.getEntriesByType) {
      let X = performance.getEntriesByType("paint");
      for (let _ of X)
        if (_.name === "first-contentful-paint") J.FCP = _.startTime;
      let Y = performance.getEntriesByType("largest-contentful-paint");
      if (Y.length > 0) J.LCP = Y[Y.length - 1].startTime;
      let Z = performance.getEntriesByType("first-input");
      if (Z.length > 0) {
        let _ = Z[0];
        J.FID = _.processingStart - _.startTime;
      }
      let $ = performance.getEntriesByType("layout-shift"),
        G = 0;
      for (let _ of $) if (!_.hadRecentInput) G += _.value;
      J.CLS = G;
    }
    return J;
  }
}
function YJ(J) {
  return new A(J);
}
var w = null;
function E(J) {
  if (!w) w = new A(J);
  return w;
}
function ZJ() {
  if (w) (w.clear(), (w = null));
}
function $J(J, X, Y) {
  return E().measure(J, X, Y);
}
function GJ(J, X, Y) {
  return E().measureAsync(J, X, Y);
}
export {
  u as withErrorBoundary,
  OX as watchProp,
  wJ as watch,
  LJ as updateDevToolsPanel,
  CJ as untrack,
  TJ as unregisterBinding,
  vJ as transformers,
  QJ as toggleDevTools,
  zX as toRaw,
  VJ as timing,
  pJ as throttle,
  mJ as syncedRune,
  NX as slide,
  VX as signalEffect,
  BX as setupTransitions,
  SX as setState,
  xJ as setSanitizer,
  lJ as setNavigationOptions,
  yJ as sendAction,
  CX as scale,
  C as rune,
  f as resourceReactive,
  jJ as renderList,
  EJ as renderIf,
  _X as remoteAction,
  GX as remote,
  AJ as registerBinding,
  FX as reactiveArray,
  LX as reactive,
  XX as prefetch,
  DJ as preEffect,
  kJ as onKey,
  y as onComponentError,
  dJ as onBeforeNavigate,
  oJ as onAfterNavigate,
  qJ as on,
  SJ as offAll,
  iJ as navigate,
  KJ as memoryUsage,
  GJ as measureAsync,
  $J as measure,
  bJ as keys,
  HX as isReactive,
  nJ as isNavigating,
  d as isInErrorState,
  UJ as inspect,
  gJ as initWebSocket,
  p as initStreaming,
  k as initPriorityHydration,
  eJ as initNavigation,
  xX as initIslands,
  PX as init,
  TX as hydrateIsland,
  aJ as go,
  uJ as getWebSocketClient,
  hX as getWebSocket,
  bX as getTransitions,
  l as getTabSync,
  b as getStreamingManager,
  qX as getState,
  $X as getRemotePrefix,
  q as getPriorityScheduler,
  E as getPerformanceMonitor,
  kX as getNavigation,
  AX as getIslandManager,
  s as getIndexedDBPersistence,
  m as getErrorBoundaryState,
  tJ as getCurrentPath,
  jX as getComponent,
  v as getAnnouncer,
  sJ as forward,
  n as focus,
  wX as fly,
  BJ as flushDOMUpdatesNow,
  WX as fade,
  WJ as effect,
  i as destroyTabSync,
  ZJ as destroyPerformanceMonitor,
  JX as destroyNavigation,
  a as destroyIndexedDBPersistence,
  EX as destroyComponent,
  JJ as destroyAnnouncer,
  HJ as derived,
  hJ as delegate,
  IJ as debounce,
  RX as crossfade,
  o as createTabSync,
  YJ as createPerformanceMonitor,
  YX as createNavigationState,
  r as createIndexedDBPersistence,
  g as createErrorFallback,
  _J as createDevToolsPanel,
  MX as createComponent,
  e as createAnnouncer,
  ZX as configureRemote,
  c as clearAllErrorBoundaries,
  RJ as cancelPendingDOMUpdates,
  IX as callAction,
  DX as blur,
  MJ as bindTwoWay,
  PJ as bindElement,
  pX as bind,
  D as batch,
  rJ as back,
  vX as autoInit,
  t as aria,
  cJ as applyStateUpdate,
  XJ as announce,
  R as WSTabSync,
  fJ as WSClient,
  NJ as StateMap,
  x as ScreenReaderAnnouncer,
  OJ as Rune,
  P as Resource,
  A as PerformanceMonitor,
  B as IndexedDBPersistence,
  FJ as Effect,
  zJ as Derived,
  QX as $state,
  KX as $effect,
  UX as $derived,
};
export {
  q as a,
  k as b,
  p as c,
  b as d,
  P as e,
  f,
  y as g,
  u as h,
  g as i,
  m as j,
  c as k,
  d as l,
  R as m,
  o as n,
  l as o,
  i as p,
  B as q,
  r,
  s,
  a as t,
  x as u,
  t as v,
  n as w,
  e as x,
  v as y,
  JJ as z,
  XJ as A,
  A as B,
  YJ as C,
  E as D,
  ZJ as E,
  $J as F,
  GJ as G,
};
