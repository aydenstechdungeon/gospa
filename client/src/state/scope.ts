import type { Disposable } from "./disposal.ts";

/**
 * EffectScope - manages a group of reactive effects and disposables.
 * When the scope is disposed, all registered disposables are also disposed.
 */
export class EffectScope implements Disposable {
  private _disposables: Set<Disposable> = new Set();
  private _disposed = false;
  private _parent: EffectScope | null = null;

  constructor(parent: EffectScope | null = currentScope) {
    this._parent = parent;
    if (parent) {
      parent.add(this);
    }
  }

  add(disposable: Disposable): void {
    if (this._disposed) {
      disposable.dispose();
      return;
    }
    this._disposables.add(disposable);
  }

  remove(disposable: Disposable): void {
    this._disposables.delete(disposable);
  }

  dispose(): void {
    if (this._disposed) return;
    this._disposed = true;

    for (const disposable of this._disposables) {
      disposable.dispose();
    }
    this._disposables.clear();

    if (this._parent) {
      this._parent.remove(this);
      this._parent = null;
    }
  }

  isDisposed(): boolean {
    return this._disposed;
  }

  /**
   * Run a function within this scope.
   * Any effects created during the execution will be registered to this scope.
   */
  run<T>(fn: () => T): T {
    const prev = currentScope;
    // eslint-disable-next-line @typescript-eslint/no-use-before-define
    currentScope = this;
    try {
      return fn();
    } finally {
      currentScope = prev;
    }
  }
}

/**
 * The currently active effect scope.
 */
export let currentScope: EffectScope | null = null;

/**
 * Set the currently active effect scope.
 */
export function setCurrentScope(scope: EffectScope | null): void {
  currentScope = scope;
}

/**
 * Create a new effect scope.
 */
export function effectScope(): EffectScope {
  return new EffectScope();
}
