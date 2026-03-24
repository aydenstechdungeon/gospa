// GoSPA Signal-based Reactivity System
// Proxy-based auto-tracking for ergonomic state management
// Inspired by Svelte 5's $state and Solid's createSignal

import {
  Rune,
  Derived,
  Effect,
  batch,
  type Unsubscribe,
  untrack,
  getCurrentEffect,
} from "./state.ts";

// Symbol to identify reactive proxies
const REACTIVE_SYMBOL = Symbol("gospa-reactive");
const RAW_SYMBOL = Symbol("gospa-raw");

/**
 * Create a reactive proxy that auto-tracks property access.
 * Similar to Svelte 5's $state or Solid's createStore.
 *
 * @example
 * ```typescript
 * const state = reactive({ count: 0, name: 'World' });
 *
 * // Auto-tracks in effects
 * effect(() => {
 *   console.log(state.count); // Re-runs when count changes
 * });
 *
 * // Direct mutation triggers updates
 * state.count = 1;
 * ```
 */
export function reactive<T extends object>(initial: T): T {
  // Return existing reactive proxy if already reactive
  if (initial && (initial as any)[REACTIVE_SYMBOL]) {
    return initial;
  }

  // Store raw values in a separate map
  const rawValues = new Map<string | symbol, unknown>();
  const runes = new Map<string | symbol, Rune<unknown>>();
  const subscribers = new Map<string | symbol, Set<() => void>>();

  // Initialize raw values
  for (const key of Object.keys(initial)) {
    rawValues.set(key, (initial as any)[key]);
    runes.set(key, new Rune((initial as any)[key]));
  }

  const handler: ProxyHandler<T> = {
    get(target, prop, receiver) {
      // Internal symbols
      if (prop === REACTIVE_SYMBOL) return true;
      if (prop === RAW_SYMBOL) return Object.fromEntries(rawValues);

      // Track dependency if inside an effect
      const currentEffect = getCurrentEffect();
      if (currentEffect) {
        const rune = runes.get(prop);
        if (rune) {
          currentEffect.addDependency(rune as unknown as Rune<unknown>);
        }
      }

      // Return the reactive value
      const rune = runes.get(prop);
      if (rune) {
        return rune.get();
      }

      // Handle array methods and other built-ins
      const value = Reflect.get(target, prop, receiver);
      if (typeof value === "function") {
        return value.bind(receiver);
      }

      return value;
    },

    set(target, prop, value, receiver) {
      // Don't allow setting internal symbols
      if (prop === REACTIVE_SYMBOL || prop === RAW_SYMBOL) {
        return false;
      }

      const oldValue = rawValues.get(prop);

      // Skip if value hasn't changed
      if (Object.is(oldValue, value)) {
        return true;
      }

      // Update raw value
      rawValues.set(prop, value);

      // Update or create rune
      let rune = runes.get(prop);
      if (rune) {
        rune.set(value);
      } else {
        rune = new Rune(value);
        runes.set(prop, rune);
      }

      // Notify subscribers
      const propSubscribers = subscribers.get(prop);
      if (propSubscribers) {
        batch(() => {
          propSubscribers.forEach((fn) => fn());
        });
      }

      return true;
    },

    has(target, prop) {
      if (prop === REACTIVE_SYMBOL || prop === RAW_SYMBOL) {
        return true;
      }
      return rawValues.has(prop) || Reflect.has(target, prop);
    },

    ownKeys(target) {
      return Array.from(rawValues.keys()).filter(
        (k) => typeof k === "string",
      ) as string[];
    },

    getOwnPropertyDescriptor(target, prop) {
      if (rawValues.has(prop)) {
        return {
          enumerable: true,
          configurable: true,
          value: rawValues.get(prop),
        };
      }
      return Reflect.getOwnPropertyDescriptor(target, prop);
    },
  };

  const proxy = new Proxy(initial, handler);

  return proxy;
}

/**
 * $state - Create reactive local state.
 * Works with both objects (deeply reactive) and primitives (boxed).
 *
 * @example
 * ```typescript
 * const count = $state(0);
 * count.value++; // Access value via .value
 *
 * const user = $state({ name: "Alice" });
 * user.name = "Bob"; // Direct property access
 * ```
 */
export function $state<T>(initial: T): T {
  if (typeof initial === "object" && initial !== null) {
    return reactive(initial as object) as T;
  }
  // For primitives, we return a Rune wrapper that behaves like a box
  // The compiler will handle the .value access if needed, or the user can use .value
  return new Rune(initial) as unknown as T;
}

/**
 * Create a derived value that auto-tracks dependencies.
 * Similar to Svelte 5's $derived or Solid's createMemo.
 *
 * @example
 * ```typescript
 * const state = reactive({ count: 0 });
 * const doubled = derived(() => state.count * 2);
 *
 * console.log(doubled()); // 0
 * state.count = 5;
 * console.log(doubled()); // 10
 * ```
 */
export function derived<T>(compute: () => T): () => T {
  const derivedInstance = new Derived(compute);
  return () => derivedInstance.get();
}

/**
 * $derived - Create a derived reactive value.
 * similar to Svelte 5's $derived.
 */
export function $derived<T>(compute: () => T): () => T {
  return derived(compute);
}

/**
 * Create an effect that auto-tracks reactive dependencies.
 * Similar to Svelte 5's $effect or Solid's createEffect.
 *
 * @example
 * ```typescript
 * const state = reactive({ count: 0 });
 *
 * effect(() => {
 *   console.log('Count changed:', state.count);
 * });
 *
 * state.count = 1; // Logs: "Count changed: 1"
 * ```
 */
export function effect(fn: () => void | (() => void)): () => void {
  const effectInstance = new Effect(fn);
  return () => effectInstance.dispose();
}

/**
 * $effect - Create a side effect that auto-tracks dependencies.
 */
export function $effect(fn: () => void | (() => void)): () => void {
  return effect(fn);
}

/**
 * Watch specific properties of a reactive object.
 *
 * @example
 * ```typescript
 * const state = reactive({ count: 0, name: 'World' });
 *
 * watchProp(state, 'count', (newVal, oldVal) => {
 *   console.log('Count changed:', oldVal, '->', newVal);
 * });
 * ```
 */
export function watchProp<T extends object, K extends keyof T>(
  obj: T,
  prop: K,
  callback: (newValue: T[K], oldValue: T[K]) => void,
): Unsubscribe {
  if (!(obj as any)[REACTIVE_SYMBOL]) {
    throw new Error(
      "watchProp requires a reactive object created with reactive()",
    );
  }

  // Create a derived that tracks the specific property
  const derivedProp = new Derived(() => (obj as T)[prop]);

  // Subscribe to changes
  return derivedProp.subscribe((newVal, oldVal) => {
    callback(newVal as T[K], oldVal as T[K] | undefined as any);
  });
}

/**
 * Get the raw (non-reactive) value of a reactive object.
 * Useful for serialization or when you need to bypass reactivity.
 */
export function toRaw<T extends object>(obj: T): T {
  if (!(obj as any)[REACTIVE_SYMBOL]) {
    return obj;
  }
  return (obj as any)[RAW_SYMBOL] as T;
}

/**
 * Check if an object is reactive.
 */
export function isReactive(obj: unknown): boolean {
  return (
    obj != null &&
    typeof obj === "object" &&
    (obj as any)[REACTIVE_SYMBOL] === true
  );
}

/**
 * Create a reactive array with auto-tracking.
 * Array mutations (push, pop, splice, etc.) trigger updates.
 *
 * Note: Due to JavaScript proxy limitations, direct array mutations like
 * push/pop/splice may not reliably trigger reactivity. For best results,
 * reassign the array: `items = [...items, newItem]` instead of `items.push()`.
 *
 * @example
 * ```typescript
 * const items = reactiveArray([1, 2, 3]);
 *
 * effect(() => {
 *   console.log('Items:', items.length);
 * });
 *
 * // Recommended: reassign to trigger reactivity
 * items = [...items, 4];
 * ```
 */
export function reactiveArray<T>(initial: T[]): T[] {
  const proxy = reactive(initial as any) as T[];

  // Wrap array methods to trigger updates
  const arrayMethods = [
    "push",
    "pop",
    "shift",
    "unshift",
    "splice",
    "sort",
    "reverse",
  ];
  for (const method of arrayMethods) {
    const original = (Array.prototype as any)[method];
    (proxy as any)[method] = function (...args: any[]) {
      const result = original.apply(this, args);
      // Trigger update by setting a dummy property
      (proxy as any).__version = Date.now();
      return result;
    };
  }

  return proxy;
}

// Re-export state primitives for convenience
export { Rune, Derived, Effect, batch, untrack, type Unsubscribe };
