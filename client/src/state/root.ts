import { Effect, type EffectFn } from "./effect.ts";

/**
 * EffectRoot - Manual effect lifecycle control.
 * The effect doesn't auto-dispose and must be manually cleaned up.
 */
export class EffectRoot {
  private _effect: Effect | null = null;
  private _fn: EffectFn;
  private _disposed = false;

  constructor(fn: EffectFn) {
    this._fn = fn;
    this._start();
  }

  private _start(): void {
    if (this._disposed) return;
    this._effect = new Effect(this._fn);
  }

  /**
   * Stop the effect and clean up.
   */
  stop(): void {
    if (this._effect) {
      this._effect.dispose();
      this._effect = null;
    }
  }

  /**
   * Restart the effect after stopping.
   */
  restart(): void {
    this.stop();
    this._start();
  }

  /**
   * Permanently dispose the effect root.
   */
  dispose(): void {
    this._disposed = true;
    this.stop();
  }
}

/**
 * Create an effect root with manual lifecycle control.
 */
export function effectRoot(fn: EffectFn): () => void {
  const root = new EffectRoot(fn);
  return () => root.dispose();
}
