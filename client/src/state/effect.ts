import type { Rune, Unsubscribe } from "./rune.ts";
import { type Notifier } from "./batch.ts";
import { currentScope } from "./scope.ts";
import type { Disposable } from "./disposal.ts";

/**
 * Declare global debug constant for build-time stripping.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

export type EffectFn = () => void | (() => void);
let effectId = 0;

// Track current effect for dependency collection
export let currentEffect: Effect | null = null;
export const effectStack: Effect[] = [];
export let _tracking = true;

/**
 * Get the currently executing effect for dependency tracking.
 */
export function getCurrentEffect(): Effect | null {
  return currentEffect;
}

/**
 * Set the currently executing effect for dependency tracking.
 */
export function setCurrentEffect(effect: Effect | null): void {
  currentEffect = effect;
}

/**
 * Push an effect onto the stack.
 * Returns the previous effect.
 */
export function pushEffect(effect: Effect): Effect | null {
  const prev = currentEffect;
  effectStack.push(effect);
  currentEffect = effect;
  return prev;
}

/**
 * Pop an effect from the stack.
 */
export function popEffect(): void {
  effectStack.pop();
  currentEffect = effectStack[effectStack.length - 1] || null;
}

/**
 * Effect - management for reactive side effects.
 * Automatically tracks dependencies and re-runs on changes.
 */
export class Effect implements Notifier, Disposable {
  private readonly _fn: EffectFn;
  private _cleanup: (() => void) | void;
  private readonly _dependencies: Set<Rune<unknown>> = new Set();
  private readonly _depUnsubs: Map<Rune<unknown>, Unsubscribe> = new Map();
  private readonly _id: number;
  private _active: boolean = true;
  private _disposed: boolean = false;

  constructor(fn: EffectFn) {
    this._fn = fn;
    this._id = ++effectId;
    this._cleanup = undefined;

    // Register with parent scope if active
    if (currentScope) {
      currentScope.add(this);
    }

    this._run();
  }

  private _run(): void {
    if (!this._active || this._disposed) return;

    // Run cleanup if exists
    if (this._cleanup) {
      try {
        this._cleanup();
      } catch (err) {
        if (typeof GOSPA_DEBUG !== "undefined" && GOSPA_DEBUG) {
          console.error("Effect cleanup failed:", err);
        }
      }
      this._cleanup = undefined;
    }

    const oldDeps = new Set(this._dependencies);
    this._dependencies.clear();

    pushEffect(this);

    try {
      this._cleanup = this._fn();
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
        const unsub = dep.subscribe(() => this.notify());
        this._depUnsubs.set(dep, unsub);
      }
    });
  }

  addDependency(rune: Rune<unknown>): void {
    if (_tracking) {
      this._dependencies.add(rune);
    }
  }

  notify(): void {
    this._run();
  }

  pause(): void {
    this._active = false;
  }

  resume(): void {
    this._active = true;
    this._run();
  }

  dispose(): void {
    if (this._cleanup) {
      this._cleanup();
    }
    this._disposed = true;
    this._depUnsubs.forEach((unsub) => unsub());
    this._depUnsubs.clear();
    this._dependencies.clear();
  }

  isDisposed(): boolean {
    return this._disposed;
  }
}

/**
 * Create a side effect that automatically tracks dependencies.
 */
export function effect(fn: EffectFn): Effect {
  return new Effect(fn);
}

/**
 * untrack - execute code without tracking dependencies.
 */
export function untrack<T>(fn: () => T): T {
  const prev = currentEffect;
  currentEffect = null;
  _tracking = false;
  try {
    return fn();
  } finally {
    currentEffect = prev;
    _tracking = true;
  }
}

/**
 * watch - watch reactive values with a callback.
 */
export function watch<T>(
  sources: Rune<T> | Rune<T>[],
  callback: (values: T | T[], oldValues: T | T[]) => void,
): Unsubscribe {
  const sourceArray = Array.isArray(sources) ? sources : [sources];
  const unsubscribers: Unsubscribe[] = [];
  let previousValues = sourceArray.map((source) => source.get());

  sourceArray.forEach((source) => {
    unsubscribers.push(
      source.subscribe(() => {
        const values = sourceArray.map((s) => s.get());
        const oldValues = previousValues;
        previousValues = [...values];
        callback(
          Array.isArray(sources) ? values : values[0],
          Array.isArray(sources) ? oldValues : oldValues[0],
        );
      }),
    );
  });

  return () => unsubscribers.forEach((unsub) => unsub());
}
