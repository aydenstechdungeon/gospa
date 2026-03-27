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
export type {
  ComponentDefinition,
  ComponentInstance,
  RuntimeConfig,
} from "./runtime-core.ts";
export type { Unsubscribe } from "./state.ts";
export type { RemoteOptions, RemoteResult } from "./remote.ts";

import { registerBinding, unregisterBinding } from "./dom.ts";
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
import {
  WSClient,
  initWebSocket,
  getWebSocketClient,
  sendAction,
  syncedRune,
  applyStateUpdate,
  type StateMessage,
} from "./websocket.ts";
import { initIslands, getIslandManager, hydrateIsland } from "./island.ts";
import { getPriorityScheduler, initPriorityHydration } from "./priority.ts";
import { initStreaming, getStreamingManager } from "./streaming.ts";
import { Resource, resourceReactive } from "./resource.ts";
import {
  navigate,
  back,
  forward,
  go,
  prefetch,
  getCurrentPath,
  isNavigating,
  onBeforeNavigate,
  onAfterNavigate,
  initNavigation,
  destroyNavigation,
  createNavigationState,
  setNavigationOptions,
  type NavigateOptions,
  type NavigationOptions,
} from "./navigation.ts";
import {
  setupTransitions,
  fade,
  fly,
  slide,
  scale,
  blur,
  crossfade,
} from "./transition.ts";

// Re-export DOM bindings
export { registerBinding, unregisterBinding };

// Re-export events
export { on, offAll, debounce, throttle, delegate, onKey, keys, transformers };

// Re-export Core Functionality for library users
export {
  // WebSocket
  WSClient,
  initWebSocket,
  getWebSocketClient,
  sendAction,
  syncedRune,
  applyStateUpdate,

  // Transitions
  fade,
  fly,
  slide,
  scale,
  blur,
  crossfade,
  setupTransitions,

  // Islands & Priority
  initIslands,
  getIslandManager,
  hydrateIsland,
  getPriorityScheduler,
  initPriorityHydration,

  // Streaming
  initStreaming,
  getStreamingManager,

  // Resources
  Resource,
  resourceReactive,

  // Navigation
  navigate,
  back,
  forward,
  go,
  prefetch,
  getCurrentPath,
  isNavigating,
  onBeforeNavigate,
  onAfterNavigate,
  initNavigation,
  destroyNavigation,
  createNavigationState,
  setNavigationOptions,
};

// Export types
export type { NavigateOptions, NavigationOptions, StateMessage };

// === New Runtime Enhancements ===

// Signal-based reactivity (proxy-based auto-tracking)
export {
  reactive,
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
export {
  WSTabSync,
  createTabSync,
  getTabSync,
  destroyTabSync,
} from "./ws-tab-sync.ts";

// IndexedDB persistence
export {
  IndexedDBPersistence,
  createIndexedDBPersistence,
  getIndexedDBPersistence,
  destroyIndexedDBPersistence,
} from "./indexeddb.ts";

// Accessibility enhancements
export {
  ScreenReaderAnnouncer,
  createAnnouncer,
  getAnnouncer,
  destroyAnnouncer,
  announce,
  aria,
  focus,
} from "./a11y.ts";

// Performance monitoring
export {
  PerformanceMonitor,
  createPerformanceMonitor,
  getPerformanceMonitor,
  destroyPerformanceMonitor,
  measure,
  measureAsync,
} from "./performance.ts";
