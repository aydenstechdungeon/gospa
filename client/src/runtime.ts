// GoSPA Client Runtime - Main entry point (Full Version)
// Includes all features: navigation, websockets, remote actions, etc.

import {
  init as coreInit,
  createComponent,
  destroyComponent,
  getComponent,
  getState,
  setState,
  bind,
  autoInit,
  getWebSocket,
  getNavigation,
  getTransitions,
  getFrameworkFeatures,
  getFrameworkFeaturesSync,
} from "./runtime-core.ts";

export {
  createComponent,
  destroyComponent,
  getComponent,
  getState,
  setState,
  bind,
  autoInit,
  getWebSocket,
  getNavigation,
  getTransitions,
};

import { initNavigation } from "./navigation.ts";

/**
 * Initialize the full GoSPA runtime with navigation and all features.
 */
export async function init(config: any = {}) {
  coreInit(config);
  initNavigation();
}

// State Primitives
export {
  Rune,
  Derived,
  Effect,
  StateMap,
  batch,
  rune,
  derived,
  watch,
  untrack,
  bindElement,
  bindTwoWay,
  renderIf,
  renderList,
  trustedHTML,
  type TrustedHTMLValue,
} from "./runtime-core.ts";

// Remote Actions (Imported directly for Full bundle)
import {
  remote,
  remoteAction,
  configureRemote,
  getRemotePrefix,
  type RemoteOptions,
  type RemoteResult,
} from "./remote.ts";

export { remote, remoteAction, configureRemote, getRemotePrefix };
export type { RemoteOptions, RemoteResult };

// Navigation
import type { NavigateOptions, NavigationOptions } from "./navigation.ts";
export type { NavigateOptions, NavigationOptions };

// WebSocket
import type { StateMessage } from "./websocket.ts";
export type { StateMessage };

// Types
export type {
  ComponentDefinition,
  ComponentInstance,
  RuntimeConfig,
} from "./runtime-core.ts";
export type { Unsubscribe } from "./state.ts";

// WebSocket Full API
export async function initWebSocket(config: any) {
  const syncMod = getFrameworkFeaturesSync();
  if (syncMod) return syncMod.initWebSocket(config);
  const mod = await getFrameworkFeatures();
  return mod.initWebSocket(config);
}

export async function getWebSocketClient() {
  const syncMod = getFrameworkFeaturesSync();
  if (syncMod) return syncMod.getWebSocketClient();
  const mod = await getFrameworkFeatures();
  return mod.getWebSocketClient();
}

export async function sendAction(name: string, payload?: any) {
  const syncMod = getFrameworkFeaturesSync();
  if (syncMod) return syncMod.sendAction(name, payload);
  const mod = await getFrameworkFeatures();
  return mod.sendAction(name, payload);
}

// Navigation Full API
export async function navigate(to: string, options?: any) {
  const syncMod = getFrameworkFeaturesSync();
  if (syncMod) return syncMod.navigate(to, options);
  const mod = await getFrameworkFeatures();
  return mod.navigate(to, options);
}

export async function back() {
  const syncMod = getFrameworkFeaturesSync();
  if (syncMod) return syncMod.back();
  const mod = await getFrameworkFeatures();
  return mod.back();
}

export async function prefetch(path: string) {
  const syncMod = getFrameworkFeaturesSync();
  if (syncMod) return syncMod.prefetch(path);
  const mod = await getFrameworkFeatures();
  return mod.prefetch(path);
}

// Islands & Priority
export async function initIslands(config?: any) {
  const mod = await getFrameworkFeatures();
  return mod.initIslands(config);
}


export async function getIslandManager() {
  const mod = await getFrameworkFeatures();
  return mod.getIslandManager();
}

export async function hydrateIsland(idOrName: string) {
  const mod = await getFrameworkFeatures();
  return mod.hydrateIsland(idOrName);
}

// Streaming
export async function initStreaming(config?: any) {
  const mod = await getFrameworkFeatures();
  return mod.initStreaming(config);
}

// Transitions
export async function setupTransitions(root?: Element) {
  const mod = await getFrameworkFeatures();
  return mod.setupTransitions(root);
}

export const fade = async (el: Element, params?: any) =>
  (await getFrameworkFeatures()).fade(el, params);
export const fly = async (el: Element, params?: any) =>
  (await getFrameworkFeatures()).fly(el, params);
export const slide = async (el: Element, params?: any) =>
  (await getFrameworkFeatures()).slide(el, params);
export const scale = async (el: Element, params?: any) =>
  (await getFrameworkFeatures()).scale(el, params);
export const blur = async (el: Element, params?: any) =>
  (await getFrameworkFeatures()).blur(el, params);
export const crossfade = async (el: Element, params?: any) =>
  (await getFrameworkFeatures()).crossfade(el, params);

// Signal-based reactivity
import {
  reactive,
  $state,
  $derived,
  $effect,
  watchProp,
  toRaw,
  isReactive,
  reactiveArray,
} from "./signals.ts";


export {
  reactive,
  $state,
  $derived,
  $effect,
  watchProp,
  toRaw,
  isReactive,
  reactiveArray,
};

// DOM Utilities
export { cancelPendingDOMUpdates, flushDOMUpdatesNow } from "./dom.ts";

// Error boundaries
import {
  withErrorBoundary,
  createErrorFallback,
  getErrorBoundaryState,
  clearAllErrorBoundaries,
  isInErrorState,
  onComponentError,
} from "./error-boundary.ts";

export {
  withErrorBoundary,
  createErrorFallback,
  getErrorBoundaryState,
  clearAllErrorBoundaries,
  isInErrorState,
  onComponentError,
};

// DevTools & Debugging
import {
  createDevToolsPanel,
  updateDevToolsPanel,
  toggleDevTools,
  inspect,
  timing,
  memoryUsage,
} from "./debug.ts";

export {
  createDevToolsPanel,
  updateDevToolsPanel,
  toggleDevTools,
  inspect,
  timing,
  memoryUsage,
};

// WebSocket tab sharing
export async function createTabSync(config?: any) {
  const mod = await getFrameworkFeatures();
  return mod.createTabSync(config);
}

// IndexedDB persistence
export async function createIndexedDBPersistence(config?: any) {
  const mod = await getFrameworkFeatures();
  return mod.createIndexedDBPersistence(config);
}


// Accessibility
export async function announce(
  message: string,
  politeness?: "polite" | "assertive",
) {
  const mod = await getFrameworkFeatures();
  return mod.announce(message, politeness);
}

// Performance monitoring
export async function measure(name: string, fn: any, metadata?: any) {
  const mod = await getFrameworkFeatures();
  return mod.measure(name, fn, metadata);
}

import GoSPA from "./runtime-core.ts";
(GoSPA as any).remote = remote;
(GoSPA as any).remoteAction = remoteAction;

// WebSocket & Navigation APIs
(GoSPA as any).initWebSocket = initWebSocket;
(GoSPA as any).sendAction = sendAction;
(GoSPA as any).navigate = navigate;
(GoSPA as any).back = back;
(GoSPA as any).prefetch = prefetch;

// Islands & Hydration
(GoSPA as any).initIslands = initIslands;
(GoSPA as any).hydrateIsland = hydrateIsland;

// Signals & Reactivity (Svelte-like)
(GoSPA as any).reactive = (GoSPA as any).$state = (GoSPA as any).rune = $state;
(GoSPA as any).derived = (GoSPA as any).$derived = $derived;
(GoSPA as any).effect = (GoSPA as any).$effect = $effect;
(GoSPA as any).watchProp = watchProp;

// Transitions
(GoSPA as any).setupTransitions = setupTransitions;
(GoSPA as any).fade = fade;
(GoSPA as any).fly = fly;
(GoSPA as any).slide = slide;

// Error Handling
(GoSPA as any).withErrorBoundary = withErrorBoundary;
(GoSPA as any).onComponentError = onComponentError;

// Debugging
(GoSPA as any).inspect = inspect;
(GoSPA as any).timing = timing;

export default GoSPA;
