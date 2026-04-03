import {
  $derived,
  $effect,
  $state,
  autoInit,
  bind,
  bindElement,
  bindTwoWay,
  callAction,
  cancelPendingDOMUpdates,
  clearAllErrorBoundaries,
  configureRemote,
  createComponent,
  createErrorFallback,
  destroyComponent,
  effect as effect2,
  flushDOMUpdatesNow,
  getComponent,
  getErrorBoundaryState,
  getFrameworkFeatures,
  getNavigation,
  getRemotePrefix,
  getState,
  getTransitions,
  getWebSocket,
  init,
  isInErrorState,
  isReactive,
  onComponentError,
  reactive,
  reactiveArray,
  remote,
  remoteAction,
  renderIf,
  renderList,
  setSanitizer,
  setState,
  toRaw,
  watchProp,
  withErrorBoundary
} from "./runtime-rq764jre.js";
import {
  Derived,
  Effect,
  Rune,
  StateMap,
  batch,
  createDevToolsPanel,
  derived,
  effect,
  inspect,
  memoryUsage,
  preEffect,
  rune,
  timing,
  toggleDevTools,
  untrack,
  updateDevToolsPanel,
  watch
} from "./websocket-g18v2mwh.js";
import"./runtime-3hqyeswk.js";
// client/src/runtime.ts
async function initWebSocket(config) {
  const mod = await getFrameworkFeatures();
  return mod.initWebSocket(config);
}
async function getWebSocketClient() {
  const mod = await getFrameworkFeatures();
  return mod.getWebSocketClient();
}
async function sendAction(name, payload) {
  const mod = await getFrameworkFeatures();
  return mod.sendAction(name, payload);
}
async function navigate(to, options) {
  const mod = await getFrameworkFeatures();
  return mod.navigate(to, options);
}
async function back() {
  const mod = await getFrameworkFeatures();
  return mod.back();
}
async function prefetch(path) {
  const mod = await getFrameworkFeatures();
  return mod.prefetch(path);
}
async function initIslands(config) {
  const mod = await getFrameworkFeatures();
  return mod.initIslands(config);
}
async function getIslandManager() {
  const mod = await getFrameworkFeatures();
  return mod.getIslandManager();
}
async function hydrateIsland(idOrName) {
  const mod = await getFrameworkFeatures();
  return mod.hydrateIsland(idOrName);
}
async function initStreaming(config) {
  const mod = await getFrameworkFeatures();
  return mod.initStreaming(config);
}
async function setupTransitions(root) {
  const mod = await getFrameworkFeatures();
  return mod.setupTransitions(root);
}
var fade = async (el, params) => (await getFrameworkFeatures()).fade(el, params);
var fly = async (el, params) => (await getFrameworkFeatures()).fly(el, params);
var slide = async (el, params) => (await getFrameworkFeatures()).slide(el, params);
var scale = async (el, params) => (await getFrameworkFeatures()).scale(el, params);
var blur = async (el, params) => (await getFrameworkFeatures()).blur(el, params);
var crossfade = async (el, params) => (await getFrameworkFeatures()).crossfade(el, params);
async function createTabSync(config) {
  const mod = await getFrameworkFeatures();
  return mod.createTabSync(config);
}
async function createIndexedDBPersistence(config) {
  const mod = await getFrameworkFeatures();
  return mod.createIndexedDBPersistence(config);
}
async function announce(message, politeness) {
  const mod = await getFrameworkFeatures();
  return mod.announce(message, politeness);
}
async function measure(name, fn, metadata) {
  const mod = await getFrameworkFeatures();
  return mod.measure(name, fn, metadata);
}
export {
  withErrorBoundary,
  watchProp,
  watch,
  updateDevToolsPanel,
  untrack,
  toggleDevTools,
  toRaw,
  timing,
  slide,
  effect2 as signalEffect,
  setupTransitions,
  setState,
  setSanitizer,
  sendAction,
  scale,
  rune,
  renderList,
  renderIf,
  remoteAction,
  remote,
  reactiveArray,
  reactive,
  prefetch,
  preEffect,
  onComponentError,
  navigate,
  memoryUsage,
  measure,
  isReactive,
  isInErrorState,
  inspect,
  initWebSocket,
  initStreaming,
  initIslands,
  init,
  hydrateIsland,
  getWebSocketClient,
  getWebSocket,
  getTransitions,
  getState,
  getRemotePrefix,
  getNavigation,
  getIslandManager,
  getErrorBoundaryState,
  getComponent,
  fly,
  flushDOMUpdatesNow,
  fade,
  effect,
  destroyComponent,
  derived,
  crossfade,
  createTabSync,
  createIndexedDBPersistence,
  createErrorFallback,
  createDevToolsPanel,
  createComponent,
  configureRemote,
  clearAllErrorBoundaries,
  cancelPendingDOMUpdates,
  callAction,
  blur,
  bindTwoWay,
  bindElement,
  bind,
  batch,
  back,
  autoInit,
  announce,
  StateMap,
  Rune,
  Effect,
  Derived,
  $state,
  $effect,
  $derived
};

export { initWebSocket, getWebSocketClient, sendAction, navigate, back, prefetch, initIslands, getIslandManager, hydrateIsland, initStreaming, setupTransitions, fade, fly, slide, scale, blur, crossfade, createTabSync, createIndexedDBPersistence, announce, measure };
