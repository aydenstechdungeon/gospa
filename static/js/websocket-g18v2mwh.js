import"./runtime-3hqyeswk.js";

// client/src/debug.ts
var devToolsPanel = null;
var devToolsInitialized = false;
function createDevToolsPanel() {
  if (!isDev() || devToolsInitialized)
    return;
  devToolsInitialized = true;
  devToolsPanel = document.createElement("div");
  devToolsPanel.id = "gospa-devtools";
  devToolsPanel.innerHTML = `
		<style>
			#gospa-devtools {
				position: fixed;
				bottom: 0;
				right: 0;
				width: 320px;
				max-height: 400px;
				background: #1a1a2e;
				color: #eee;
				font-family: 'SF Mono', 'Fira Code', monospace;
				font-size: 12px;
				border-top-left-radius: 8px;
				box-shadow: -4px -4px 20px rgba(0,0,0,0.3);
				z-index: 99999;
				overflow: hidden;
				display: flex;
				flex-direction: column;
			}
			#gospa-devtools-header {
				display: flex;
				justify-content: space-between;
				align-items: center;
				padding: 8px 12px;
				background: #16213e;
				border-bottom: 1px solid #0f3460;
				cursor: move;
			}
			#gospa-devtools-header span {
				font-weight: bold;
				color: #e94560;
			}
			#gospa-devtools-header button {
				background: none;
				border: none;
				color: #888;
				cursor: pointer;
				font-size: 16px;
				padding: 0 4px;
			}
			#gospa-devtools-header button:hover {
				color: #fff;
			}
			#gospa-devtools-tabs {
				display: flex;
				background: #16213e;
				border-bottom: 1px solid #0f3460;
			}
			#gospa-devtools-tabs button {
				flex: 1;
				background: none;
				border: none;
				color: #888;
				padding: 8px;
				cursor: pointer;
				font-size: 11px;
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			#gospa-devtools-tabs button.active {
				color: #e94560;
				border-bottom: 2px solid #e94560;
			}
			#gospa-devtools-content {
				flex: 1;
				overflow-y: auto;
				padding: 8px;
			}
			.gospa-devtools-section {
				margin-bottom: 12px;
			}
			.gospa-devtools-section-title {
				color: #e94560;
				font-weight: bold;
				margin-bottom: 4px;
				font-size: 11px;
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			.gospa-devtools-item {
				padding: 4px 8px;
				margin: 2px 0;
				background: #16213e;
				border-radius: 4px;
				font-size: 11px;
			}
			.gospa-devtools-item:hover {
				background: #0f3460;
			}
			.gospa-devtools-key {
				color: #00d9ff;
			}
			.gospa-devtools-value {
				color: #a8ff60;
			}
			.gospa-devtools-error {
				color: #ff6b6b;
			}
			.gospa-devtools-metric {
				display: flex;
				justify-content: space-between;
				padding: 4px 8px;
				margin: 2px 0;
				background: #16213e;
				border-radius: 4px;
			}
			.gospa-devtools-metric-label {
				color: #888;
			}
			.gospa-devtools-metric-value {
				color: #a8ff60;
				font-weight: bold;
			}
		</style>
		<div id="gospa-devtools-header">
			<span>GoSPA DevTools</span>
			<button id="gospa-devtools-close">×</button>
		</div>
		<div id="gospa-devtools-tabs">
			<button class="active" data-tab="components">Components</button>
			<button data-tab="state">State</button>
			<button data-tab="performance">Performance</button>
		</div>
		<div id="gospa-devtools-content">
			<div id="gospa-devtools-components" class="gospa-devtools-tab-content active"></div>
			<div id="gospa-devtools-state" class="gospa-devtools-tab-content" style="display:none"></div>
			<div id="gospa-devtools-performance" class="gospa-devtools-tab-content" style="display:none"></div>
		</div>
	`;
  document.body.appendChild(devToolsPanel);
  const closeBtn = devToolsPanel.querySelector("#gospa-devtools-close");
  closeBtn?.addEventListener("click", () => {
    devToolsPanel?.remove();
    devToolsPanel = null;
    devToolsInitialized = false;
  });
  const tabs = devToolsPanel.querySelectorAll("#gospa-devtools-tabs button");
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      tabs.forEach((t) => t.classList.remove("active"));
      tab.classList.add("active");
      const tabName = tab.getAttribute("data-tab");
      const contents = devToolsPanel?.querySelectorAll(".gospa-devtools-tab-content");
      contents?.forEach((content) => {
        content.style.display = content.id === `gospa-devtools-${tabName}` ? "block" : "none";
      });
    });
  });
  const header = devToolsPanel.querySelector("#gospa-devtools-header");
  let isDragging = false;
  let dragOffsetX = 0;
  let dragOffsetY = 0;
  header?.addEventListener("mousedown", (e) => {
    const mouseEvent = e;
    isDragging = true;
    dragOffsetX = mouseEvent.clientX - (devToolsPanel?.offsetLeft || 0);
    dragOffsetY = mouseEvent.clientY - (devToolsPanel?.offsetTop || 0);
  });
  document.addEventListener("mousemove", (e) => {
    if (isDragging && devToolsPanel) {
      const mouseEvent = e;
      devToolsPanel.style.left = `${mouseEvent.clientX - dragOffsetX}px`;
      devToolsPanel.style.top = `${mouseEvent.clientY - dragOffsetY}px`;
      devToolsPanel.style.right = "auto";
      devToolsPanel.style.bottom = "auto";
    }
  });
  document.addEventListener("mouseup", () => {
    isDragging = false;
  });
  console.log("%c[GoSPA DevTools] Panel initialized", "color: #e94560");
}
function updateDevToolsPanel() {
  if (!devToolsPanel || !isDev())
    return;
  const componentsContent = devToolsPanel.querySelector("#gospa-devtools-components");
  if (componentsContent) {
    const components = window.__GOSPA__?.components;
    if (components) {
      let html = '<div class="gospa-devtools-section">';
      html += '<div class="gospa-devtools-section-title">Components</div>';
      for (const [id, component] of components) {
        const stateKeys = component.states ? Array.from(component.states.keys()) : [];
        html += `<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${id}</span>
					<span class="gospa-devtools-value">(${stateKeys.length} states)</span>
				</div>`;
      }
      html += "</div>";
      componentsContent.innerHTML = html;
    }
  }
  const stateContent = devToolsPanel.querySelector("#gospa-devtools-state");
  if (stateContent) {
    const globalState = window.__GOSPA__?.globalState;
    if (globalState) {
      let html = '<div class="gospa-devtools-section">';
      html += '<div class="gospa-devtools-section-title">Global State</div>';
      const stateObj = globalState.toJSON ? globalState.toJSON() : {};
      for (const [key, value] of Object.entries(stateObj)) {
        const valueStr = typeof value === "object" ? JSON.stringify(value) : String(value);
        html += `<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${key}:</span>
					<span class="gospa-devtools-value">${valueStr}</span>
				</div>`;
      }
      html += "</div>";
      const stores = window.__GOSPA_STORES__;
      if (stores) {
        html += '<div class="gospa-devtools-section">';
        html += '<div class="gospa-devtools-section-title">Reactive Stores</div>';
        for (const [name, store] of Object.entries(stores)) {
          const valueStr = typeof store === "object" ? JSON.stringify(store) : String(store);
          html += `<div class="gospa-devtools-item">
            <span class="gospa-devtools-key">${name}:</span>
            <span class="gospa-devtools-value">${valueStr}</span>
          </div>`;
        }
        html += "</div>";
      }
      stateContent.innerHTML = html;
    }
  }
  const perfContent = devToolsPanel.querySelector("#gospa-devtools-performance");
  if (perfContent) {
    let html = '<div class="gospa-devtools-section">';
    html += '<div class="gospa-devtools-section-title">Performance Metrics</div>';
    if ("memory" in performance && performance.memory) {
      const memory = performance.memory;
      const usedMB = (memory.usedJSHeapSize / 1024 / 1024).toFixed(2);
      const totalMB = (memory.totalJSHeapSize / 1024 / 1024).toFixed(2);
      html += `<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Heap Used</span>
				<span class="gospa-devtools-metric-value">${usedMB}MB / ${totalMB}MB</span>
			</div>`;
    }
    const timing = performance.getEntriesByType("measure");
    if (timing.length > 0) {
      const lastTiming = timing[timing.length - 1];
      html += `<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Last Measure</span>
				<span class="gospa-devtools-metric-value">${lastTiming.name}: ${lastTiming.duration.toFixed(2)}ms</span>
			</div>`;
    }
    html += "</div>";
    perfContent.innerHTML = html;
  }
}
function toggleDevTools() {
  if (!isDev())
    return;
  if (devToolsPanel) {
    devToolsPanel.remove();
    devToolsPanel = null;
    devToolsInitialized = false;
  } else {
    createDevToolsPanel();
  }
}
function isDev() {
  return typeof window !== "undefined" && window.__GOSPA_DEV__ !== false;
}
function inspect(...values) {
  if (!isDev()) {
    return { with: () => {} };
  }
  let firstRun = true;
  const callbacks = [];
  const getValues = () => values.map((v) => typeof v === "function" ? v() : v);
  const logValues = (type) => {
    const currentValues = getValues();
    console.log(`%c[${type}]`, "color: #888", ...currentValues);
    callbacks.forEach((cb) => cb(type, currentValues));
  };
  new Effect(() => {
    getValues();
    if (firstRun) {
      firstRun = false;
      logValues("init");
    } else {
      logValues("update");
    }
  });
  return {
    with: (callback) => {
      callbacks.push(callback);
    }
  };
}
inspect.trace = (label) => {
  if (!isDev())
    return;
  console.log(`%c[trace]${label ? ` ${label}` : ""}`, "color: #666; font-style: italic");
};
function timing(name) {
  if (!isDev()) {
    return { end: () => {} };
  }
  const start = performance.now();
  return {
    end: () => {
      const duration = performance.now() - start;
      console.log(`%c[timing] ${name}: ${duration.toFixed(2)}ms`, "color: #0a0");
    }
  };
}
function memoryUsage(label) {
  if (!isDev())
    return;
  if ("memory" in performance && performance.memory) {
    const memory = performance.memory;
    const mb = (memory.usedJSHeapSize / 1024 / 1024).toFixed(2);
    console.log(`%c[memory] ${label}: ${mb}MB`, "color: #a0a");
  }
}
function debugLog(...args) {
  if (!isDev())
    return;
  console.log("%c[debug]", "color: #888", ...args);
}
function createInspector(name, state) {
  if (!isDev()) {
    return { log: () => {}, dispose: () => {} };
  }
  console.log(`%c[inspector] ${name} created`, "color: #08f");
  const unsub = state.subscribe((value) => {
    console.log(`%c[${name}]`, "color: #08f", value);
  });
  return {
    log: () => {
      console.log(`%c[${name}]`, "color: #08f", state.get());
    },
    dispose: () => {
      unsub();
      console.log(`%c[inspector] ${name} disposed`, "color: #888");
    }
  };
}

// client/src/state.ts
var runeId = 0;
var effectId = 0;
var batchDepth = 0;
var pendingNotifications = new Set;
var autoBatchScheduled = false;
function scheduleAutoBatch() {
  if (autoBatchScheduled || batchDepth > 0)
    return;
  autoBatchScheduled = true;
  queueMicrotask(() => {
    autoBatchScheduled = false;
    if (batchDepth === 0 && pendingNotifications.size > 0) {
      const pending = [...pendingNotifications];
      pendingNotifications.clear();
      pending.forEach((n) => n.notify());
    }
  });
}
var activeDisposables = new Set;
var finalizationRegistry = null;
if (typeof globalThis.FinalizationRegistry !== "undefined") {
  finalizationRegistry = new globalThis.FinalizationRegistry((_id) => {});
}
var disposalTrackingEnabled = false;
function trackDisposable(disposable) {
  if (disposalTrackingEnabled) {
    if (typeof globalThis.WeakRef !== "undefined") {
      const WeakRefCtor = globalThis.WeakRef;
      activeDisposables.add(new WeakRefCtor(disposable));
    } else {
      activeDisposables.add({ deref: () => disposable });
    }
    if (finalizationRegistry) {
      finalizationRegistry.register(disposable, `disposable-${Date.now()}`);
    }
  }
  return disposable;
}
var currentEffect = null;
var effectStack = [];
function getCurrentEffect() {
  return currentEffect;
}
function fastDeepEqual(a, b) {
  if (a === b)
    return true;
  if (typeof a !== typeof b)
    return false;
  if (typeof a !== "object" || a === null || b === null)
    return false;
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length)
      return false;
    for (let i = 0;i < a.length; i++) {
      if (!fastDeepEqual(a[i], b[i]))
        return false;
    }
    return true;
  }
  if (a instanceof Date && b instanceof Date) {
    return a.getTime() === b.getTime();
  }
  if (a instanceof Set && b instanceof Set) {
    if (a.size !== b.size)
      return false;
    for (const val of a) {
      if (!b.has(val))
        return false;
    }
    return true;
  }
  if (a instanceof Map && b instanceof Map) {
    if (a.size !== b.size)
      return false;
    for (const [key, val] of a) {
      if (!b.has(key) || !fastDeepEqual(val, b.get(key)))
        return false;
    }
    return true;
  }
  if (Array.isArray(a) !== Array.isArray(b))
    return false;
  const keysA = Object.keys(a);
  const keysB = Object.keys(b);
  if (keysA.length !== keysB.length)
    return false;
  for (const key of keysA) {
    if (!Object.prototype.hasOwnProperty.call(b, key))
      return false;
    if (!fastDeepEqual(a[key], b[key]))
      return false;
  }
  return true;
}
function batch(fn) {
  batchDepth++;
  try {
    fn();
  } finally {
    batchDepth--;
    if (batchDepth === 0) {
      const pending = [...pendingNotifications];
      pendingNotifications.clear();
      pending.forEach((n) => n.notify());
    }
  }
}

class Rune {
  _value;
  _id;
  _subscribers = new Set;
  _dirty = false;
  _disposed = false;
  _hasPendingOldValue = false;
  _pendingOldValue;
  constructor(initialValue) {
    this._value = initialValue;
    this._id = ++runeId;
    trackDisposable(this);
  }
  get value() {
    this.trackDependency();
    return this._value;
  }
  set value(newValue) {
    if (this._equal(this._value, newValue))
      return;
    const oldValue = this._value;
    this._value = newValue;
    this._dirty = true;
    this._notifySubscribers(oldValue);
  }
  get() {
    this.trackDependency();
    return this._value;
  }
  set(newValue) {
    this.value = newValue;
  }
  peek() {
    return this._value;
  }
  update(fn) {
    this.value = fn(this._value);
  }
  subscribe(fn) {
    this._subscribers.add(fn);
    return () => this._subscribers.delete(fn);
  }
  _notifySubscribers(oldValue) {
    if (!this._hasPendingOldValue) {
      this._hasPendingOldValue = true;
      this._pendingOldValue = oldValue;
    }
    if (batchDepth > 0) {
      pendingNotifications.add(this);
      return;
    }
    this.notify(oldValue);
  }
  notify(prevValue) {
    const value = this._value;
    const old = this._hasPendingOldValue ? this._pendingOldValue : prevValue !== undefined ? prevValue : value;
    this._hasPendingOldValue = false;
    this._pendingOldValue = undefined;
    this._subscribers.forEach((fn) => fn(value, old));
  }
  _equal(a, b) {
    if (Object.is(a, b))
      return true;
    if (typeof a !== typeof b)
      return false;
    if (typeof a !== "object" || a === null || b === null)
      return false;
    return fastDeepEqual(a, b);
  }
  trackDependency() {
    if (currentEffect) {
      currentEffect.addDependency(this);
    }
  }
  toJSON() {
    return { id: this._id, value: this._value };
  }
  dispose() {
    this._disposed = true;
    this._subscribers.clear();
  }
  isDisposed() {
    return this._disposed;
  }
}
function rune(initialValue) {
  return new Rune(initialValue);
}

class Derived {
  _value;
  _compute;
  _dependencies = new Set;
  _subscribers = new Set;
  _depUnsubs = new Map;
  _dirty = true;
  _disposed = false;
  constructor(compute) {
    this._compute = compute;
    this._value = undefined;
    this._recompute();
  }
  get value() {
    if (this._dirty) {
      this._recompute();
    }
    this.trackDependency();
    return this._value;
  }
  get() {
    return this.value;
  }
  subscribe(fn) {
    this._subscribers.add(fn);
    return () => this._subscribers.delete(fn);
  }
  _recompute() {
    const oldDeps = new Set(this._dependencies);
    this._dependencies.clear();
    const prevEffect = currentEffect;
    const collector = {
      addDependency: (rune2) => {
        this._dependencies.add(rune2);
      }
    };
    currentEffect = collector;
    try {
      this._value = this._compute();
      this._dirty = false;
    } finally {
      currentEffect = prevEffect;
    }
    oldDeps.forEach((dep) => {
      if (!this._dependencies.has(dep)) {
        const unsub = this._depUnsubs.get(dep);
        if (unsub) {
          unsub();
          this._depUnsubs.delete(dep);
        }
      }
    });
    this._dependencies.forEach((dep) => {
      if (!oldDeps.has(dep)) {
        const unsub = dep.subscribe(() => {
          this._dirty = true;
          this._notifySubscribers();
        });
        this._depUnsubs.set(dep, unsub);
      }
    });
  }
  _notifySubscribers() {
    if (batchDepth > 0) {
      pendingNotifications.add(this);
      return;
    }
    this.notify();
  }
  notify() {
    const prevValue = this._dirty ? undefined : this._value;
    if (this._dirty) {
      this._recompute();
    }
    const value = this._value;
    this._subscribers.forEach((fn) => fn(value, prevValue ?? value));
  }
  trackDependency() {
    if (currentEffect) {
      currentEffect.addDependency(this);
    }
  }
  dispose() {
    this._disposed = true;
    this._depUnsubs.forEach((unsub) => unsub());
    this._depUnsubs.clear();
    this._dependencies.clear();
    this._subscribers.clear();
  }
  isDisposed() {
    return this._disposed;
  }
}
function derived(compute) {
  return new Derived(compute);
}

class Effect {
  _fn;
  _cleanup;
  _dependencies = new Set;
  _depUnsubs = new Map;
  _id;
  _active = true;
  _disposed = false;
  constructor(fn) {
    this._fn = fn;
    this._id = ++effectId;
    this._cleanup = undefined;
    this._run();
  }
  _run() {
    if (!this._active || this._disposed)
      return;
    if (this._cleanup) {
      this._cleanup();
      this._cleanup = undefined;
    }
    const oldDeps = new Set(this._dependencies);
    this._dependencies.clear();
    effectStack.push(this);
    currentEffect = this;
    try {
      this._cleanup = this._fn();
    } finally {
      effectStack.pop();
      currentEffect = effectStack[effectStack.length - 1] || null;
    }
    oldDeps.forEach((dep) => {
      if (!this._dependencies.has(dep)) {
        const unsub = this._depUnsubs.get(dep);
        if (unsub) {
          unsub();
          this._depUnsubs.delete(dep);
        }
      }
    });
    this._dependencies.forEach((dep) => {
      if (!oldDeps.has(dep)) {
        const unsub = dep.subscribe(() => this.notify());
        this._depUnsubs.set(dep, unsub);
      }
    });
  }
  addDependency(rune2) {
    this._dependencies.add(rune2);
  }
  notify() {
    this._run();
  }
  pause() {
    this._active = false;
  }
  resume() {
    this._active = true;
    this._run();
  }
  dispose() {
    if (this._cleanup) {
      this._cleanup();
    }
    this._disposed = true;
    this._depUnsubs.forEach((unsub) => unsub());
    this._depUnsubs.clear();
    this._dependencies.clear();
  }
  isDisposed() {
    return this._disposed;
  }
}
function effect(fn) {
  return new Effect(fn);
}
function watch(sources, callback) {
  const sourceArray = Array.isArray(sources) ? sources : [sources];
  const unsubscribers = [];
  let previousValues = sourceArray.map((source) => source.get());
  sourceArray.forEach((source) => {
    unsubscribers.push(source.subscribe(() => {
      const values = sourceArray.map((s) => s.get());
      const oldValues = previousValues;
      previousValues = [...values];
      callback(Array.isArray(sources) ? values : values[0], Array.isArray(sources) ? oldValues : oldValues[0]);
    }));
  });
  return () => unsubscribers.forEach((unsub) => unsub());
}

class StateMap {
  _runes = new Map;
  set(key, value) {
    const existing = this._runes.get(key);
    if (existing) {
      existing.set(value);
      return existing;
    }
    const r = new Rune(value);
    this._runes.set(key, r);
    return r;
  }
  get(key) {
    return this._runes.get(key);
  }
  has(key) {
    return this._runes.has(key);
  }
  delete(key) {
    return this._runes.delete(key);
  }
  clear() {
    this._runes.clear();
  }
  toJSON() {
    const result = {};
    this._runes.forEach((rune2, key) => {
      result[key] = rune2.get();
    });
    return result;
  }
  fromJSON(data) {
    Object.entries(data).forEach(([key, value]) => {
      if (this._runes.has(key)) {
        this._runes.get(key).set(value);
      } else {
        this.set(key, value);
      }
    });
  }
  dispose() {
    this._runes.forEach((rune2) => {
      if ("dispose" in rune2 && typeof rune2.dispose === "function") {
        rune2.dispose();
      }
    });
    this._runes.clear();
  }
  isDisposed() {
    return this._runes.size === 0;
  }
}
function untrack(fn) {
  const prevEffect = currentEffect;
  currentEffect = null;
  try {
    return fn();
  } finally {
    currentEffect = prevEffect;
  }
}

class PreEffect extends Effect {
  static _preEffects = [];
  static _scheduled = false;
  constructor(fn) {
    super(fn);
    PreEffect._preEffects.push(this);
    PreEffect._scheduleFlush();
  }
  static _scheduleFlush() {
    if (!PreEffect._scheduled) {
      PreEffect._scheduled = true;
      queueMicrotask(() => {
        PreEffect._scheduled = false;
        const effects = [...PreEffect._preEffects];
        PreEffect._preEffects = [];
        effects.forEach((e) => e.notify());
      });
    }
  }
  dispose() {
    const idx = PreEffect._preEffects.indexOf(this);
    if (idx > -1)
      PreEffect._preEffects.splice(idx, 1);
    super.dispose();
  }
}
function preEffect(fn) {
  return new PreEffect(fn);
}

class RuneRaw {
  _value;
  _id;
  _subscribers = new Set;
  constructor(initialValue) {
    this._value = initialValue;
    this._id = ++runeId;
  }
  get value() {
    this.trackDependency();
    return this._value;
  }
  set value(newValue) {
    if (Object.is(this._value, newValue))
      return;
    const oldValue = this._value;
    this._value = newValue;
    this._notifySubscribers(oldValue);
  }
  get() {
    this.trackDependency();
    return this._value;
  }
  set(newValue) {
    this.value = newValue;
  }
  subscribe(fn) {
    this._subscribers.add(fn);
    return () => this._subscribers.delete(fn);
  }
  _notifySubscribers(_oldValue) {
    if (batchDepth > 0) {
      pendingNotifications.add(this);
      return;
    }
    pendingNotifications.add(this);
    scheduleAutoBatch();
  }
  notify() {
    const value = this._value;
    this._subscribers.forEach((fn) => fn(value, this._value));
  }
  trackDependency() {
    if (currentEffect) {
      currentEffect.addDependency(this);
    }
  }
  snapshot() {
    const val = this._value;
    if (typeof val === "object" && val !== null) {
      if (Array.isArray(val))
        return [...val];
      return { ...val };
    }
    return val;
  }
}
class DerivedAsync {
  _value;
  _error;
  _status = "pending";
  _compute;
  _dependencies = new Set;
  _subscribers = new Set;
  _dirty = true;
  _disposed = false;
  _abortController = null;
  constructor(compute) {
    this._compute = compute;
    this._recompute();
  }
  get value() {
    if (this._dirty) {
      this._recompute();
    }
    this.trackDependency();
    return this._value;
  }
  get error() {
    return this._error;
  }
  get status() {
    return this._status;
  }
  get isPending() {
    return this._status === "pending";
  }
  get isSuccess() {
    return this._status === "success";
  }
  get isError() {
    return this._status === "error";
  }
  get() {
    return this.value;
  }
  subscribe(fn) {
    this._subscribers.add(fn);
    return () => this._subscribers.delete(fn);
  }
  async _recompute() {
    if (this._abortController) {
      this._abortController.abort();
    }
    this._abortController = new AbortController;
    const oldDeps = new Set(this._dependencies);
    this._dependencies.clear();
    const prevEffect = currentEffect;
    const collector = {
      addDependency: (rune2) => {
        this._dependencies.add(rune2);
      }
    };
    currentEffect = collector;
    let promise;
    try {
      promise = this._compute();
      this._dirty = false;
    } finally {
      currentEffect = prevEffect;
    }
    this._dependencies.forEach((dep) => {
      if (!oldDeps.has(dep)) {
        dep.subscribe(() => {
          this._dirty = true;
          this._recompute();
        });
      }
    });
    this._status = "pending";
    this._notifySubscribers();
    try {
      const result = await promise;
      if (this._abortController?.signal.aborted)
        return;
      this._value = result;
      this._error = undefined;
      this._status = "success";
    } catch (err) {
      if (this._abortController?.signal.aborted)
        return;
      this._error = err;
      this._status = "error";
    }
    this._notifySubscribers();
  }
  _notifySubscribers() {
    if (batchDepth > 0) {
      pendingNotifications.add(this);
      return;
    }
    this.notify();
  }
  notify() {
    const value = this._value;
    this._subscribers.forEach((fn) => fn(value, this._value));
  }
  trackDependency() {
    if (currentEffect) {
      currentEffect.addDependency(this);
    }
  }
  dispose() {
    this._disposed = true;
    if (this._abortController) {
      this._abortController.abort();
    }
    this._dependencies.clear();
    this._subscribers.clear();
  }
  isDisposed() {
    return this._disposed;
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/utils/utf8.mjs
function utf8Count(str) {
  const strLength = str.length;
  let byteLength = 0;
  let pos = 0;
  while (pos < strLength) {
    let value = str.charCodeAt(pos++);
    if ((value & 4294967168) === 0) {
      byteLength++;
      continue;
    } else if ((value & 4294965248) === 0) {
      byteLength += 2;
    } else {
      if (value >= 55296 && value <= 56319) {
        if (pos < strLength) {
          const extra = str.charCodeAt(pos);
          if ((extra & 64512) === 56320) {
            ++pos;
            value = ((value & 1023) << 10) + (extra & 1023) + 65536;
          }
        }
      }
      if ((value & 4294901760) === 0) {
        byteLength += 3;
      } else {
        byteLength += 4;
      }
    }
  }
  return byteLength;
}
function utf8EncodeJs(str, output, outputOffset) {
  const strLength = str.length;
  let offset = outputOffset;
  let pos = 0;
  while (pos < strLength) {
    let value = str.charCodeAt(pos++);
    if ((value & 4294967168) === 0) {
      output[offset++] = value;
      continue;
    } else if ((value & 4294965248) === 0) {
      output[offset++] = value >> 6 & 31 | 192;
    } else {
      if (value >= 55296 && value <= 56319) {
        if (pos < strLength) {
          const extra = str.charCodeAt(pos);
          if ((extra & 64512) === 56320) {
            ++pos;
            value = ((value & 1023) << 10) + (extra & 1023) + 65536;
          }
        }
      }
      if ((value & 4294901760) === 0) {
        output[offset++] = value >> 12 & 15 | 224;
        output[offset++] = value >> 6 & 63 | 128;
      } else {
        output[offset++] = value >> 18 & 7 | 240;
        output[offset++] = value >> 12 & 63 | 128;
        output[offset++] = value >> 6 & 63 | 128;
      }
    }
    output[offset++] = value & 63 | 128;
  }
}
var sharedTextEncoder = new TextEncoder;
var TEXT_ENCODER_THRESHOLD = 50;
function utf8EncodeTE(str, output, outputOffset) {
  sharedTextEncoder.encodeInto(str, output.subarray(outputOffset));
}
function utf8Encode(str, output, outputOffset) {
  if (str.length > TEXT_ENCODER_THRESHOLD) {
    utf8EncodeTE(str, output, outputOffset);
  } else {
    utf8EncodeJs(str, output, outputOffset);
  }
}
var CHUNK_SIZE = 4096;
function utf8DecodeJs(bytes, inputOffset, byteLength) {
  let offset = inputOffset;
  const end = offset + byteLength;
  const units = [];
  let result = "";
  while (offset < end) {
    const byte1 = bytes[offset++];
    if ((byte1 & 128) === 0) {
      units.push(byte1);
    } else if ((byte1 & 224) === 192) {
      const byte2 = bytes[offset++] & 63;
      units.push((byte1 & 31) << 6 | byte2);
    } else if ((byte1 & 240) === 224) {
      const byte2 = bytes[offset++] & 63;
      const byte3 = bytes[offset++] & 63;
      units.push((byte1 & 31) << 12 | byte2 << 6 | byte3);
    } else if ((byte1 & 248) === 240) {
      const byte2 = bytes[offset++] & 63;
      const byte3 = bytes[offset++] & 63;
      const byte4 = bytes[offset++] & 63;
      let unit = (byte1 & 7) << 18 | byte2 << 12 | byte3 << 6 | byte4;
      if (unit > 65535) {
        unit -= 65536;
        units.push(unit >>> 10 & 1023 | 55296);
        unit = 56320 | unit & 1023;
      }
      units.push(unit);
    } else {
      units.push(byte1);
    }
    if (units.length >= CHUNK_SIZE) {
      result += String.fromCharCode(...units);
      units.length = 0;
    }
  }
  if (units.length > 0) {
    result += String.fromCharCode(...units);
  }
  return result;
}
var sharedTextDecoder = new TextDecoder;
var TEXT_DECODER_THRESHOLD = 200;
function utf8DecodeTD(bytes, inputOffset, byteLength) {
  const stringBytes = bytes.subarray(inputOffset, inputOffset + byteLength);
  return sharedTextDecoder.decode(stringBytes);
}
function utf8Decode(bytes, inputOffset, byteLength) {
  if (byteLength > TEXT_DECODER_THRESHOLD) {
    return utf8DecodeTD(bytes, inputOffset, byteLength);
  } else {
    return utf8DecodeJs(bytes, inputOffset, byteLength);
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/ExtData.mjs
class ExtData {
  type;
  data;
  constructor(type, data) {
    this.type = type;
    this.data = data;
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/DecodeError.mjs
class DecodeError extends Error {
  constructor(message) {
    super(message);
    const proto = Object.create(DecodeError.prototype);
    Object.setPrototypeOf(this, proto);
    Object.defineProperty(this, "name", {
      configurable: true,
      enumerable: false,
      value: DecodeError.name
    });
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/utils/int.mjs
var UINT32_MAX = 4294967295;
function setUint64(view, offset, value) {
  const high = value / 4294967296;
  const low = value;
  view.setUint32(offset, high);
  view.setUint32(offset + 4, low);
}
function setInt64(view, offset, value) {
  const high = Math.floor(value / 4294967296);
  const low = value;
  view.setUint32(offset, high);
  view.setUint32(offset + 4, low);
}
function getInt64(view, offset) {
  const high = view.getInt32(offset);
  const low = view.getUint32(offset + 4);
  return high * 4294967296 + low;
}
function getUint64(view, offset) {
  const high = view.getUint32(offset);
  const low = view.getUint32(offset + 4);
  return high * 4294967296 + low;
}

// client/node_modules/@msgpack/msgpack/dist.esm/timestamp.mjs
var EXT_TIMESTAMP = -1;
var TIMESTAMP32_MAX_SEC = 4294967296 - 1;
var TIMESTAMP64_MAX_SEC = 17179869184 - 1;
function encodeTimeSpecToTimestamp({ sec, nsec }) {
  if (sec >= 0 && nsec >= 0 && sec <= TIMESTAMP64_MAX_SEC) {
    if (nsec === 0 && sec <= TIMESTAMP32_MAX_SEC) {
      const rv = new Uint8Array(4);
      const view = new DataView(rv.buffer);
      view.setUint32(0, sec);
      return rv;
    } else {
      const secHigh = sec / 4294967296;
      const secLow = sec & 4294967295;
      const rv = new Uint8Array(8);
      const view = new DataView(rv.buffer);
      view.setUint32(0, nsec << 2 | secHigh & 3);
      view.setUint32(4, secLow);
      return rv;
    }
  } else {
    const rv = new Uint8Array(12);
    const view = new DataView(rv.buffer);
    view.setUint32(0, nsec);
    setInt64(view, 4, sec);
    return rv;
  }
}
function encodeDateToTimeSpec(date) {
  const msec = date.getTime();
  const sec = Math.floor(msec / 1000);
  const nsec = (msec - sec * 1000) * 1e6;
  const nsecInSec = Math.floor(nsec / 1e9);
  return {
    sec: sec + nsecInSec,
    nsec: nsec - nsecInSec * 1e9
  };
}
function encodeTimestampExtension(object) {
  if (object instanceof Date) {
    const timeSpec = encodeDateToTimeSpec(object);
    return encodeTimeSpecToTimestamp(timeSpec);
  } else {
    return null;
  }
}
function decodeTimestampToTimeSpec(data) {
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
  switch (data.byteLength) {
    case 4: {
      const sec = view.getUint32(0);
      const nsec = 0;
      return { sec, nsec };
    }
    case 8: {
      const nsec30AndSecHigh2 = view.getUint32(0);
      const secLow32 = view.getUint32(4);
      const sec = (nsec30AndSecHigh2 & 3) * 4294967296 + secLow32;
      const nsec = nsec30AndSecHigh2 >>> 2;
      return { sec, nsec };
    }
    case 12: {
      const sec = getInt64(view, 4);
      const nsec = view.getUint32(0);
      return { sec, nsec };
    }
    default:
      throw new DecodeError(`Unrecognized data size for timestamp (expected 4, 8, or 12): ${data.length}`);
  }
}
function decodeTimestampExtension(data) {
  const timeSpec = decodeTimestampToTimeSpec(data);
  return new Date(timeSpec.sec * 1000 + timeSpec.nsec / 1e6);
}
var timestampExtension = {
  type: EXT_TIMESTAMP,
  encode: encodeTimestampExtension,
  decode: decodeTimestampExtension
};

// client/node_modules/@msgpack/msgpack/dist.esm/ExtensionCodec.mjs
class ExtensionCodec {
  static defaultCodec = new ExtensionCodec;
  __brand;
  builtInEncoders = [];
  builtInDecoders = [];
  encoders = [];
  decoders = [];
  constructor() {
    this.register(timestampExtension);
  }
  register({ type, encode, decode }) {
    if (type >= 0) {
      this.encoders[type] = encode;
      this.decoders[type] = decode;
    } else {
      const index = -1 - type;
      this.builtInEncoders[index] = encode;
      this.builtInDecoders[index] = decode;
    }
  }
  tryToEncode(object, context) {
    for (let i = 0;i < this.builtInEncoders.length; i++) {
      const encodeExt = this.builtInEncoders[i];
      if (encodeExt != null) {
        const data = encodeExt(object, context);
        if (data != null) {
          const type = -1 - i;
          return new ExtData(type, data);
        }
      }
    }
    for (let i = 0;i < this.encoders.length; i++) {
      const encodeExt = this.encoders[i];
      if (encodeExt != null) {
        const data = encodeExt(object, context);
        if (data != null) {
          const type = i;
          return new ExtData(type, data);
        }
      }
    }
    if (object instanceof ExtData) {
      return object;
    }
    return null;
  }
  decode(data, type, context) {
    const decodeExt = type < 0 ? this.builtInDecoders[-1 - type] : this.decoders[type];
    if (decodeExt) {
      return decodeExt(data, type, context);
    } else {
      return new ExtData(type, data);
    }
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/utils/typedArrays.mjs
function isArrayBufferLike(buffer) {
  return buffer instanceof ArrayBuffer || typeof SharedArrayBuffer !== "undefined" && buffer instanceof SharedArrayBuffer;
}
function ensureUint8Array(buffer) {
  if (buffer instanceof Uint8Array) {
    return buffer;
  } else if (ArrayBuffer.isView(buffer)) {
    return new Uint8Array(buffer.buffer, buffer.byteOffset, buffer.byteLength);
  } else if (isArrayBufferLike(buffer)) {
    return new Uint8Array(buffer);
  } else {
    return Uint8Array.from(buffer);
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/Encoder.mjs
var DEFAULT_MAX_DEPTH = 100;
var DEFAULT_INITIAL_BUFFER_SIZE = 2048;

class Encoder {
  extensionCodec;
  context;
  useBigInt64;
  maxDepth;
  initialBufferSize;
  sortKeys;
  forceFloat32;
  ignoreUndefined;
  forceIntegerToFloat;
  pos;
  view;
  bytes;
  entered = false;
  constructor(options) {
    this.extensionCodec = options?.extensionCodec ?? ExtensionCodec.defaultCodec;
    this.context = options?.context;
    this.useBigInt64 = options?.useBigInt64 ?? false;
    this.maxDepth = options?.maxDepth ?? DEFAULT_MAX_DEPTH;
    this.initialBufferSize = options?.initialBufferSize ?? DEFAULT_INITIAL_BUFFER_SIZE;
    this.sortKeys = options?.sortKeys ?? false;
    this.forceFloat32 = options?.forceFloat32 ?? false;
    this.ignoreUndefined = options?.ignoreUndefined ?? false;
    this.forceIntegerToFloat = options?.forceIntegerToFloat ?? false;
    this.pos = 0;
    this.view = new DataView(new ArrayBuffer(this.initialBufferSize));
    this.bytes = new Uint8Array(this.view.buffer);
  }
  clone() {
    return new Encoder({
      extensionCodec: this.extensionCodec,
      context: this.context,
      useBigInt64: this.useBigInt64,
      maxDepth: this.maxDepth,
      initialBufferSize: this.initialBufferSize,
      sortKeys: this.sortKeys,
      forceFloat32: this.forceFloat32,
      ignoreUndefined: this.ignoreUndefined,
      forceIntegerToFloat: this.forceIntegerToFloat
    });
  }
  reinitializeState() {
    this.pos = 0;
  }
  encodeSharedRef(object) {
    if (this.entered) {
      const instance = this.clone();
      return instance.encodeSharedRef(object);
    }
    try {
      this.entered = true;
      this.reinitializeState();
      this.doEncode(object, 1);
      return this.bytes.subarray(0, this.pos);
    } finally {
      this.entered = false;
    }
  }
  encode(object) {
    if (this.entered) {
      const instance = this.clone();
      return instance.encode(object);
    }
    try {
      this.entered = true;
      this.reinitializeState();
      this.doEncode(object, 1);
      return this.bytes.slice(0, this.pos);
    } finally {
      this.entered = false;
    }
  }
  doEncode(object, depth) {
    if (depth > this.maxDepth) {
      throw new Error(`Too deep objects in depth ${depth}`);
    }
    if (object == null) {
      this.encodeNil();
    } else if (typeof object === "boolean") {
      this.encodeBoolean(object);
    } else if (typeof object === "number") {
      if (!this.forceIntegerToFloat) {
        this.encodeNumber(object);
      } else {
        this.encodeNumberAsFloat(object);
      }
    } else if (typeof object === "string") {
      this.encodeString(object);
    } else if (this.useBigInt64 && typeof object === "bigint") {
      this.encodeBigInt64(object);
    } else {
      this.encodeObject(object, depth);
    }
  }
  ensureBufferSizeToWrite(sizeToWrite) {
    const requiredSize = this.pos + sizeToWrite;
    if (this.view.byteLength < requiredSize) {
      this.resizeBuffer(requiredSize * 2);
    }
  }
  resizeBuffer(newSize) {
    const newBuffer = new ArrayBuffer(newSize);
    const newBytes = new Uint8Array(newBuffer);
    const newView = new DataView(newBuffer);
    newBytes.set(this.bytes);
    this.view = newView;
    this.bytes = newBytes;
  }
  encodeNil() {
    this.writeU8(192);
  }
  encodeBoolean(object) {
    if (object === false) {
      this.writeU8(194);
    } else {
      this.writeU8(195);
    }
  }
  encodeNumber(object) {
    if (!this.forceIntegerToFloat && Number.isSafeInteger(object)) {
      if (object >= 0) {
        if (object < 128) {
          this.writeU8(object);
        } else if (object < 256) {
          this.writeU8(204);
          this.writeU8(object);
        } else if (object < 65536) {
          this.writeU8(205);
          this.writeU16(object);
        } else if (object < 4294967296) {
          this.writeU8(206);
          this.writeU32(object);
        } else if (!this.useBigInt64) {
          this.writeU8(207);
          this.writeU64(object);
        } else {
          this.encodeNumberAsFloat(object);
        }
      } else {
        if (object >= -32) {
          this.writeU8(224 | object + 32);
        } else if (object >= -128) {
          this.writeU8(208);
          this.writeI8(object);
        } else if (object >= -32768) {
          this.writeU8(209);
          this.writeI16(object);
        } else if (object >= -2147483648) {
          this.writeU8(210);
          this.writeI32(object);
        } else if (!this.useBigInt64) {
          this.writeU8(211);
          this.writeI64(object);
        } else {
          this.encodeNumberAsFloat(object);
        }
      }
    } else {
      this.encodeNumberAsFloat(object);
    }
  }
  encodeNumberAsFloat(object) {
    if (this.forceFloat32) {
      this.writeU8(202);
      this.writeF32(object);
    } else {
      this.writeU8(203);
      this.writeF64(object);
    }
  }
  encodeBigInt64(object) {
    if (object >= BigInt(0)) {
      this.writeU8(207);
      this.writeBigUint64(object);
    } else {
      this.writeU8(211);
      this.writeBigInt64(object);
    }
  }
  writeStringHeader(byteLength) {
    if (byteLength < 32) {
      this.writeU8(160 + byteLength);
    } else if (byteLength < 256) {
      this.writeU8(217);
      this.writeU8(byteLength);
    } else if (byteLength < 65536) {
      this.writeU8(218);
      this.writeU16(byteLength);
    } else if (byteLength < 4294967296) {
      this.writeU8(219);
      this.writeU32(byteLength);
    } else {
      throw new Error(`Too long string: ${byteLength} bytes in UTF-8`);
    }
  }
  encodeString(object) {
    const maxHeaderSize = 1 + 4;
    const byteLength = utf8Count(object);
    this.ensureBufferSizeToWrite(maxHeaderSize + byteLength);
    this.writeStringHeader(byteLength);
    utf8Encode(object, this.bytes, this.pos);
    this.pos += byteLength;
  }
  encodeObject(object, depth) {
    const ext = this.extensionCodec.tryToEncode(object, this.context);
    if (ext != null) {
      this.encodeExtension(ext);
    } else if (Array.isArray(object)) {
      this.encodeArray(object, depth);
    } else if (ArrayBuffer.isView(object)) {
      this.encodeBinary(object);
    } else if (typeof object === "object") {
      this.encodeMap(object, depth);
    } else {
      throw new Error(`Unrecognized object: ${Object.prototype.toString.apply(object)}`);
    }
  }
  encodeBinary(object) {
    const size = object.byteLength;
    if (size < 256) {
      this.writeU8(196);
      this.writeU8(size);
    } else if (size < 65536) {
      this.writeU8(197);
      this.writeU16(size);
    } else if (size < 4294967296) {
      this.writeU8(198);
      this.writeU32(size);
    } else {
      throw new Error(`Too large binary: ${size}`);
    }
    const bytes = ensureUint8Array(object);
    this.writeU8a(bytes);
  }
  encodeArray(object, depth) {
    const size = object.length;
    if (size < 16) {
      this.writeU8(144 + size);
    } else if (size < 65536) {
      this.writeU8(220);
      this.writeU16(size);
    } else if (size < 4294967296) {
      this.writeU8(221);
      this.writeU32(size);
    } else {
      throw new Error(`Too large array: ${size}`);
    }
    for (const item of object) {
      this.doEncode(item, depth + 1);
    }
  }
  countWithoutUndefined(object, keys) {
    let count = 0;
    for (const key of keys) {
      if (object[key] !== undefined) {
        count++;
      }
    }
    return count;
  }
  encodeMap(object, depth) {
    const keys = Object.keys(object);
    if (this.sortKeys) {
      keys.sort();
    }
    const size = this.ignoreUndefined ? this.countWithoutUndefined(object, keys) : keys.length;
    if (size < 16) {
      this.writeU8(128 + size);
    } else if (size < 65536) {
      this.writeU8(222);
      this.writeU16(size);
    } else if (size < 4294967296) {
      this.writeU8(223);
      this.writeU32(size);
    } else {
      throw new Error(`Too large map object: ${size}`);
    }
    for (const key of keys) {
      const value = object[key];
      if (!(this.ignoreUndefined && value === undefined)) {
        this.encodeString(key);
        this.doEncode(value, depth + 1);
      }
    }
  }
  encodeExtension(ext) {
    if (typeof ext.data === "function") {
      const data = ext.data(this.pos + 6);
      const size2 = data.length;
      if (size2 >= 4294967296) {
        throw new Error(`Too large extension object: ${size2}`);
      }
      this.writeU8(201);
      this.writeU32(size2);
      this.writeI8(ext.type);
      this.writeU8a(data);
      return;
    }
    const size = ext.data.length;
    if (size === 1) {
      this.writeU8(212);
    } else if (size === 2) {
      this.writeU8(213);
    } else if (size === 4) {
      this.writeU8(214);
    } else if (size === 8) {
      this.writeU8(215);
    } else if (size === 16) {
      this.writeU8(216);
    } else if (size < 256) {
      this.writeU8(199);
      this.writeU8(size);
    } else if (size < 65536) {
      this.writeU8(200);
      this.writeU16(size);
    } else if (size < 4294967296) {
      this.writeU8(201);
      this.writeU32(size);
    } else {
      throw new Error(`Too large extension object: ${size}`);
    }
    this.writeI8(ext.type);
    this.writeU8a(ext.data);
  }
  writeU8(value) {
    this.ensureBufferSizeToWrite(1);
    this.view.setUint8(this.pos, value);
    this.pos++;
  }
  writeU8a(values) {
    const size = values.length;
    this.ensureBufferSizeToWrite(size);
    this.bytes.set(values, this.pos);
    this.pos += size;
  }
  writeI8(value) {
    this.ensureBufferSizeToWrite(1);
    this.view.setInt8(this.pos, value);
    this.pos++;
  }
  writeU16(value) {
    this.ensureBufferSizeToWrite(2);
    this.view.setUint16(this.pos, value);
    this.pos += 2;
  }
  writeI16(value) {
    this.ensureBufferSizeToWrite(2);
    this.view.setInt16(this.pos, value);
    this.pos += 2;
  }
  writeU32(value) {
    this.ensureBufferSizeToWrite(4);
    this.view.setUint32(this.pos, value);
    this.pos += 4;
  }
  writeI32(value) {
    this.ensureBufferSizeToWrite(4);
    this.view.setInt32(this.pos, value);
    this.pos += 4;
  }
  writeF32(value) {
    this.ensureBufferSizeToWrite(4);
    this.view.setFloat32(this.pos, value);
    this.pos += 4;
  }
  writeF64(value) {
    this.ensureBufferSizeToWrite(8);
    this.view.setFloat64(this.pos, value);
    this.pos += 8;
  }
  writeU64(value) {
    this.ensureBufferSizeToWrite(8);
    setUint64(this.view, this.pos, value);
    this.pos += 8;
  }
  writeI64(value) {
    this.ensureBufferSizeToWrite(8);
    setInt64(this.view, this.pos, value);
    this.pos += 8;
  }
  writeBigUint64(value) {
    this.ensureBufferSizeToWrite(8);
    this.view.setBigUint64(this.pos, value);
    this.pos += 8;
  }
  writeBigInt64(value) {
    this.ensureBufferSizeToWrite(8);
    this.view.setBigInt64(this.pos, value);
    this.pos += 8;
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/encode.mjs
function encode(value, options) {
  const encoder = new Encoder(options);
  return encoder.encodeSharedRef(value);
}

// client/node_modules/@msgpack/msgpack/dist.esm/utils/prettyByte.mjs
function prettyByte(byte) {
  return `${byte < 0 ? "-" : ""}0x${Math.abs(byte).toString(16).padStart(2, "0")}`;
}

// client/node_modules/@msgpack/msgpack/dist.esm/CachedKeyDecoder.mjs
var DEFAULT_MAX_KEY_LENGTH = 16;
var DEFAULT_MAX_LENGTH_PER_KEY = 16;

class CachedKeyDecoder {
  hit = 0;
  miss = 0;
  caches;
  maxKeyLength;
  maxLengthPerKey;
  constructor(maxKeyLength = DEFAULT_MAX_KEY_LENGTH, maxLengthPerKey = DEFAULT_MAX_LENGTH_PER_KEY) {
    this.maxKeyLength = maxKeyLength;
    this.maxLengthPerKey = maxLengthPerKey;
    this.caches = [];
    for (let i = 0;i < this.maxKeyLength; i++) {
      this.caches.push([]);
    }
  }
  canBeCached(byteLength) {
    return byteLength > 0 && byteLength <= this.maxKeyLength;
  }
  find(bytes, inputOffset, byteLength) {
    const records = this.caches[byteLength - 1];
    FIND_CHUNK:
      for (const record of records) {
        const recordBytes = record.bytes;
        for (let j = 0;j < byteLength; j++) {
          if (recordBytes[j] !== bytes[inputOffset + j]) {
            continue FIND_CHUNK;
          }
        }
        return record.str;
      }
    return null;
  }
  store(bytes, value) {
    const records = this.caches[bytes.length - 1];
    const record = { bytes, str: value };
    if (records.length >= this.maxLengthPerKey) {
      records[Math.random() * records.length | 0] = record;
    } else {
      records.push(record);
    }
  }
  decode(bytes, inputOffset, byteLength) {
    const cachedValue = this.find(bytes, inputOffset, byteLength);
    if (cachedValue != null) {
      this.hit++;
      return cachedValue;
    }
    this.miss++;
    const str = utf8DecodeJs(bytes, inputOffset, byteLength);
    const slicedCopyOfBytes = Uint8Array.prototype.slice.call(bytes, inputOffset, inputOffset + byteLength);
    this.store(slicedCopyOfBytes, str);
    return str;
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/Decoder.mjs
var STATE_ARRAY = "array";
var STATE_MAP_KEY = "map_key";
var STATE_MAP_VALUE = "map_value";
var mapKeyConverter = (key) => {
  if (typeof key === "string" || typeof key === "number") {
    return key;
  }
  throw new DecodeError("The type of key must be string or number but " + typeof key);
};

class StackPool {
  stack = [];
  stackHeadPosition = -1;
  get length() {
    return this.stackHeadPosition + 1;
  }
  top() {
    return this.stack[this.stackHeadPosition];
  }
  pushArrayState(size) {
    const state = this.getUninitializedStateFromPool();
    state.type = STATE_ARRAY;
    state.position = 0;
    state.size = size;
    state.array = new Array(size);
  }
  pushMapState(size) {
    const state = this.getUninitializedStateFromPool();
    state.type = STATE_MAP_KEY;
    state.readCount = 0;
    state.size = size;
    state.map = {};
  }
  getUninitializedStateFromPool() {
    this.stackHeadPosition++;
    if (this.stackHeadPosition === this.stack.length) {
      const partialState = {
        type: undefined,
        size: 0,
        array: undefined,
        position: 0,
        readCount: 0,
        map: undefined,
        key: null
      };
      this.stack.push(partialState);
    }
    return this.stack[this.stackHeadPosition];
  }
  release(state) {
    const topStackState = this.stack[this.stackHeadPosition];
    if (topStackState !== state) {
      throw new Error("Invalid stack state. Released state is not on top of the stack.");
    }
    if (state.type === STATE_ARRAY) {
      const partialState = state;
      partialState.size = 0;
      partialState.array = undefined;
      partialState.position = 0;
      partialState.type = undefined;
    }
    if (state.type === STATE_MAP_KEY || state.type === STATE_MAP_VALUE) {
      const partialState = state;
      partialState.size = 0;
      partialState.map = undefined;
      partialState.readCount = 0;
      partialState.type = undefined;
    }
    this.stackHeadPosition--;
  }
  reset() {
    this.stack.length = 0;
    this.stackHeadPosition = -1;
  }
}
var HEAD_BYTE_REQUIRED = -1;
var EMPTY_VIEW = new DataView(new ArrayBuffer(0));
var EMPTY_BYTES = new Uint8Array(EMPTY_VIEW.buffer);
try {
  EMPTY_VIEW.getInt8(0);
} catch (e) {
  if (!(e instanceof RangeError)) {
    throw new Error("This module is not supported in the current JavaScript engine because DataView does not throw RangeError on out-of-bounds access");
  }
}
var MORE_DATA = new RangeError("Insufficient data");
var sharedCachedKeyDecoder = new CachedKeyDecoder;

class Decoder {
  extensionCodec;
  context;
  useBigInt64;
  rawStrings;
  maxStrLength;
  maxBinLength;
  maxArrayLength;
  maxMapLength;
  maxExtLength;
  keyDecoder;
  mapKeyConverter;
  totalPos = 0;
  pos = 0;
  view = EMPTY_VIEW;
  bytes = EMPTY_BYTES;
  headByte = HEAD_BYTE_REQUIRED;
  stack = new StackPool;
  entered = false;
  constructor(options) {
    this.extensionCodec = options?.extensionCodec ?? ExtensionCodec.defaultCodec;
    this.context = options?.context;
    this.useBigInt64 = options?.useBigInt64 ?? false;
    this.rawStrings = options?.rawStrings ?? false;
    this.maxStrLength = options?.maxStrLength ?? UINT32_MAX;
    this.maxBinLength = options?.maxBinLength ?? UINT32_MAX;
    this.maxArrayLength = options?.maxArrayLength ?? UINT32_MAX;
    this.maxMapLength = options?.maxMapLength ?? UINT32_MAX;
    this.maxExtLength = options?.maxExtLength ?? UINT32_MAX;
    this.keyDecoder = options?.keyDecoder !== undefined ? options.keyDecoder : sharedCachedKeyDecoder;
    this.mapKeyConverter = options?.mapKeyConverter ?? mapKeyConverter;
  }
  clone() {
    return new Decoder({
      extensionCodec: this.extensionCodec,
      context: this.context,
      useBigInt64: this.useBigInt64,
      rawStrings: this.rawStrings,
      maxStrLength: this.maxStrLength,
      maxBinLength: this.maxBinLength,
      maxArrayLength: this.maxArrayLength,
      maxMapLength: this.maxMapLength,
      maxExtLength: this.maxExtLength,
      keyDecoder: this.keyDecoder
    });
  }
  reinitializeState() {
    this.totalPos = 0;
    this.headByte = HEAD_BYTE_REQUIRED;
    this.stack.reset();
  }
  setBuffer(buffer) {
    const bytes = ensureUint8Array(buffer);
    this.bytes = bytes;
    this.view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    this.pos = 0;
  }
  appendBuffer(buffer) {
    if (this.headByte === HEAD_BYTE_REQUIRED && !this.hasRemaining(1)) {
      this.setBuffer(buffer);
    } else {
      const remainingData = this.bytes.subarray(this.pos);
      const newData = ensureUint8Array(buffer);
      const newBuffer = new Uint8Array(remainingData.length + newData.length);
      newBuffer.set(remainingData);
      newBuffer.set(newData, remainingData.length);
      this.setBuffer(newBuffer);
    }
  }
  hasRemaining(size) {
    return this.view.byteLength - this.pos >= size;
  }
  createExtraByteError(posToShow) {
    const { view, pos } = this;
    return new RangeError(`Extra ${view.byteLength - pos} of ${view.byteLength} byte(s) found at buffer[${posToShow}]`);
  }
  decode(buffer) {
    if (this.entered) {
      const instance = this.clone();
      return instance.decode(buffer);
    }
    try {
      this.entered = true;
      this.reinitializeState();
      this.setBuffer(buffer);
      const object = this.doDecodeSync();
      if (this.hasRemaining(1)) {
        throw this.createExtraByteError(this.pos);
      }
      return object;
    } finally {
      this.entered = false;
    }
  }
  *decodeMulti(buffer) {
    if (this.entered) {
      const instance = this.clone();
      yield* instance.decodeMulti(buffer);
      return;
    }
    try {
      this.entered = true;
      this.reinitializeState();
      this.setBuffer(buffer);
      while (this.hasRemaining(1)) {
        yield this.doDecodeSync();
      }
    } finally {
      this.entered = false;
    }
  }
  async decodeAsync(stream) {
    if (this.entered) {
      const instance = this.clone();
      return instance.decodeAsync(stream);
    }
    try {
      this.entered = true;
      let decoded = false;
      let object;
      for await (const buffer of stream) {
        if (decoded) {
          this.entered = false;
          throw this.createExtraByteError(this.totalPos);
        }
        this.appendBuffer(buffer);
        try {
          object = this.doDecodeSync();
          decoded = true;
        } catch (e) {
          if (!(e instanceof RangeError)) {
            throw e;
          }
        }
        this.totalPos += this.pos;
      }
      if (decoded) {
        if (this.hasRemaining(1)) {
          throw this.createExtraByteError(this.totalPos);
        }
        return object;
      }
      const { headByte, pos, totalPos } = this;
      throw new RangeError(`Insufficient data in parsing ${prettyByte(headByte)} at ${totalPos} (${pos} in the current buffer)`);
    } finally {
      this.entered = false;
    }
  }
  decodeArrayStream(stream) {
    return this.decodeMultiAsync(stream, true);
  }
  decodeStream(stream) {
    return this.decodeMultiAsync(stream, false);
  }
  async* decodeMultiAsync(stream, isArray) {
    if (this.entered) {
      const instance = this.clone();
      yield* instance.decodeMultiAsync(stream, isArray);
      return;
    }
    try {
      this.entered = true;
      let isArrayHeaderRequired = isArray;
      let arrayItemsLeft = -1;
      for await (const buffer of stream) {
        if (isArray && arrayItemsLeft === 0) {
          throw this.createExtraByteError(this.totalPos);
        }
        this.appendBuffer(buffer);
        if (isArrayHeaderRequired) {
          arrayItemsLeft = this.readArraySize();
          isArrayHeaderRequired = false;
          this.complete();
        }
        try {
          while (true) {
            yield this.doDecodeSync();
            if (--arrayItemsLeft === 0) {
              break;
            }
          }
        } catch (e) {
          if (!(e instanceof RangeError)) {
            throw e;
          }
        }
        this.totalPos += this.pos;
      }
    } finally {
      this.entered = false;
    }
  }
  doDecodeSync() {
    DECODE:
      while (true) {
        const headByte = this.readHeadByte();
        let object;
        if (headByte >= 224) {
          object = headByte - 256;
        } else if (headByte < 192) {
          if (headByte < 128) {
            object = headByte;
          } else if (headByte < 144) {
            const size = headByte - 128;
            if (size !== 0) {
              this.pushMapState(size);
              this.complete();
              continue DECODE;
            } else {
              object = {};
            }
          } else if (headByte < 160) {
            const size = headByte - 144;
            if (size !== 0) {
              this.pushArrayState(size);
              this.complete();
              continue DECODE;
            } else {
              object = [];
            }
          } else {
            const byteLength = headByte - 160;
            object = this.decodeString(byteLength, 0);
          }
        } else if (headByte === 192) {
          object = null;
        } else if (headByte === 194) {
          object = false;
        } else if (headByte === 195) {
          object = true;
        } else if (headByte === 202) {
          object = this.readF32();
        } else if (headByte === 203) {
          object = this.readF64();
        } else if (headByte === 204) {
          object = this.readU8();
        } else if (headByte === 205) {
          object = this.readU16();
        } else if (headByte === 206) {
          object = this.readU32();
        } else if (headByte === 207) {
          if (this.useBigInt64) {
            object = this.readU64AsBigInt();
          } else {
            object = this.readU64();
          }
        } else if (headByte === 208) {
          object = this.readI8();
        } else if (headByte === 209) {
          object = this.readI16();
        } else if (headByte === 210) {
          object = this.readI32();
        } else if (headByte === 211) {
          if (this.useBigInt64) {
            object = this.readI64AsBigInt();
          } else {
            object = this.readI64();
          }
        } else if (headByte === 217) {
          const byteLength = this.lookU8();
          object = this.decodeString(byteLength, 1);
        } else if (headByte === 218) {
          const byteLength = this.lookU16();
          object = this.decodeString(byteLength, 2);
        } else if (headByte === 219) {
          const byteLength = this.lookU32();
          object = this.decodeString(byteLength, 4);
        } else if (headByte === 220) {
          const size = this.readU16();
          if (size !== 0) {
            this.pushArrayState(size);
            this.complete();
            continue DECODE;
          } else {
            object = [];
          }
        } else if (headByte === 221) {
          const size = this.readU32();
          if (size !== 0) {
            this.pushArrayState(size);
            this.complete();
            continue DECODE;
          } else {
            object = [];
          }
        } else if (headByte === 222) {
          const size = this.readU16();
          if (size !== 0) {
            this.pushMapState(size);
            this.complete();
            continue DECODE;
          } else {
            object = {};
          }
        } else if (headByte === 223) {
          const size = this.readU32();
          if (size !== 0) {
            this.pushMapState(size);
            this.complete();
            continue DECODE;
          } else {
            object = {};
          }
        } else if (headByte === 196) {
          const size = this.lookU8();
          object = this.decodeBinary(size, 1);
        } else if (headByte === 197) {
          const size = this.lookU16();
          object = this.decodeBinary(size, 2);
        } else if (headByte === 198) {
          const size = this.lookU32();
          object = this.decodeBinary(size, 4);
        } else if (headByte === 212) {
          object = this.decodeExtension(1, 0);
        } else if (headByte === 213) {
          object = this.decodeExtension(2, 0);
        } else if (headByte === 214) {
          object = this.decodeExtension(4, 0);
        } else if (headByte === 215) {
          object = this.decodeExtension(8, 0);
        } else if (headByte === 216) {
          object = this.decodeExtension(16, 0);
        } else if (headByte === 199) {
          const size = this.lookU8();
          object = this.decodeExtension(size, 1);
        } else if (headByte === 200) {
          const size = this.lookU16();
          object = this.decodeExtension(size, 2);
        } else if (headByte === 201) {
          const size = this.lookU32();
          object = this.decodeExtension(size, 4);
        } else {
          throw new DecodeError(`Unrecognized type byte: ${prettyByte(headByte)}`);
        }
        this.complete();
        const stack = this.stack;
        while (stack.length > 0) {
          const state = stack.top();
          if (state.type === STATE_ARRAY) {
            state.array[state.position] = object;
            state.position++;
            if (state.position === state.size) {
              object = state.array;
              stack.release(state);
            } else {
              continue DECODE;
            }
          } else if (state.type === STATE_MAP_KEY) {
            if (object === "__proto__") {
              throw new DecodeError("The key __proto__ is not allowed");
            }
            state.key = this.mapKeyConverter(object);
            state.type = STATE_MAP_VALUE;
            continue DECODE;
          } else {
            state.map[state.key] = object;
            state.readCount++;
            if (state.readCount === state.size) {
              object = state.map;
              stack.release(state);
            } else {
              state.key = null;
              state.type = STATE_MAP_KEY;
              continue DECODE;
            }
          }
        }
        return object;
      }
  }
  readHeadByte() {
    if (this.headByte === HEAD_BYTE_REQUIRED) {
      this.headByte = this.readU8();
    }
    return this.headByte;
  }
  complete() {
    this.headByte = HEAD_BYTE_REQUIRED;
  }
  readArraySize() {
    const headByte = this.readHeadByte();
    switch (headByte) {
      case 220:
        return this.readU16();
      case 221:
        return this.readU32();
      default: {
        if (headByte < 160) {
          return headByte - 144;
        } else {
          throw new DecodeError(`Unrecognized array type byte: ${prettyByte(headByte)}`);
        }
      }
    }
  }
  pushMapState(size) {
    if (size > this.maxMapLength) {
      throw new DecodeError(`Max length exceeded: map length (${size}) > maxMapLengthLength (${this.maxMapLength})`);
    }
    this.stack.pushMapState(size);
  }
  pushArrayState(size) {
    if (size > this.maxArrayLength) {
      throw new DecodeError(`Max length exceeded: array length (${size}) > maxArrayLength (${this.maxArrayLength})`);
    }
    this.stack.pushArrayState(size);
  }
  decodeString(byteLength, headerOffset) {
    if (!this.rawStrings || this.stateIsMapKey()) {
      return this.decodeUtf8String(byteLength, headerOffset);
    }
    return this.decodeBinary(byteLength, headerOffset);
  }
  decodeUtf8String(byteLength, headerOffset) {
    if (byteLength > this.maxStrLength) {
      throw new DecodeError(`Max length exceeded: UTF-8 byte length (${byteLength}) > maxStrLength (${this.maxStrLength})`);
    }
    if (this.bytes.byteLength < this.pos + headerOffset + byteLength) {
      throw MORE_DATA;
    }
    const offset = this.pos + headerOffset;
    let object;
    if (this.stateIsMapKey() && this.keyDecoder?.canBeCached(byteLength)) {
      object = this.keyDecoder.decode(this.bytes, offset, byteLength);
    } else {
      object = utf8Decode(this.bytes, offset, byteLength);
    }
    this.pos += headerOffset + byteLength;
    return object;
  }
  stateIsMapKey() {
    if (this.stack.length > 0) {
      const state = this.stack.top();
      return state.type === STATE_MAP_KEY;
    }
    return false;
  }
  decodeBinary(byteLength, headOffset) {
    if (byteLength > this.maxBinLength) {
      throw new DecodeError(`Max length exceeded: bin length (${byteLength}) > maxBinLength (${this.maxBinLength})`);
    }
    if (!this.hasRemaining(byteLength + headOffset)) {
      throw MORE_DATA;
    }
    const offset = this.pos + headOffset;
    const object = this.bytes.subarray(offset, offset + byteLength);
    this.pos += headOffset + byteLength;
    return object;
  }
  decodeExtension(size, headOffset) {
    if (size > this.maxExtLength) {
      throw new DecodeError(`Max length exceeded: ext length (${size}) > maxExtLength (${this.maxExtLength})`);
    }
    const extType = this.view.getInt8(this.pos + headOffset);
    const data = this.decodeBinary(size, headOffset + 1);
    return this.extensionCodec.decode(data, extType, this.context);
  }
  lookU8() {
    return this.view.getUint8(this.pos);
  }
  lookU16() {
    return this.view.getUint16(this.pos);
  }
  lookU32() {
    return this.view.getUint32(this.pos);
  }
  readU8() {
    const value = this.view.getUint8(this.pos);
    this.pos++;
    return value;
  }
  readI8() {
    const value = this.view.getInt8(this.pos);
    this.pos++;
    return value;
  }
  readU16() {
    const value = this.view.getUint16(this.pos);
    this.pos += 2;
    return value;
  }
  readI16() {
    const value = this.view.getInt16(this.pos);
    this.pos += 2;
    return value;
  }
  readU32() {
    const value = this.view.getUint32(this.pos);
    this.pos += 4;
    return value;
  }
  readI32() {
    const value = this.view.getInt32(this.pos);
    this.pos += 4;
    return value;
  }
  readU64() {
    const value = getUint64(this.view, this.pos);
    this.pos += 8;
    return value;
  }
  readI64() {
    const value = getInt64(this.view, this.pos);
    this.pos += 8;
    return value;
  }
  readU64AsBigInt() {
    const value = this.view.getBigUint64(this.pos);
    this.pos += 8;
    return value;
  }
  readI64AsBigInt() {
    const value = this.view.getBigInt64(this.pos);
    this.pos += 8;
    return value;
  }
  readF32() {
    const value = this.view.getFloat32(this.pos);
    this.pos += 4;
    return value;
  }
  readF64() {
    const value = this.view.getFloat64(this.pos);
    this.pos += 8;
    return value;
  }
}

// client/node_modules/@msgpack/msgpack/dist.esm/decode.mjs
function decode(buffer, options) {
  const decoder = new Decoder(options);
  return decoder.decode(buffer);
}

// client/src/websocket.ts
function validateMessage(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }
  const msg = raw;
  if (typeof msg.type !== "string") {
    return null;
  }
  const validated = { type: msg.type };
  if (typeof msg.componentId === "string")
    validated.componentId = msg.componentId;
  if (typeof msg.action === "string")
    validated.action = msg.action;
  if (typeof msg.key === "string")
    validated.key = msg.key;
  if (msg.value !== undefined)
    validated.value = msg.value;
  if (typeof msg.success === "boolean")
    validated.success = msg.success;
  if (msg.data !== undefined) {
    validated.data = msg.data;
  }
  if (msg.payload && typeof msg.payload === "object" && !Array.isArray(msg.payload)) {
    validated.payload = msg.payload;
  }
  if (msg.state && typeof msg.state === "object" && !Array.isArray(msg.state)) {
    validated.state = msg.state;
  }
  if (msg.diff && typeof msg.diff === "object" && !Array.isArray(msg.diff)) {
    validated.diff = msg.diff;
  }
  if (msg.patch && typeof msg.patch === "object" && !Array.isArray(msg.patch)) {
    validated.patch = msg.patch;
  }
  if (typeof msg.compressed === "boolean")
    validated.compressed = msg.compressed;
  if (typeof msg.error === "string")
    validated.error = msg.error;
  if (typeof msg.timestamp === "number")
    validated.timestamp = msg.timestamp;
  if (typeof msg.sessionToken === "string")
    validated.sessionToken = msg.sessionToken;
  if (typeof msg.clientId === "string")
    validated.clientId = msg.clientId;
  return validated;
}
var SESSION_COOKIE_KEY = "gospa_session";
function loadSession() {
  try {
    const saved = localStorage.getItem(SESSION_COOKIE_KEY);
    if (saved) {
      return JSON.parse(saved);
    }
  } catch (e) {
    console.warn("[GoSPA] Failed to load session:", e);
  }
  return null;
}
function saveSession(data) {
  try {
    localStorage.setItem(SESSION_COOKIE_KEY, JSON.stringify({ clientId: data.clientId }));
  } catch (e) {
    console.warn("[GoSPA] Failed to save session:", e);
  }
}
function clearSession() {
  try {
    localStorage.removeItem(SESSION_COOKIE_KEY);
  } catch (e) {
    console.warn("[GoSPA] Failed to clear session:", e);
  }
}

class WSClient {
  ws = null;
  config;
  reconnectAttempts = 0;
  heartbeatTimer = null;
  messageQueue = [];
  connectionState;
  pendingRequests = new Map;
  requestId = 0;
  sessionData = null;
  beforeUnloadHandler = null;
  constructor(config) {
    this.config = {
      reconnect: true,
      reconnectInterval: 1000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
      onOpen: () => {},
      onClose: () => {},
      onError: () => {},
      onConnectionFailed: () => {},
      onMessage: () => {},
      serializationFormat: "json",
      persistSession: false,
      persistQueueOnUnload: true,
      ...config
    };
    this.connectionState = new Rune("disconnected");
    this.sessionData = this.config.persistSession ? loadSession() : null;
    if (!this.config.persistSession) {
      clearSession();
    }
    if (this.config.persistQueueOnUnload) {
      try {
        const savedQueue = sessionStorage.getItem("gospa_ws_queue");
        if (savedQueue) {
          this.messageQueue = JSON.parse(savedQueue) || [];
          sessionStorage.removeItem("gospa_ws_queue");
        }
      } catch (e) {
        console.warn("[GoSPA] Failed to restore message queue:", e);
      }
    }
    this.beforeUnloadHandler = () => {
      if (!this.config.persistQueueOnUnload)
        return;
      if (this.messageQueue.length > 0) {
        try {
          sessionStorage.setItem("gospa_ws_queue", JSON.stringify(this.messageQueue));
        } catch (e) {
          console.warn("[GoSPA] Failed to persist message queue:", e);
        }
      }
    };
    window.addEventListener("beforeunload", this.beforeUnloadHandler);
  }
  get state() {
    return this.connectionState.get();
  }
  get isConnected() {
    return this.connectionState.get() === "connected";
  }
  stableConnectionTimer = null;
  reconnectTimer = null;
  connect() {
    return new Promise((resolve, reject) => {
      if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
        if (this.ws.readyState === WebSocket.OPEN) {
          resolve();
        } else {
          const check = setInterval(() => {
            if (this.ws?.readyState === WebSocket.OPEN) {
              clearInterval(check);
              resolve();
            } else if (!this.ws || this.ws.readyState === WebSocket.CLOSED) {
              clearInterval(check);
              reject(new Error("Connection failed"));
            }
          }, 100);
        }
        return;
      }
      this.connectionState.set("connecting");
      try {
        this.ws = new WebSocket(this.config.url);
        if (this.config.serializationFormat === "msgpack") {
          this.ws.binaryType = "arraybuffer";
        }
      } catch (error) {
        this.connectionState.set("disconnected");
        reject(error);
        return;
      }
      this.ws.onopen = () => {
        this.connectionState.set("connected");
        if (this.stableConnectionTimer)
          clearTimeout(this.stableConnectionTimer);
        this.stableConnectionTimer = setTimeout(() => {
          this.reconnectAttempts = 0;
          console.debug("[GoSPA] WebSocket connection stable, resetting backoff.");
        }, 5000);
        this.startHeartbeat();
        if (this.sessionData?.clientId) {
          const initMsg = {
            type: "init",
            clientId: this.sessionData.clientId
          };
          if (this.sessionData.token) {
            initMsg.sessionToken = this.sessionData.token;
          }
          this.send(initMsg);
        }
        this.flushMessageQueue();
        this.send({ type: "sync" });
        this.config.onOpen();
        resolve();
      };
      this.ws.onclose = (event) => {
        this.connectionState.set("disconnected");
        this.stopHeartbeat();
        if (this.stableConnectionTimer) {
          clearTimeout(this.stableConnectionTimer);
          this.stableConnectionTimer = null;
        }
        this.config.onClose(event);
        if (this.config.reconnect && this.reconnectAttempts < this.config.maxReconnectAttempts) {
          this.scheduleReconnect();
        } else if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
          this.config.onConnectionFailed(new Error("Max reconnect attempts reached"));
        }
      };
      this.ws.onerror = (error) => {
        this.config.onError(error);
        if (this.connectionState.get() === "connecting") {
          reject(new Error("WebSocket connection failed"));
        }
      };
      this.ws.onmessage = (event) => {
        this.handleMessage(event.data);
      };
    });
  }
  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.connectionState.set("disconnecting");
      this.stopHeartbeat();
      this.ws.close(1000, "Client disconnect");
      this.ws = null;
      this.connectionState.set("disconnected");
    }
    if (this.beforeUnloadHandler) {
      window.removeEventListener("beforeunload", this.beforeUnloadHandler);
      this.beforeUnloadHandler = null;
    }
  }
  scheduleReconnect() {
    if (this.reconnectTimer)
      return;
    this.reconnectAttempts++;
    const baseDelay = this.config.reconnectInterval;
    const expDelay = Math.min(baseDelay * Math.pow(2, this.reconnectAttempts - 1), 30000);
    const jitter = expDelay * 0.2 * (Math.random() * 2 - 1);
    const delay = Math.max(1000, expDelay + jitter);
    console.warn(`[GoSPA] WebSocket disconnected. Reconnecting in ${Math.round(delay)}ms (attempt ${this.reconnectAttempts})...`);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.connectionState.get() === "disconnected") {
        this.connect().catch(() => {});
      }
    }, delay);
  }
  startHeartbeat() {
    this.heartbeatTimer = setInterval(() => {
      this.send({ type: "ping" });
    }, this.config.heartbeatInterval);
  }
  stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }
  flushMessageQueue() {
    while (this.messageQueue.length > 0 && this.isConnected) {
      const message = this.messageQueue.shift();
      if (message) {
        this.send(message);
      }
    }
  }
  send(message) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      if (this.config.serializationFormat === "msgpack") {
        this.ws.send(encode(message));
      } else {
        this.ws.send(JSON.stringify(message));
      }
    } else {
      this.messageQueue.push(message);
    }
  }
  sendWithResponse(message) {
    return new Promise((resolve, reject) => {
      const id = `req_${++this.requestId}`;
      message.data = { ...message.data, _requestId: id };
      const timeout = setTimeout(() => {
        if (this.pendingRequests.has(id)) {
          this.pendingRequests.delete(id);
          reject(new Error("Request timeout"));
        }
      }, 30000);
      this.pendingRequests.set(id, {
        resolve,
        reject,
        timeout
      });
      this.send(message);
    });
  }
  async handleMessage(data) {
    try {
      let raw;
      if (this.config.serializationFormat === "msgpack" && (data instanceof ArrayBuffer || data instanceof Uint8Array)) {
        const buffer = data instanceof ArrayBuffer ? data : data.buffer;
        raw = decode(new Uint8Array(buffer));
      } else if (data instanceof Blob) {
        const buffer = await data.arrayBuffer();
        return this.handleMessage(buffer);
      } else {
        raw = typeof data === "string" ? JSON.parse(data) : data;
      }
      const message = validateMessage(raw);
      if (!message) {
        console.debug("[GoSPA] Received invalid WebSocket message, ignoring:", raw);
        return;
      }
      if (message.type === "compressed" && typeof message.data === "string") {
        try {
          const compressedData = Uint8Array.from(atob(message.data), (c) => c.charCodeAt(0));
          const ds = new DecompressionStream("gzip");
          const writer = ds.writable.getWriter();
          writer.write(compressedData);
          writer.close();
          const response = new Response(ds.readable);
          const decompressed = await response.arrayBuffer();
          return this.handleMessage(decompressed);
        } catch (err) {
          console.error("[GoSPA] Failed to decompress message:", err);
          return;
        }
      }
      if (message.type === "pong") {
        return;
      }
      if (message.type === "init" && message.sessionToken && message.clientId) {
        this.sessionData = {
          token: message.sessionToken,
          clientId: message.clientId
        };
        if (this.config.persistSession) {
          saveSession(this.sessionData);
        }
      }
      if (message.data?._responseId) {
        const id = message.data._responseId;
        const pending = this.pendingRequests.get(id);
        if (pending) {
          clearTimeout(pending.timeout);
          this.pendingRequests.delete(id);
          if (message.type === "error") {
            const rawError = message.error || "Unknown error";
            pending.reject(new Error(rawError));
          } else {
            pending.resolve(message.data);
          }
        }
      }
      this.config.onMessage(message);
    } catch (error) {
      console.error("[GoSPA] Failed to handle WebSocket message:", error);
    }
  }
  requestSync() {
    this.send({ type: "sync" });
  }
  sendAction(action, payload = {}) {
    this.send({
      type: "action",
      action,
      payload
    });
  }
  requestState(componentId) {
    return this.sendWithResponse({
      type: "init",
      componentId
    });
  }
}
function sendAction(action, payload = {}) {
  if (clientInstance) {
    clientInstance.sendAction(action, payload);
  } else {
    console.warn("[GoSPA] Cannot send action: WebSocket not initialized");
  }
}
var clientInstance = null;
function getWebSocketClient() {
  return clientInstance;
}
function initWebSocket(config) {
  if (clientInstance) {
    clientInstance.disconnect();
  }
  clientInstance = new WSClient(config);
  return clientInstance;
}
function syncedRune(initial, options) {
  const rune2 = new Rune(initial);
  const ws = options.ws || clientInstance;
  let isReverting = false;
  const originalSet = rune2.set.bind(rune2);
  rune2.set = (newValue) => {
    if (isReverting) {
      originalSet(newValue);
      return;
    }
    const backupValue = rune2.get();
    originalSet(newValue);
    if (ws?.isConnected) {
      try {
        const executeSync = () => {
          ws.send({
            type: "update",
            payload: { key: options.key, value: newValue }
          });
        };
        if (options.debounce) {
          setTimeout(executeSync, options.debounce);
        } else {
          executeSync();
        }
      } catch (e) {
        console.warn("[GoSPA] Optimistic update failed, rolling back.", e);
        isReverting = true;
        originalSet(backupValue);
        isReverting = false;
      }
    } else {
      console.warn("[GoSPA] WS disconnected, optimistic update rolled back.");
      isReverting = true;
      originalSet(backupValue);
      isReverting = false;
    }
  };
  return rune2;
}
function syncBatch(componentId, states, ws) {
  const client = ws || clientInstance;
  if (!client?.isConnected)
    return;
  for (const [key, rune2] of Object.entries(states)) {
    client.send({
      type: "update",
      payload: { key, value: rune2.get() }
    });
  }
}
function applyStateUpdate(states, data) {
  batch(() => {
    for (const [key, value] of Object.entries(data)) {
      const rune2 = states[key];
      if (rune2) {
        rune2.set(value);
      }
    }
  });
}
export {
  syncedRune,
  syncBatch,
  sendAction,
  initWebSocket,
  getWebSocketClient,
  applyStateUpdate,
  WSClient
};

export { createDevToolsPanel, updateDevToolsPanel, toggleDevTools, isDev, inspect, timing, memoryUsage, debugLog, createInspector, getCurrentEffect, batch, Rune, rune, Derived, derived, Effect, effect, watch, StateMap, untrack, preEffect, WSClient, sendAction, getWebSocketClient, initWebSocket, syncedRune, syncBatch, applyStateUpdate };
