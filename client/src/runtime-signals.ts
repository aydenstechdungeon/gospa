/**
 * Lightweight debug/event signaling for runtime observability.
 * Signals are emitted only when debug mode is enabled.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

export function emitRuntimeSignal<T = unknown>(type: string, detail?: T): void {
  if (!(typeof GOSPA_DEBUG !== "undefined" && GOSPA_DEBUG)) return;
  if (typeof window === "undefined") return;

  try {
    window.dispatchEvent(new CustomEvent(type, { detail }));
  } catch {
    // Ignore custom-event failures in unsupported environments.
  }
}
