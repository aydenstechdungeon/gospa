type HookType = "mounted" | "updated" | "destroyed";
type Callback = () => void;

const hooks = new Map<HookType, Set<Callback>>([
  ["mounted", new Set()],
  ["updated", new Set()],
  ["destroyed", new Set()],
]);

/**
 * Register a callback to run when the current component is mounted.
 */
export function onMounted(callback: Callback): void {
  hooks.get("mounted")?.add(callback);
}

/**
 * Register a callback to run when the current component is updated.
 */
export function onUpdated(callback: Callback): void {
  hooks.get("updated")?.add(callback);
}

/**
 * Register a callback to run when the current component is destroyed.
 */
export function onDestroyed(callback: Callback): void {
  hooks.get("destroyed")?.add(callback);
}

/**
 * Declare global debug constant for build-time stripping.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

/**
 * Internal: Run all hooks of a specific type.
 */
export function runHooks(type: HookType): void {
  const callbacks = hooks.get(type);
  if (callbacks) {
    callbacks.forEach((cb) => {
      try {
        cb();
      } catch (e) {
        if (typeof GOSPA_DEBUG !== "undefined" && GOSPA_DEBUG) {
          console.error(`[GoSPA] Error in ${type} hook:`, e);
        }
      }
    });

    // Clear hooks after running if they are lifecycle-ending
    if (type === "destroyed") {
      callbacks.clear();
    }
  }
}
