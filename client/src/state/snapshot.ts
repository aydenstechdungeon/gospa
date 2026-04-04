import { Rune, RuneRaw } from "./rune.ts";

/**
 * snapshot - Create a non-reactive plain copy of a value.
 * Works with Rune, RuneRaw, or plain values.
 */
export function snapshot<T>(value: T | Rune<T> | RuneRaw<T>): T {
  if (value instanceof RuneRaw) {
    return value.snapshot();
  }
  if (value instanceof Rune) {
    const val = value.peek();
    if (typeof val === "object" && val !== null) {
      if (Array.isArray(val)) return [...val] as T;
      return { ...val } as T;
    }
    return val;
  }
  // Plain value - return as-is or shallow copy
  if (typeof value === "object" && value !== null) {
    if (Array.isArray(value)) return [...value] as T;
    return { ...value } as T;
  }
  return value;
}
