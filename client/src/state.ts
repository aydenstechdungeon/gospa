// Shim for backward compatibility with monolithic state.ts
export * from "./state/rune.ts";
export * from "./state/derived.ts";
export * from "./state/effect.ts";
export * from "./state/batch.ts";
export * from "./state/map.ts";
export * from "./state/disposal.ts";
export * from "./state/effect.ts";
export * from "./state/resource.ts";
export * from "./state/root.ts";
export * from "./state/snapshot.ts";
export * from "./state/path.ts";
export { fastDeepEqual } from "./state/equality.ts";

/**
 * tracking - Check if currently inside a reactive tracking context.
 */
import { currentEffect } from "./state/effect.ts";
export function tracking(): boolean {
  return currentEffect !== null;
}

// === Debug utilities - lazy loaded from separate module ===
export {
  inspect,
  isDev,
  timing,
  memoryUsage,
  debugLog,
  createInspector,
} from "./debug.ts";
export type { InspectType } from "./debug.ts";
