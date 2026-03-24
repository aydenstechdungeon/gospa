// Debug utilities - tree-shaken in production builds
// This module is lazy-loaded only when debug features are used

import { Effect } from "./state.ts";

export type InspectType = "init" | "update";

// === DevTools Inspector Panel ===
let devToolsPanel: HTMLElement | null = null;
let devToolsInitialized = false;

/**
 * Create a visual DevTools inspector panel.
 * Shows component tree, state, and performance metrics.
 * Only available in development mode.
 */
export function createDevToolsPanel(): void {
  if (!isDev() || devToolsInitialized) return;
  devToolsInitialized = true;

  // Create panel container
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

  // Setup close button
  const closeBtn = devToolsPanel.querySelector("#gospa-devtools-close");
  closeBtn?.addEventListener("click", () => {
    devToolsPanel?.remove();
    devToolsPanel = null;
    devToolsInitialized = false;
  });

  // Setup tab switching
  const tabs = devToolsPanel.querySelectorAll("#gospa-devtools-tabs button");
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      tabs.forEach((t) => t.classList.remove("active"));
      tab.classList.add("active");

      const tabName = tab.getAttribute("data-tab");
      const contents = devToolsPanel?.querySelectorAll(
        ".gospa-devtools-tab-content",
      );
      contents?.forEach((content) => {
        (content as HTMLElement).style.display =
          content.id === `gospa-devtools-${tabName}` ? "block" : "none";
      });
    });
  });

  // Make panel draggable
  const header = devToolsPanel.querySelector("#gospa-devtools-header");
  let isDragging = false;
  let dragOffsetX = 0;
  let dragOffsetY = 0;

  header?.addEventListener("mousedown", (e: Event) => {
    const mouseEvent = e as MouseEvent;
    isDragging = true;
    dragOffsetX = mouseEvent.clientX - (devToolsPanel?.offsetLeft || 0);
    dragOffsetY = mouseEvent.clientY - (devToolsPanel?.offsetTop || 0);
  });

  document.addEventListener("mousemove", (e: Event) => {
    if (isDragging && devToolsPanel) {
      const mouseEvent = e as MouseEvent;
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

/**
 * Update DevTools panel with current state
 */
export function updateDevToolsPanel(): void {
  if (!devToolsPanel || !isDev()) return;

  // Update components tab
  const componentsContent = devToolsPanel.querySelector(
    "#gospa-devtools-components",
  );
  if (componentsContent) {
    const components = (window as any).__GOSPA__?.components;
    if (components) {
      let html = '<div class="gospa-devtools-section">';
      html += '<div class="gospa-devtools-section-title">Components</div>';
      for (const [id, component] of components) {
        const stateKeys = component.states
          ? Array.from(component.states.keys())
          : [];
        html += `<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${id}</span>
					<span class="gospa-devtools-value">(${stateKeys.length} states)</span>
				</div>`;
      }
      html += "</div>";
      componentsContent.innerHTML = html;
    }
  }

  // Update state tab
  const stateContent = devToolsPanel.querySelector("#gospa-devtools-state");
  if (stateContent) {
    const globalState = (window as any).__GOSPA__?.globalState;
    if (globalState) {
      let html = '<div class="gospa-devtools-section">';
      html += '<div class="gospa-devtools-section-title">Global State</div>';
      const stateObj = globalState.toJSON ? globalState.toJSON() : {};
      for (const [key, value] of Object.entries(stateObj)) {
        const valueStr =
          typeof value === "object" ? JSON.stringify(value) : String(value);
        html += `<div class="gospa-devtools-item">
					<span class="gospa-devtools-key">${key}:</span>
					<span class="gospa-devtools-value">${valueStr}</span>
				</div>`;
      }
      html += "</div>";

      // Add Global Stores
      const stores = (window as any).__GOSPA_STORES__;
      if (stores) {
        html += '<div class="gospa-devtools-section">';
        html +=
          '<div class="gospa-devtools-section-title">Reactive Stores</div>';
        for (const [name, store] of Object.entries(stores)) {
          const valueStr =
            typeof store === "object" ? JSON.stringify(store) : String(store);
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

  // Update performance tab
  const perfContent = devToolsPanel.querySelector(
    "#gospa-devtools-performance",
  );
  if (perfContent) {
    let html = '<div class="gospa-devtools-section">';
    html +=
      '<div class="gospa-devtools-section-title">Performance Metrics</div>';

    // Memory usage
    if ("memory" in performance && (performance as any).memory) {
      const memory = (performance as any).memory;
      const usedMB = (memory.usedJSHeapSize / 1024 / 1024).toFixed(2);
      const totalMB = (memory.totalJSHeapSize / 1024 / 1024).toFixed(2);
      html += `<div class="gospa-devtools-metric">
				<span class="gospa-devtools-metric-label">Heap Used</span>
				<span class="gospa-devtools-metric-value">${usedMB}MB / ${totalMB}MB</span>
			</div>`;
    }

    // Timing
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

/**
 * Toggle DevTools panel visibility
 */
export function toggleDevTools(): void {
  if (!isDev()) return;

  if (devToolsPanel) {
    devToolsPanel.remove();
    devToolsPanel = null;
    devToolsInitialized = false;
  } else {
    createDevToolsPanel();
  }
}

/**
 * Check if running in development mode
 */
export function isDev(): boolean {
  return (
    typeof window !== "undefined" &&
    (window as unknown as { __GOSPA_DEV__?: boolean }).__GOSPA_DEV__ !== false
  );
}

/**
 * $inspect - Debug helper for observing state changes (dev only).
 * In production, this becomes a no-op.
 */
export function inspect<T>(...values: (() => T)[] | T[]): {
  with: (callback: (type: InspectType, value: T[]) => void) => void;
} {
  if (!isDev()) {
    return { with: () => {} };
  }

  let firstRun = true;
  const callbacks: Array<(type: InspectType, value: T[]) => void> = [];

  // Log initial values
  const getValues = (): T[] =>
    values.map((v) => (typeof v === "function" ? (v as () => T)() : v));

  const logValues = (type: InspectType) => {
    const currentValues = getValues();
    console.log(`%c[${type}]`, "color: #888", ...currentValues);
    callbacks.forEach((cb) => cb(type, currentValues));
  };

  // Set up effect to track changes
  new Effect(() => {
    // Read all values to track them
    getValues();

    if (firstRun) {
      firstRun = false;
      logValues("init");
    } else {
      logValues("update");
    }
  });

  return {
    with: (callback: (type: InspectType, value: T[]) => void) => {
      callbacks.push(callback);
    },
  };
}

/**
 * $inspect.trace - Log which dependencies triggered an effect.
 */
inspect.trace = (label?: string) => {
  if (!isDev()) return;

  console.log(
    `%c[trace]${label ? ` ${label}` : ""}`,
    "color: #666; font-style: italic",
  );
};

/**
 * Performance timing helper for development
 */
export function timing(name: string) {
  if (!isDev()) {
    return { end: () => {} };
  }

  const start = performance.now();
  return {
    end: () => {
      const duration = performance.now() - start;
      console.log(
        `%c[timing] ${name}: ${duration.toFixed(2)}ms`,
        "color: #0a0",
      );
    },
  };
}

/**
 * Memory usage helper for development
 */
export function memoryUsage(label: string) {
  if (!isDev()) return;

  if (
    "memory" in performance &&
    (performance as unknown as { memory?: { usedJSHeapSize: number } }).memory
  ) {
    const memory = (
      performance as unknown as { memory: { usedJSHeapSize: number } }
    ).memory;
    const mb = (memory.usedJSHeapSize / 1024 / 1024).toFixed(2);
    console.log(`%c[memory] ${label}: ${mb}MB`, "color: #a0a");
  }
}

/**
 * Debug logger that only logs in development
 */
export function debugLog(...args: unknown[]): void {
  if (!isDev()) return;
  console.log("%c[debug]", "color: #888", ...args);
}

/**
 * Create a debug inspector for reactive state
 */
export function createInspector<T>(
  name: string,
  state: { get: () => T; subscribe: (fn: (v: T) => void) => () => void },
) {
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
    },
  };
}
