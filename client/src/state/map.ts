import { Rune, type RuneOptions } from "./rune.ts";

/**
 * StateMap - collection of named Runes.
 * Used by components to manage internal state.
 */
export class StateMap {
  private readonly _runes: Map<string, Rune<unknown>> = new Map();

  set<T>(key: string, value: T, options?: RuneOptions): Rune<T> {
    const existing = this._runes.get(key);
    if (existing) {
      existing.set(value);
      return existing as Rune<T>;
    }
    const r = new Rune(value, options);
    this._runes.set(key, r as unknown as Rune<unknown>);
    return r;
  }

  get<T>(key: string): Rune<T> | undefined {
    return this._runes.get(key) as Rune<T> | undefined;
  }

  has(key: string): boolean {
    return this._runes.has(key);
  }

  delete(key: string): boolean {
    return this._runes.delete(key);
  }

  clear(): void {
    this._runes.clear();
  }

  toJSON(): Record<string, unknown> {
    const result: Record<string, unknown> = {};
    this._runes.forEach((rune, key) => {
      result[key] = rune.peek();
    });
    return result;
  }

  fromJSON(data: Record<string, unknown>, options?: RuneOptions): void {
    Object.entries(data).forEach(([key, value]) => {
      if (this._runes.has(key)) {
        (this._runes.get(key) as Rune<unknown>).set(value);
      } else {
        this.set(key, value, options);
      }
    });
  }

  dispose(): void {
    this._runes.forEach((rune) => {
      if ("dispose" in rune && typeof rune.dispose === "function") {
        rune.dispose();
      }
    });
    this._runes.clear();
  }

  isDisposed(): boolean {
    return this._runes.size === 0;
  }
}

/**
 * Create a state map.
 */
export function stateMap(): StateMap {
  return new StateMap();
}
