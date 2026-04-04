import type { Rune, Subscriber, Unsubscribe } from "./rune.ts";
import { type Notifier, batchDepth, pendingNotifications } from "./batch.ts";
import {
  pushEffect,
  popEffect,
  getCurrentEffect,
  type Effect,
} from "./effect.ts";

/**
 * Declare global debug constant for build-time stripping.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

export type ComputeFn<T> = () => T;

/**
 * Derived - computed state.
 * Automatically computes a value from other runes and caches it.
 */
export class Derived<T> implements Notifier {
  private _value: T;
  private readonly _compute: ComputeFn<T>;
  private readonly _dependencies: Set<Rune<unknown>> = new Set();
  private readonly _subscribers: Set<Subscriber<T>> = new Set();
  private readonly _depUnsubs: Map<Rune<unknown>, Unsubscribe> = new Map();
  private _dirty: boolean = true;
  private _disposed: boolean = false;

  constructor(compute: ComputeFn<T>) {
    this._compute = compute;
    this._value = undefined as T;
    this._recompute();
  }

  get value(): T {
    if (this._dirty) {
      this._recompute();
    }
    this.trackDependency();
    return this._value;
  }

  get(): T {
    return this.value;
  }

  subscribe(fn: Subscriber<T>): Unsubscribe {
    this._subscribers.add(fn);
    return () => this._subscribers.delete(fn);
  }

  private _recompute(): void {
    const oldDeps = new Set(this._dependencies);
    this._dependencies.clear();

    const collector = {
      addDependency: (rune: Rune<unknown>) => {
        this._dependencies.add(rune);
      },
    } as Effect;

    pushEffect(collector);
    try {
      this._value = this._compute();
      this._dirty = false;
    } finally {
      popEffect();
    }

    // Unsubscribe from obsolete dependencies
    oldDeps.forEach((dep) => {
      if (!this._dependencies.has(dep)) {
        const unsub = this._depUnsubs.get(dep);
        if (unsub) {
          unsub();
          this._depUnsubs.delete(dep);
        }
      }
    });

    // Subscribe to new dependencies
    this._dependencies.forEach((dep) => {
      if (!oldDeps.has(dep)) {
        const unsub = dep.subscribe(() => {
          this._dirty = true;
          this._notifySubscribers();
        });
        this._depUnsubs.set(dep, unsub);
      }
    });
  }

  private _notifySubscribers(): void {
    if (batchDepth > 0) {
      pendingNotifications.add(this);
      return;
    }
    this.notify();
  }

  notify(): void {
    const oldValue = this._value;
    if (this._dirty) {
      this._recompute();
    }
    this._subscribers.forEach((fn) => fn(this._value, oldValue));
  }

  private trackDependency(): void {
    // Collect this derived as a dependency of the active effect
    const active = getCurrentEffect();
    if (active) {
      (active as any).addDependency(this as unknown as Rune<unknown>);
    }
  }

  dispose(): void {
    this._disposed = true;
    this._depUnsubs.forEach((unsub) => unsub());
    this._depUnsubs.clear();
    this._dependencies.clear();
    this._subscribers.clear();
  }

  isDisposed(): boolean {
    return this._disposed;
  }
}

/**
 * Create a computed reactive state.
 */
export function derived<T>(compute: ComputeFn<T>): Derived<T> {
  return new Derived(compute);
}
