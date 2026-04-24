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
  type Unsubscribe,
};

import {
  bindElement,
  bindTwoWay,
  renderIf,
  renderList,
  registerBinding,
  unregisterBinding,
} from "./dom.ts";
import { trustedHTML, type TrustedHTMLValue } from "./html-policy.ts";

export {
  bindElement,
  bindTwoWay,
  renderIf,
  renderList,
  registerBinding,
  unregisterBinding,
  trustedHTML,
  type TrustedHTMLValue,
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

import { $state, $derived, $effect } from "./signals.ts";
import { createStore, getStore } from "./store.ts";

export { $state, $derived, $effect, createStore, getStore };

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
    mode: "immediate" | "idle" | "visible" | "manual" | "progressive" | "lazy";
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
  /** Unified transport fallback settings (WebSocket -> SSE -> polling). */
  transport?: {
    enabled?: boolean;
    sseUrl?: string;
    pollUrl?: string;
    pollInterval?: number;
  };
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
export const components = new Map<string, ComponentInstance>();
export const globalState = new StateMap();

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
export let config: RuntimeConfig = {};

function shouldAutoInitDocument(): boolean {
  if (typeof document === "undefined") return false;
  return document.querySelector("[data-gospa-root], [data-gospa-component], [data-gospa-island]") !== null;
}

// Lazy-loaded aggregate features bundle
let featuresModule: Promise<typeof import("./framework-features.ts")> | null =
  null;
let cachedFeatures: typeof import("./framework-features.ts") | null = null;

/**
 * Declare global debug constant for build-time stripping.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

/**
 * Initialize the GoSPA runtime.
 * Should be called once at application startup.
 */
export function init(userConfig: Partial<RuntimeConfig> = {}): void {
  if (isInitialized) {
    if (Object.keys(userConfig).length > 0) {
      config = { ...config, ...userConfig };
    }
    return;
  }
  isInitialized = true;
  config = { ...config, ...userConfig };

  if (config.wsUrl && (config.transport?.enabled ?? true)) {
    void getFrameworkFeatures()
      .then((mod) => {
        if (typeof mod.initTransport !== "function") return;
        mod.initTransport({
          wsUrl: config.wsUrl,
          sseUrl: config.transport?.sseUrl,
          pollUrl: config.transport?.pollUrl,
          pollInterval: config.transport?.pollInterval,
          wsReconnectDelay: config.wsReconnectDelay,
          wsMaxReconnect: config.wsMaxReconnect,
          wsHeartbeat: config.wsHeartbeat,
          serializationFormat: config.serializationFormat,
          debug: Boolean(config.debug),
        });
      })
      .catch(() => {
        // Keep runtime boot resilient if transport module fails to load.
      });
  }

  // Auto-initialize when GoSPA markers are present.
  if (shouldAutoInitDocument()) {
    autoInit();
  }
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

/**
 * Scan DOM for GoSPA components and islands, initialize them.
 */
export function autoInit(): void {
  const componentRoots = document.querySelectorAll("[data-gospa-component]");
  componentRoots.forEach((root) => {
    const el = root as HTMLElement;
    const name = el.getAttribute("data-gospa-component")!;
    const id = el.id || `c-${Math.random().toString(36).substring(2, 9)}`;
    if (!el.id) el.id = id;

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
    const el = root as HTMLElement;
    const name = el.getAttribute("data-gospa-island");
    if (!name) return;

    let setup = setupFunctions.get(name);
    if (!setup) {
      const globalSetups = (window as any).__GOSPA_SETUPS__;
      if (globalSetups && typeof globalSetups[name] === "function") {
        setup = globalSetups[name];
      }
    }

    if (setup) {
      try {
        let stateData: Record<string, any> = {};
        const stateAttr = el.getAttribute("data-gospa-state");
        if (stateAttr) {
          try {
            stateData = JSON.parse(stateAttr);
          } catch {
            /* ignore */
          }
        }

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
    }
  });
}

// Lazy module loaders using the aggregate bundle
/**
 * Get the framework features module synchronously if already loaded.
 */
export function getFrameworkFeaturesSync() {
  return cachedFeatures;
}

/**
 * Get the framework features module, loading it if necessary.
 */
export async function getFrameworkFeatures() {
  if (cachedFeatures) return cachedFeatures;
  if (!featuresModule) {
    featuresModule = import("./framework-features.ts").then((mod) => {
      cachedFeatures = mod;
      return mod;
    });
  }
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
      if (shouldAutoInitDocument()) autoInit();
    });
  } else if (shouldAutoInitDocument()) {
    autoInit();
  }
}

// Registry for global GoSPA object
const GoSPA = {
  config,
  components,
  globalState,
  init,
  createComponent,
  destroyComponent,
  getComponent,
  getState,
  setState,
  bind,
  autoInit,
  createIsland,
  getFrameworkFeatures,
  getWebSocket,
  getNavigation,
  getTransitions,
};

if (typeof window !== "undefined") {
  (window as any).GoSPA = GoSPA;
  (window as any).__GOSPA__ = GoSPA;
}

export default GoSPA;
