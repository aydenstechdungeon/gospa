import { $state, $derived, $effect } from "./signals.ts";
import { Rune, StateMap } from "./state.ts";

/**
 * SharedStore handles global reactive state that can be shared across islands.
 */
export class SharedStore {
  private static instance: SharedStore;
  private stores: Map<string, any> = new Map();

  private constructor() {}

  static getInstance(): SharedStore {
    if (!SharedStore.instance) {
      SharedStore.instance = new SharedStore();
    }
    return SharedStore.instance;
  }

  /**
   * Create or get a named reactive store.
   */
  create<T>(name: string, initialValue: T): T {
    if (this.stores.has(name)) {
      return this.stores.get(name);
    }

    const store = $state(initialValue);
    this.stores.set(name, store);
    
    // DevTools integration
    this.updateDevTools();

    return store as T;
  }

  /**
   * Get an existing named store.
   */
  get<T>(name: string): T | undefined {
    return this.stores.get(name);
  }

  /**
   * Check if a store exists.
   */
  has(name: string): boolean {
    return this.stores.has(name);
  }

  /**
   * List all store names.
   */
  list(): string[] {
    return Array.from(this.stores.keys());
  }

  /**
   * Update DevTools global state.
   */
  private updateDevTools() {
    if (typeof window !== "undefined") {
      (window as any).__GOSPA_STORES__ = Object.fromEntries(this.stores);
    }
  }
}

/**
 * Create a new global reactive store.
 */
export function createStore<T>(name: string, initialValue: T): T {
  return SharedStore.getInstance().create(name, initialValue);
}

/**
 * Access a global reactive store.
 */
export function getStore<T>(name: string): T | undefined {
  return SharedStore.getInstance().get(name);
}
