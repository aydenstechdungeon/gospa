import { Rune, type Subscriber, type Unsubscribe } from "./rune.ts";
import { type Notifier, batchDepth, pendingNotifications } from "./batch.ts";
import { pushEffect, popEffect, getCurrentEffect } from "./effect.ts";
import type { Effect } from "./effect.ts";

/**
 * Declare global debug constant for build-time stripping.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

export type ResourceStatus = "idle" | "pending" | "success" | "error";

/**
 * DerivedAsync - Async computed values with loading/error states.
 * Recomputes when dependencies change.
 */
export class DerivedAsync<T, E = Error> implements Notifier {
  private _value: T | undefined;
  private _error: E | undefined;
  private _status: ResourceStatus = "pending";
  private readonly _compute: () => Promise<T>;
  private readonly _dependencies: Set<Rune<unknown>> = new Set();
  private readonly _subscribers: Set<Subscriber<T | undefined>> = new Set();
  private _dirty: boolean = true;
  private _disposed: boolean = false;
  private _abortController: AbortController | null = null;

  constructor(compute: () => Promise<T>) {
    this._compute = compute;
    this._recompute();
  }

  get value(): T | undefined {
    if (this._dirty) {
      this._recompute();
    }
    this.trackDependency();
    return this._value;
  }

  get error(): E | undefined {
    return this._error;
  }

  get status(): ResourceStatus {
    return this._status;
  }

  get isPending(): boolean {
    return this._status === "pending";
  }

  get isSuccess(): boolean {
    return this._status === "success";
  }

  get isError(): boolean {
    return this._status === "error";
  }

  get(): T | undefined {
    return this.value;
  }

  subscribe(fn: Subscriber<T | undefined>): Unsubscribe {
    this._subscribers.add(fn);
    return () => this._subscribers.delete(fn);
  }

  private async _recompute(): Promise<void> {
    if (this._abortController) {
      this._abortController.abort();
    }
    this._abortController = new AbortController();

    const oldDeps = new Set(this._dependencies);
    this._dependencies.clear();

    const collector = {
      addDependency: (rune: Rune<unknown>) => {
        this._dependencies.add(rune);
      },
    } as Effect;

    pushEffect(collector);
    let promise: Promise<T>;
    try {
      promise = this._compute();
      this._dirty = false;
    } finally {
      popEffect();
    }

    this._dependencies.forEach((dep) => {
      if (!oldDeps.has(dep)) {
        dep.subscribe(() => {
          this._dirty = true;
          this._recompute();
        });
      }
    });

    this._status = "pending";
    this._notifySubscribers();

    try {
      const result = await promise;
      if (this._abortController?.signal.aborted) return;
      this._value = result;
      this._error = undefined;
      this._status = "success";
    } catch (err) {
      if (this._abortController?.signal.aborted) return;
      this._error = err as E;
      this._status = "error";
    }

    this._notifySubscribers();
  }

  private _notifySubscribers(): void {
    if (batchDepth > 0) {
      pendingNotifications.add(this);
      return;
    }
    this.notify();
  }

  notify(): void {
    const value = this._value;
    this._subscribers.forEach((fn) => fn(value, this._value));
  }

  private trackDependency(): void {
    const active = getCurrentEffect();
    if (active) {
      (active as any).addDependency(this as unknown as Rune<unknown>);
    }
  }

  dispose(): void {
    this._disposed = true;
    if (this._abortController) {
      this._abortController.abort();
    }
    this._dependencies.clear();
    this._subscribers.clear();
  }

  isDisposed(): boolean {
    return this._disposed;
  }
}

/**
 * Resource - Wrap async data fetching with reactive status.
 */
export class Resource<T, E = Error> {
  private _data: Rune<T | undefined>;
  private _error: Rune<E | undefined>;
  private _status: Rune<ResourceStatus>;
  private _fetcher: () => Promise<T>;
  private _abortController: AbortController | null = null;

  constructor(fetcher: () => Promise<T>) {
    this._fetcher = fetcher;
    this._data = new Rune<T | undefined>(undefined);
    this._error = new Rune<E | undefined>(undefined);
    this._status = new Rune<ResourceStatus>("idle");
  }

  get data(): T | undefined {
    return this._data.get();
  }

  get error(): E | undefined {
    return this._error.get();
  }

  get status(): ResourceStatus {
    return this._status.get();
  }

  get isIdle(): boolean {
    return this._status.get() === "idle";
  }

  get isPending(): boolean {
    return this._status.get() === "pending";
  }

  get isSuccess(): boolean {
    return this._status.get() === "success";
  }

  get isError(): boolean {
    return this._status.get() === "error";
  }

  async refetch(): Promise<void> {
    if (this._abortController) {
      this._abortController.abort();
    }
    this._abortController = new AbortController();

    this._status.set("pending");
    this._error.set(undefined);

    try {
      const result = await this._fetcher();
      if (this._abortController?.signal.aborted) return;
      this._data.set(result);
      this._status.set("success");
    } catch (err) {
      if (this._abortController?.signal.aborted) return;
      this._error.set(err as E);
      this._status.set("error");
    }
  }

  reset(): void {
    if (this._abortController) {
      this._abortController.abort();
      this._abortController = null;
    }
    this._data.set(undefined);
    this._error.set(undefined);
    this._status.set("idle");
  }

  dispose(): void {
    if (this._abortController) {
      this._abortController.abort();
      this._abortController = null;
    }
    this._data.dispose();
    this._error.dispose();
    this._status.dispose();
  }

  isDisposed(): boolean {
    return (
      this._data.isDisposed() &&
      this._error.isDisposed() &&
      this._status.isDisposed()
    );
  }
}

export function derivedAsync<T, E = Error>(
  compute: () => Promise<T>,
): DerivedAsync<T, E> {
  return new DerivedAsync(compute);
}

export function resource<T, E = Error>(
  fetcher: () => Promise<T>,
): Resource<T, E> {
  return new Resource(fetcher);
}

export function resourceReactive<T, E = Error>(
  sources: Rune<unknown> | Rune<unknown>[],
  fetcher: () => Promise<T>,
): Resource<T, E> {
  const res = new Resource<T, E>(fetcher);
  const sourceArray = Array.isArray(sources) ? sources : [sources];

  sourceArray.forEach((source) => {
    source.subscribe(() => {
      res.refetch();
    });
  });

  res.refetch();
  return res;
}
