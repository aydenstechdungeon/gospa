export interface Notifier {
  notify(): void;
}

export let batchDepth = 0;
export const pendingNotifications: Set<Notifier> = new Set();
let autoBatchScheduled = false;

/**
 * Schedule automatic batch flush for the next microtask.
 * This enables auto-batching of rapid synchronous state updates
 * without requiring manual batch() calls.
 */
export function scheduleAutoBatch(): void {
  if (autoBatchScheduled || batchDepth > 0) return;
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

/**
 * Batch state updates to prevent redundant re-computations and re-renders.
 */
export function batch(fn: () => void): void {
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
