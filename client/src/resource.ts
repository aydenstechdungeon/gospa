/**
 * GoSPA Resource utility for async data fetching.
 * Provides reactive status, data, and error tracking.
 */

import { Rune, rune } from "./state.ts";

export type ResourceStatus = "idle" | "pending" | "success" | "error";

export class Resource<T, E = Error> {
  private _status: Rune<ResourceStatus>;
  private _data: Rune<T | undefined>;
  private _error: Rune<E | undefined>;
  private _fetcher: () => Promise<T>;

  constructor(fetcher: () => Promise<T>) {
    this._fetcher = fetcher;
    this._status = rune<ResourceStatus>("idle");
    this._data = rune<T | undefined>(undefined);
    this._error = rune<E | undefined>(undefined);
  }

  get status(): ResourceStatus {
    return this._status.get();
  }

  get data(): T | undefined {
    return this._data.get();
  }

  get error(): E | undefined {
    return this._error.get();
  }

  get isPending(): boolean {
    return this.status === "pending";
  }

  get isSuccess(): boolean {
    return this.status === "success";
  }

  get isError(): boolean {
    return this.status === "error";
  }

  async fetch(): Promise<T | undefined> {
    if (this._status.peek() === "pending") return;

    this._status.set("pending");
    this._error.set(undefined);

    try {
      const result = await this._fetcher();
      this._data.set(result);
      this._status.set("success");
      return result;
    } catch (err) {
      this._error.set(err as E);
      this._status.set("error");
      throw err;
    }
  }

  async refetch(): Promise<T | undefined> {
    return this.fetch();
  }

  reset(): void {
    this._status.set("idle");
    this._data.set(undefined);
    this._error.set(undefined);
  }
}

/**
 * Factory function for creating a reactive resource.
 */
export function resourceReactive<T>(fetcher: () => Promise<T>): Resource<T> {
  const r = new Resource(fetcher);
  return r;
}
