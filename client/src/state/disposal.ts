/**
 * Declare global debug constant for build-time stripping.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

/**
 * Disposable interface for reactive primitives that need cleanup.
 */
export interface Disposable {
  dispose(): void;
  isDisposed(): boolean;
}

/**
 * Global registry for tracking active reactive primitives.
 * Useful for debugging memory leaks in development.
 */
const activeDisposables: Set<{ deref: () => Disposable | undefined }> =
  new Set();

// FinalizationRegistry for automatic cleanup notification (ES2021+)
let finalizationRegistry: {
  register(target: object, heldValue: string): void;
  unregister(target: object): void;
} | null = null;

// Check for FinalizationRegistry availability at runtime
if (
  typeof (globalThis as unknown as { FinalizationRegistry?: unknown })
    .FinalizationRegistry !== "undefined"
) {
  finalizationRegistry = new (
    globalThis as unknown as {
      FinalizationRegistry: new <T>(cb: (heldValue: T) => void) => {
        register(target: object, heldValue: string): void;
        unregister(target: object): void;
      };
    }
  ).FinalizationRegistry<string>((_id) => {
    // Called when object is GC'd - could log in dev mode
  });
}

let disposalTrackingEnabled = false;

/**
 * Enable disposal tracking for debugging memory leaks.
 * In production, this should be disabled.
 */
export function enableDisposalTracking(enabled: boolean = true): void {
  disposalTrackingEnabled = enabled;
}

/**
 * Get count of tracked disposables (for debugging).
 * Note: Some may have been garbage collected.
 */
export function getActiveDisposableCount(): number {
  let count = 0;
  for (const ref of activeDisposables) {
    if (ref.deref()) count++;
  }
  return count;
}

/**
 * Force dispose all tracked disposables (for testing/cleanup).
 */
export function disposeAll(): void {
  for (const ref of activeDisposables) {
    const disposable = ref.deref();
    if (disposable && !disposable.isDisposed()) {
      disposable.dispose();
    }
  }
  activeDisposables.clear();
}

/**
 * Track a disposable object for debugging memory leaks.
 */
export function trackDisposable<T extends Disposable>(disposable: T): T {
  if (typeof GOSPA_DEBUG !== "undefined" && GOSPA_DEBUG) {
    if (disposalTrackingEnabled) {
      // Use WeakRef if available (ES2021+)
      if (
        typeof (globalThis as unknown as { WeakRef?: unknown }).WeakRef !==
        "undefined"
      ) {
        const WeakRefCtor = (
          globalThis as unknown as {
            WeakRef: new <T extends object>(
              target: T,
            ) => { deref: () => T | undefined };
          }
        ).WeakRef;
        activeDisposables.add(new WeakRefCtor(disposable));
      } else {
        // Fallback: store directly (not ideal but works)
        activeDisposables.add({ deref: () => disposable });
      }
      if (finalizationRegistry) {
        finalizationRegistry.register(disposable, `disposable-${Date.now()}`);
      }
    }
  }
  return disposable;
}
