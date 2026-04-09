export interface Notifier {
  notify(): void;
}

export let batchDepth = 0;
export const pendingNotifications: Notifier[] = [];
const pendingSet = new Set<Notifier>();
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
    if (batchDepth === 0 && pendingNotifications.length > 0) {
      flushPending();
    }
  });
}

/**
 * Add a notifier to the current batch.
 */
export function addToBatch(n: Notifier): void {
  if (!pendingSet.has(n)) {
    pendingSet.add(n);
    pendingNotifications.push(n);
  }
}

function flushPending(): void {
  for (let i = 0; i < pendingNotifications.length; i++) {
    pendingNotifications[i].notify();
  }
  pendingNotifications.length = 0;
  pendingSet.clear();
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
      flushPending();
    }
  }
}
