// GoSPA Client Runtime - Main entry point
// A lightweight runtime for reactive SPAs with Go/Fiber/Templ
//
// HTML sanitization is NOT included by default. The runtime trusts server-rendered
// content (Templ auto-escapes). For user-generated content, use 'gospa/runtime-secure'
//
// Bundle size: ~15KB (without DOMPurify)

// Core exports (re-exported from runtime-core for convenience)
import {
  init,
  createComponent,
  destroyComponent,
  getComponent,
  getState,
  setState,
  callAction,
  bind,
  autoInit,
  getWebSocket,
  getNavigation,
  getTransitions,
  remote,
  remoteAction,
  configureRemote,
  getRemotePrefix,
} from "./runtime-core.ts";

export {
  init,
  createComponent,
  destroyComponent,
  getComponent,
  getState,
  setState,
  callAction,
  bind,
  autoInit,
  getWebSocket,
  getNavigation,
  getTransitions,
  remote,
  remoteAction,
  configureRemote,
  getRemotePrefix,
};

export {
  Rune,
  Derived,
  Effect,
  StateMap,
  batch,
  rune,
  derived,
  effect,
  watch,
  untrack,
  preEffect,
  bindElement,
  bindTwoWay,
  renderIf,
  renderList,
} from "./runtime-core.ts";

// Export types
import { registerBinding, unregisterBinding } from "./dom.ts";
import { getFrameworkFeatures } from "./runtime-core.ts";
import type { NavigateOptions, NavigationOptions } from "./navigation.ts";
import type { StateMessage } from "./websocket.ts";

// Lazy-loaded wrappers for feature-rich modules
// This allows the standard runtime bundle to stay tiny while loading
// heavier functionality only when used ("on demand").

// Re-export types (static imports are fine for types as they are erased)
export type {
  ComponentDefinition,
  ComponentInstance,
  RuntimeConfig,
} from "./runtime-core.ts";
export type { NavigateOptions, NavigationOptions };
export type { StateMessage };
export type { Unsubscribe } from "./state.ts";
export type { RemoteOptions, RemoteResult } from "./remote.ts";

// WebSocket
export async function initWebSocket(config: any) {
  const mod = await getFrameworkFeatures();
  return mod.initWebSocket(config);
}

export async function getWebSocketClient() {
  const mod = await getFrameworkFeatures();
  return mod.getWebSocketClient();
}

export async function sendAction(name: string, payload?: any) {
  const mod = await getFrameworkFeatures();
  return mod.sendAction(name, payload);
}

// Navigation
export async function navigate(to: string, options?: any) {
  const mod = await getFrameworkFeatures();
  return mod.navigate(to, options);
}

export async function back() {
  const mod = await getFrameworkFeatures();
  return mod.back();
}

export async function prefetch(path: string) {
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

// Signal-based reactivity (proxy-based auto-tracking)
export {
  reactive,
  $state,
  $derived,
  $effect,
  // derived is already exported from runtime-core above
  effect as signalEffect,
  watchProp,
  toRaw,
  isReactive,
  reactiveArray,
} from "./signals.ts";

// RAF-batched DOM updates
export {
  cancelPendingDOMUpdates,
  flushDOMUpdatesNow,
  setSanitizer,
} from "./dom.ts";

// Error boundaries
export {
  withErrorBoundary,
  createErrorFallback,
  getErrorBoundaryState,
  clearAllErrorBoundaries,
  isInErrorState,
  onComponentError,
} from "./error-boundary.ts";

// DevTools & Debugging
export {
  createDevToolsPanel,
  updateDevToolsPanel,
  toggleDevTools,
  inspect,
  timing,
  memoryUsage,
} from "./debug.ts";

// WebSocket tab sharing (BroadcastChannel)
export async function createTabSync(config?: any) {
  const mod = await getFrameworkFeatures();
  return mod.createTabSync(config);
}

// IndexedDB persistence
export async function createIndexedDBPersistence(config?: any) {
  const mod = await getFrameworkFeatures();
  return mod.createIndexedDBPersistence(config);
}

// Accessibility enhancements
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
