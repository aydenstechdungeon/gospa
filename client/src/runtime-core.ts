// GoSPA Core Runtime - Minimal core (~15KB target)
// Only includes essential state, DOM bindings, and events

import {
  Rune,
  Derived,
  Effect,
  StateMap,
  batch,
  effect,
  rune,
  derived,
  watch,
  untrack,
  preEffect,
  type Unsubscribe,
} from "./state.ts";
export {
  Rune,
  Derived,
  Effect,
  StateMap,
  batch,
  effect,
  rune,
  derived,
  watch,
  untrack,
  preEffect,
  type Unsubscribe,
};
import {
  bindElement,
  bindTwoWay,
  renderIf,
  renderList,
  registerBinding,
  unregisterBinding,
  sanitizeHtml,
} from "./dom.ts";
export {
  bindElement,
  bindTwoWay,
  renderIf,
  renderList,
  registerBinding,
  unregisterBinding,
  sanitizeHtml,
};
import {
  on,
  offAll,
  debounce,
  throttle,
  delegate,
  onKey,
  keys,
  transformers,
} from "./events.ts";
export { on, offAll, debounce, throttle, delegate, onKey, keys, transformers };
import {
  initWebSocket,
  getWebSocketClient,
  sendAction,
  type StateMessage,
} from "./websocket.ts";
export { initWebSocket, getWebSocketClient, sendAction, type StateMessage };
import { navigate, back, prefetch } from "./navigation.ts";
export { navigate, back, prefetch };
import {
  remote,
  remoteAction,
  configureRemote,
  getRemotePrefix,
  type RemoteOptions,
  type RemoteResult,
} from "./remote.ts";
export {
  remote,
  remoteAction,
  configureRemote,
  getRemotePrefix,
  type RemoteOptions,
  type RemoteResult,
};
import { $state, $derived, $effect } from "./signals.ts";
import { createStore, getStore } from "./store.ts";

// Re-export types

/** Core runtime configuration */
export interface RuntimeConfig {
  /** WebSocket URL for real-time updates */
  wsUrl?: string;
  /** Whether to enable debug logging */
  debug?: boolean;
  /** Custom WebSocket reconnection delay (ms) */
  wsReconnectDelay?: number;
  /** Maximum WebSocket reconnection attempts */
  wsMaxReconnect?: number;
  /** WebSocket heartbeat interval (ms) */
  wsHeartbeat?: number;
  /** Hydration mode ('immediate', 'idle', 'visible') */
  hydration?: {
    mode: "immediate" | "idle" | "visible";
    timeout?: number;
  };
  /** Callback for WebSocket connection errors */
  onConnectionError?: (err: Error) => void;
  /** Whether to use simple runtime SVGs for hydration */
  simpleRuntimeSVGs?: boolean;
  /** Whether to disable HTML sanitization (not recommended) */
  disableSanitization?: boolean;
  /** WebSocket serialization format */
  serializationFormat?: "json" | "msgpack";
}

/** Component definition from server */
export interface ComponentDefinition {
  name: string;
  initialState?: Record<string, any>;
  onMount?: (instance: ComponentInstance) => void;
  onDestroy?: (instance: ComponentInstance) => void;
}

/** Active component instance */
export interface ComponentInstance {
  id: string;
  name: string;
  states: StateMap;
  elements: Set<HTMLElement>;
  dispose: () => void;
}

// Global component registry
const components = new Map<string, ComponentInstance>();
const globalState = new StateMap();

// Island setup function registry - populated by bundled island modules
type IslandSetupFn = (
  element: Element,
  props: Record<string, any>,
  state: any,
) => void;
const setupFunctions = new Map<string, IslandSetupFn>();

/**
 * Register a setup function for an island component.
 * Called by bundled island modules at load time.
 */
export function registerSetup(name: string, setup: IslandSetupFn): void {
  setupFunctions.set(name, setup);
}

/**
 * Get a registered setup function for an island.
 */
export function getSetup(name: string): IslandSetupFn | undefined {
  const local = setupFunctions.get(name);
  if (local) return local;
  const globalSetups = (window as any).__GOSPA_SETUPS__;
  if (globalSetups && typeof globalSetups[name] === "function") {
    return globalSetups[name];
  }
  return undefined;
}

// Runtime state
let isInitialized = false;
let config: RuntimeConfig = {};

// Lazy-loaded aggregate features bundle
let featuresModule: Promise<typeof import("./framework-features.ts")> | null =
  null;

/**
 * Initialize the GoSPA runtime.
 * Should be called once at application startup.
 */
export function init(userConfig: Partial<RuntimeConfig> = {}): void {
  // Prevent multiple initializations (safe to call multiple times if no config change)
  if (isInitialized) {
    if (Object.keys(userConfig).length > 0) {
      config = { ...config, ...userConfig };
    }
    return;
  }
  isInitialized = true;
  config = { ...config, ...userConfig };

  // Initialize WebSocket if URL provided (lazy load via aggregate)
  if (config.wsUrl) {
    featuresModule = import("./framework-features.ts").then((mod) => {
      const ws = mod.initWebSocket({
        url: config.wsUrl!,
        onMessage: handleServerMessage,
        serializationFormat: config.serializationFormat,
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

  // Set up global error handler
  if (typeof window !== "undefined") {
    window.addEventListener("error", (event) => {
      if (config.debug) console.error("Runtime error:", event.error);
    });
  }
}

// The public GoSPA global object
const GoSPA = {
  // Configuration
  get config() {
    return config;
  },
  components,
  globalState,
  // Core API
  init,
  createComponent,
  destroyComponent,
  getComponent,
  getState,
  setState,
  callAction,
  bind,
  autoInit,
  // Remote actions
  remote,
  remoteAction,
  configureRemote,
  getRemotePrefix,
  // State primitives
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
  // Utility functions
  batch,
  effect,
  watch,
  // Events
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
  // Unified Reactive Store API
  $state,
  $derived,
  $effect,
  createStore,
  getStore,
  createIsland,
  // Realtime & Navigation (Full API)
  initWebSocket,
  getWebSocketClient,
  sendAction,
  navigate,
  back,
  prefetch,
};

// Expose to window immediately
if (typeof window !== "undefined") {
  (window as any).GoSPA = GoSPA;
  (window as any).__GOSPA__ = GoSPA;
}

/**
 * Create a new component instance.
 */
export function createComponent(id: string, name: string): ComponentInstance {
  if (components.has(id)) return components.get(id)!;

  const instance: ComponentInstance = {
    id,
    name,
    states: new StateMap(),
    elements: new Set(),
    dispose: () => {
      instance.states.dispose();
      instance.elements.clear();
      components.delete(id);
    },
  };

  components.set(id, instance);
  return instance;
}

/**
 * Destroy a component instance.
 */
export function destroyComponent(id: string): void {
  const component = components.get(id);
  if (component) component.dispose();
}

/**
 * Get a component instance by ID.
 */
export function getComponent(id: string): ComponentInstance | undefined {
  return components.get(id);
}

/**
 * Get state value for a component.
 */
export function getState<T>(componentId: string, key: string): T | undefined {
  const component = components.get(componentId);
  if (!component) return undefined;
  const rune = component.states.get<T>(key);
  return rune ? rune.get() : undefined;
}

/**
 * Set state value for a component.
 */
export function setState<T>(componentId: string, key: string, value: T): void {
  const component = components.get(componentId);
  if (component) {
    component.states.set(key, value);
  }
}

/**
 * Call a remote action (alias for remote).
 */
export function callAction<T = any, R = any>(
  name: string,
  input?: T,
): Promise<RemoteResult<R>> {
  return remote(name, input);
}

/**
 * Bind an element to a reactive state.
 */
export function bind(
  componentId: string,
  element: HTMLElement,
  property: string,
  key: string,
  options: { twoWay?: boolean; transformer?: (v: any) => any } = {},
): Unsubscribe {
  const component = components.get(componentId);
  if (!component) return () => {};

  component.elements.add(element);

  let rune = component.states.get(key);
  if (!rune) {
    // Check if initial state exists in DOM (data-gospa-state)
    const container = element.closest("[data-gospa-state]");
    if (container) {
      try {
        const initialState = JSON.parse(
          container.getAttribute("data-gospa-state") || "{}",
        );
        if (initialState[key] !== undefined) {
          rune = component.states.set(key, initialState[key]);
        }
      } catch (e) {
        /* ignore */
      }
    }
    if (!rune) rune = component.states.set(key, undefined as any);
  }

  if (options.twoWay) {
    return bindTwoWay(element as any, rune! as any);
  }
  return bindElement(element, rune!, {
    type: property as any,
    transform: options.transformer,
  });
}

/**
 * Create a reactive island for SFC components.
 */
export function createIsland(id: string, name: string): ComponentInstance {
  const instance = createComponent(id, name);
  // Auto-bind elements with data-gospa-bind or data-model
  const root = document.querySelector(
    `[data-gospa-component="${name}"][id="${id}"]`,
  ) as HTMLElement;
  if (root) {
    autoBindIsland(id, root);
  }
  return instance;
}

function autoBindIsland(componentId: string, root: HTMLElement): void {
  const elements = root.querySelectorAll("[data-gospa-bind], [data-model]");
  for (const el of elements) {
    const element = el as HTMLElement;
    const bindAttr = element.getAttribute("data-gospa-bind");
    if (bindAttr) {
      const [prop, key] = bindAttr.split(":");
      bind(componentId, element, prop, key);
      continue;
    }

    const key = element.getAttribute("data-model");
    if (key) bind(componentId, element, "value", key, { twoWay: true });
  }
}

// Handle messages from server
function handleServerMessage(message: StateMessage): void {
  switch (message.type) {
    case "init":
      if (message.componentId && message.data) {
        const component = components.get(message.componentId);
        if (component) component.states.fromJSON(message.data);
      } else if (message.state) {
        const stateObj = message.state as Record<string, unknown>;
        for (const [scopedKey, value] of Object.entries(stateObj)) {
          const dotIndex = scopedKey.indexOf(".");
          if (dotIndex > 0) {
            const componentId = scopedKey.substring(0, dotIndex);
            const stateKey = scopedKey.substring(dotIndex + 1);
            const component = components.get(componentId);
            if (component) component.states.set(stateKey, value);
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
        globalState.fromJSON(message.patch as Record<string, unknown>);
      }
      break;
    case "update":
      if (message.componentId && message.diff) {
        const component = components.get(message.componentId);
        if (component) component.states.fromJSON(message.diff);
      }
      break;
    case "sync":
      if (message.data) {
        globalState.fromJSON(message.data);
      } else if (message.key !== undefined && message.value !== undefined) {
        const scopedKey = message.key as string;
        const componentId = message.componentId as string;
        if (componentId) {
          const component = components.get(componentId);
          if (component) component.states.set(scopedKey, message.value);
        } else {
          globalState.set(scopedKey, message.value);
        }
      }
      break;
    case "error":
      if (config.debug) console.error("Server error:", message.error);
      break;
  }
}

/**
 * Scan DOM for GoSPA components and islands, initialize them.
 */
export function autoInit(): void {
  // Initialize components (data-gospa-component)
  const componentRoots = document.querySelectorAll("[data-gospa-component]");
  componentRoots.forEach((root) => {
    const el = root as HTMLElement;
    const name = el.getAttribute("data-gospa-component")!;
    const id = el.id || `c-${Math.random().toString(36).substring(2, 9)}`;
    if (!el.id) el.id = id;

    const instance = createComponent(id, name);

    // Initial state from data-gospa-state
    const stateData = el.getAttribute("data-gospa-state");
    if (stateData) {
      try {
        instance.states.fromJSON(JSON.parse(stateData));
      } catch (e) {
        if (config.debug)
          console.error("Error parsing initial state for", name, e);
      }
    }

    // Bind elements
    autoBindIsland(id, el);
  });

  // Initialize islands (data-gospa-island) using registered setup functions
  const islandRoots = document.querySelectorAll("[data-gospa-island]");
  islandRoots.forEach((root) => {
    const el = root as HTMLElement;
    const name = el.getAttribute("data-gospa-island");
    if (!name) return;

    // Check module-scoped registry first, then global registry
    let setup = setupFunctions.get(name);
    if (!setup) {
      const globalSetups = (window as any).__GOSPA_SETUPS__;
      if (globalSetups && typeof globalSetups[name] === "function") {
        setup = globalSetups[name];
      }
    }

    if (setup) {
      try {
        // Parse initial state from data-gospa-state
        let stateData: Record<string, any> = {};
        const stateAttr = el.getAttribute("data-gospa-state");
        if (stateAttr) {
          try {
            stateData = JSON.parse(stateAttr);
          } catch {
            /* ignore */
          }
        }

        // Parse initial props from data-gospa-props
        let propsData: Record<string, any> = {};
        const propsAttr = el.getAttribute("data-gospa-props");
        if (propsAttr) {
          try {
            propsData = JSON.parse(propsAttr);
          } catch {
            /* ignore */
          }
        }

        setup(el, propsData, stateData);
      } catch (e) {
        if (config.debug) console.error("Error initializing island", name, e);
      }
    } else if (config.debug) {
      console.warn("No setup function registered for island:", name);
    }
  });
}

// Lazy module loaders using the aggregate bundle
export async function getFrameworkFeatures() {
  if (!featuresModule) featuresModule = import("./framework-features.ts");
  return featuresModule;
}

export async function getWebSocket() {
  return getFrameworkFeatures();
}

export async function getNavigation() {
  return getFrameworkFeatures();
}

export async function getTransitions() {
  return getFrameworkFeatures();
}

// Auto-initialize on DOM ready
if (typeof document !== "undefined") {
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => {
      if (document.documentElement.hasAttribute("data-gospa-auto")) autoInit();
    });
  } else if (document.documentElement.hasAttribute("data-gospa-auto")) {
    autoInit();
  }
}

// FIX: Register navigation callbacks to clean up stale state on page navigation
function registerNavigationCleanup(): void {
  if (typeof window === "undefined") return;

  // Lazy load framework features for navigation cleanup
  getFrameworkFeatures()
    .then((mod) => {
      mod.onBeforeNavigate(() => {
        // Cleanup component instances
        for (const [id] of components) {
          destroyComponent(id);
        }
        globalState.clear();

        // Cleanup island manager resources
        mod.getIslandManager()?.destroy();
      });

      // Re-discover islands after navigation completed
      document.addEventListener("gospa:navigated", () => {
        mod.getIslandManager()?.discoverIslands();
      });
    })
    .catch(() => {
      /* skip */
    });
}

if (typeof window !== "undefined") {
  registerNavigationCleanup();
}

export default GoSPA;
