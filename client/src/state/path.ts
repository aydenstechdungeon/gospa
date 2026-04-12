import { type Rune, type Unsubscribe } from "./rune.ts";
import { Derived } from "./derived.ts";

/**
 * Get a nested property value by dot-separated path.
 */
function getByPath<T>(obj: T, path: string): unknown {
  const parts = path.split(".");
  let current: unknown = obj;

  for (const part of parts) {
    if (current === null || current === undefined) return undefined;
    if (typeof current !== "object") return undefined;
    current = (current as Record<string, unknown>)[part];
  }

  return current;
}

/**
 * Watch a specific path in an object for changes.
 */
export function watchPath<T extends object>(
  obj: Rune<T>,
  path: string,
  callback: (value: unknown, oldValue: unknown) => void,
): Unsubscribe {
  let oldValue: unknown;

  return obj.subscribe((current) => {
    const newValue = getByPath(current, path);
    if (newValue !== oldValue) {
      callback(newValue, oldValue);
      oldValue = newValue;
    }
  });
}

/**
 * Create a derived value from a specific path in a reactive object.
 */
export function derivedPath<T extends object>(
  obj: Rune<T>,
  path: string,
): Derived<unknown> {
  return new Derived(() => getByPath(obj.value, path));
}
