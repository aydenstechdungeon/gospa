import { areEqual } from "./equality.ts";
import { type Notifier, batchDepth, addToBatch } from "./batch.ts";
import { type Disposable, trackDisposable } from "./disposal.ts";
import { currentEffect } from "./effect.ts";

export type Subscriber<T> = (value: T, oldValue: T) => void;
export type Unsubscribe = () => void;

let runeId = 0;

export interface RuneOptions {
  deep?: boolean;
}

/**
 * Rune - core reactive primitive.
 * Can store any type of state and notifies subscribers on changes.
 */
export class Rune<T> implements Notifier, Disposable {
  private _value: T;
  private readonly _id: number;
  private readonly _subscribers: (Subscriber<T> | null)[] = [];
  private _sv = 0;
  private _disposed: boolean = false;
  private _hasPendingOldValue: boolean = false;
  private _pendingOldValue?: T;
  private _deep: boolean;

  constructor(initialValue: T, options: RuneOptions = {}) {
    this._value = initialValue;
    this._id = ++runeId;
    this._deep = options.deep ?? false;
    trackDisposable(this);
  }

  get value(): T {
    this.trackDependency();
    return this._value;
  }

  set value(newValue: T) {
    if (this._equal(this._value, newValue)) return;
    const oldValue = this._value;
    this._value = newValue;
    this._notifySubscribers(oldValue);
  }

  get(): T {
    this.trackDependency();
    return this._value;
  }

  set(newValue: T): void {
    this.value = newValue;
  }

  /**
   * Get value without tracking dependencies.
   */
  peek(): T {
    return this._value;
  }

  update(fn: (current: T) => T): void {
    this.value = fn(this._value);
  }

  subscribe(fn: Subscriber<T>): Unsubscribe {
    this._subscribers.push(fn);
    const i = this._subscribers.length - 1;
    const v = this._sv;
    return () => {
      if (this._sv === v) {
        this._subscribers[i] = null;
      }
    };
  }

  private _notifySubscribers(oldValue: T): void {
    if (!this._hasPendingOldValue) {
      this._hasPendingOldValue = true;
      this._pendingOldValue = oldValue;
    }

    if (batchDepth > 0) {
      addToBatch(this);
      return;
    }

    this.notify(oldValue);
  }

  notify(prevValue?: T): void {
    const value = this._value;
    const old = this._hasPendingOldValue
      ? (this._pendingOldValue as T)
      : prevValue !== undefined
        ? prevValue
        : value;
    this._hasPendingOldValue = false;
    this._pendingOldValue = undefined;
    
    const subs = this._subscribers;
    for (let i = 0; i < subs.length; i++) {
      const fn = subs[i];
      if (fn) fn(value, old);
    }
  }

  private _equal(a: T, b: T): boolean {
    return areEqual(a, b, this._deep);
  }

  private trackDependency(): void {
    if (currentEffect) {
      currentEffect.addDependency(this as unknown as Rune<unknown>);
    }
  }

  toJSON(): { id: number; value: T } {
    return { id: this._id, value: this._value };
  }

  /**
   * Dispose the rune, clearing all subscribers.
   * After disposal, the rune will no longer notify subscribers.
   */
  dispose(): void {
    this._disposed = true;
    this._sv++;
    this._subscribers.length = 0;
  }

  /**
   * Check if the rune has been disposed.
   */
  isDisposed(): boolean {
    return this._disposed;
  }
}

/**
 * Create a new Rune.
 * By default, use referential equality for updates.
 * Set options.deep = true for deep property comparison.
 */
export function rune<T>(initialValue: T, options?: RuneOptions): Rune<T> {
  return new Rune(initialValue, options);
}

/**
 * RuneRaw - Shallow reactive state without deep proxying.
 * Updates require reassignment of the entire value.
 * @deprecated Use rune(value, { deep: false }) instead
 */
export class RuneRaw<T> extends Rune<T> {
  constructor(initialValue: T) {
    super(initialValue, { deep: false });
  }

  /**
   * Create a snapshot - non-reactive plain copy.
   * For objects/arrays, returns a shallow copy.
   */
  snapshot(): T {
    const val = this.get();
    if (typeof val === "object" && val !== null) {
      if (Array.isArray(val)) return [...val] as T;
      return { ...val } as T;
    }
    return val;
  }
}

export function runeRaw<T>(initialValue: T): RuneRaw<T> {
  return new RuneRaw(initialValue);
}
