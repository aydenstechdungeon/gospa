// Client-side reactive primitives mirroring Go state package

export type Unsubscribe = () => void;
export type Subscriber<T> = (value: T, oldValue: T) => void;
export type EffectFn = () => void | (() => void);
export type ComputeFn<T> = () => T;
export type CleanupFn = () => void;

let runeId = 0;
let effectId = 0;
let batchDepth = 0;
let pendingNotifications: Set<Notifier> = new Set();
let autoBatchScheduled = false;

interface Notifier {
	notify(): void;
}

/**
 * Schedule automatic batch flush for the next microtask.
 * This enables auto-batching of rapid synchronous state updates
 * without requiring manual batch() calls.
 */
function scheduleAutoBatch(): void {
	if (autoBatchScheduled || batchDepth > 0) return;
	autoBatchScheduled = true;
	
	queueMicrotask(() => {
		autoBatchScheduled = false;
		if (batchDepth === 0 && pendingNotifications.size > 0) {
			const pending = [...pendingNotifications];
			pendingNotifications.clear();
			pending.forEach(n => n.notify());
		}
	});
}

// === Disposal Tracking for Memory Management ===

/**
 * Disposable interface for reactive primitives that need cleanup.
 */
export interface Disposable {
	dispose(): void;
	isDisposed(): boolean;
}

/**
 * Global registry for tracking active reactive primitives.
 * Useful for debugging memory leaks in development.
 */
const activeDisposables: Set<{ deref: () => Disposable | undefined }> = new Set();

// FinalizationRegistry for automatic cleanup notification (ES2021+)
let finalizationRegistry: {
	register(target: object, heldValue: string): void;
	unregister(target: object): void;
} | null = null;

// Check for FinalizationRegistry availability at runtime
if (typeof (globalThis as unknown as { FinalizationRegistry?: unknown }).FinalizationRegistry !== 'undefined') {
	finalizationRegistry = new (globalThis as unknown as { FinalizationRegistry: new <T>(cb: (heldValue: T) => void) => { register(target: object, heldValue: string): void; unregister(target: object): void } }).FinalizationRegistry<string>((_id) => {
		// Called when object is GC'd - could log in dev mode
	});
}

let disposalTrackingEnabled = false;

/**
 * Enable disposal tracking for debugging memory leaks.
 * In production, this should be disabled.
 */
export function enableDisposalTracking(enabled: boolean = true): void {
	disposalTrackingEnabled = enabled;
}

/**
 * Get count of tracked disposables (for debugging).
 * Note: Some may have been garbage collected.
 */
export function getActiveDisposableCount(): number {
	let count = 0;
	for (const ref of activeDisposables) {
		if (ref.deref()) count++;
	}
	return count;
}

/**
 * Force dispose all tracked disposables (for testing/cleanup).
 */
export function disposeAll(): void {
	for (const ref of activeDisposables) {
		const disposable = ref.deref();
		if (disposable && !disposable.isDisposed()) {
			disposable.dispose();
		}
	}
	activeDisposables.clear();
}

function trackDisposable<T extends Disposable>(disposable: T): T {
	if (disposalTrackingEnabled) {
		// Use WeakRef if available (ES2021+)
		if (typeof (globalThis as unknown as { WeakRef?: unknown }).WeakRef !== 'undefined') {
			const WeakRefCtor = (globalThis as unknown as { WeakRef: new <T extends object>(target: T) => { deref: () => T | undefined } }).WeakRef;
			activeDisposables.add(new WeakRefCtor(disposable));
		} else {
			// Fallback: store directly (not ideal but works)
			activeDisposables.add({ deref: () => disposable });
		}
		if (finalizationRegistry) {
			finalizationRegistry.register(disposable, `disposable-${Date.now()}`);
		}
	}
	return disposable;
}

// Track current effect for dependency collection
let currentEffect: Effect | null = null;
const effectStack: Effect[] = [];

// Batch updates
export function batch(fn: () => void): void {
	batchDepth++;
	try {
		fn();
	} finally {
		batchDepth--;
		if (batchDepth === 0) {
			const pending = [...pendingNotifications];
			pendingNotifications.clear();
			pending.forEach(n => n.notify());
		}
	}
}

// Rune - core reactive primitive
export class Rune<T> implements Notifier, Disposable {
	private _value: T;
	private readonly _id: number;
	private readonly _subscribers: Set<Subscriber<T>> = new Set();
	private _dirty: boolean = false;
	private _disposed: boolean = false;

	constructor(initialValue: T) {
		this._value = initialValue;
		this._id = ++runeId;
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
		this._dirty = true;
		this._notifySubscribers(oldValue);
	}

	get(): T {
		this.trackDependency();
		return this._value;
	}

	set(newValue: T): void {
		this.value = newValue;
	}

	update(fn: (current: T) => T): void {
		this.value = fn(this._value);
	}

	subscribe(fn: Subscriber<T>): Unsubscribe {
		this._subscribers.add(fn);
		return () => this._subscribers.delete(fn);
	}

	private _notifySubscribers(oldValue: T): void {
		if (batchDepth > 0) {
			pendingNotifications.add(this);
			return;
		}
		// Auto-batch: schedule notification for next microtask if not already scheduled
		pendingNotifications.add(this);
		scheduleAutoBatch();
	}

	notify(prevValue?: T): void {
		const value = this._value;
		const old = prevValue !== undefined ? prevValue : value;
		this._subscribers.forEach(fn => fn(value, old));
	}

	private _equal(a: T, b: T): boolean {
		if (Object.is(a, b)) return true;
		if (typeof a !== typeof b) return false;
		if (typeof a !== 'object' || a === null || b === null) return false;
		return JSON.stringify(a) === JSON.stringify(b);
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
		this._subscribers.clear();
	}

	/**
	 * Check if the rune has been disposed.
	 */
	isDisposed(): boolean {
		return this._disposed;
	}
}

// Create a new Rune
export function rune<T>(initialValue: T): Rune<T> {
	return new Rune(initialValue);
}

// Derived - computed state
export class Derived<T> implements Notifier {
	private _value: T;
	private readonly _compute: ComputeFn<T>;
	private readonly _dependencies: Set<Rune<unknown>> = new Set();
	private readonly _subscribers: Set<Subscriber<T>> = new Set();
	// _depUnsubs tracks unsubscribe functions for each dep to prevent subscriber leaks
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

		const prevEffect = currentEffect;
		const collector = {
			addDependency: (rune: Rune<unknown>) => {
				this._dependencies.add(rune);
			}
		} as Effect;

		currentEffect = collector;
		try {
			this._value = this._compute();
			this._dirty = false;
		} finally {
			currentEffect = prevEffect;
		}

		// Unsubscribe from deps that are no longer needed (B4 fix: prevents subscriber accumulation)
		oldDeps.forEach(dep => {
			if (!this._dependencies.has(dep)) {
				const unsub = this._depUnsubs.get(dep);
				if (unsub) {
					unsub();
					this._depUnsubs.delete(dep);
				}
			}
		});

		// Subscribe to new deps only
		this._dependencies.forEach(dep => {
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
		const prevValue = this._dirty ? undefined : this._value;
		if (this._dirty) {
			this._recompute();
		}
		const value = this._value;
		this._subscribers.forEach(fn => fn(value, prevValue ?? value));
	}

	private trackDependency(): void {
		if (currentEffect) {
			currentEffect.addDependency(this as unknown as Rune<unknown>);
		}
	}

	dispose(): void {
		this._disposed = true;
		// Unsubscribe from all deps to prevent subscriber leaks
		this._depUnsubs.forEach(unsub => unsub());
		this._depUnsubs.clear();
		this._dependencies.clear();
		this._subscribers.clear();
	}

	/**
	 * Check if the derived has been disposed.
	 */
	isDisposed(): boolean {
		return this._disposed;
	}
}

// Create a Derived
export function derived<T>(compute: ComputeFn<T>): Derived<T> {
	return new Derived(compute);
}

// Effect - side effects
export class Effect implements Notifier {
	private readonly _fn: EffectFn;
	private _cleanup: (() => void) | void;
	private readonly _dependencies: Set<Rune<unknown>> = new Set();
	private readonly _id: number;
	private _active: boolean = true;
	private _disposed: boolean = false;

	constructor(fn: EffectFn) {
		this._fn = fn;
		this._id = ++effectId;
		this._cleanup = undefined;
		this._run();
	}

	private _run(): void {
		if (!this._active || this._disposed) return;

		// Run cleanup if exists
		if (this._cleanup) {
			this._cleanup();
			this._cleanup = undefined;
		}

		// Clear old dependencies
		this._dependencies.clear();

		// Push to effect stack
		effectStack.push(this);
		currentEffect = this;

		try {
			this._cleanup = this._fn();
		} finally {
			effectStack.pop();
			currentEffect = effectStack[effectStack.length - 1] || null;
		}
	}

	addDependency(rune: Rune<unknown>): void {
		this._dependencies.add(rune);
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
		this._dependencies.clear();
	}

	/**
	 * Check if the effect has been disposed.
	 */
	isDisposed(): boolean {
		return this._disposed;
	}
}

// Create an Effect
export function effect(fn: EffectFn): Effect {
	return new Effect(fn);
}

// Watch multiple runes
export function watch<T>(
	sources: Rune<T> | Rune<T>[],
	callback: (values: T | T[], oldValues: T | T[]) => void
): Unsubscribe {
	const sourceArray = Array.isArray(sources) ? sources : [sources];
	const unsubscribers: Unsubscribe[] = [];

	sourceArray.forEach(source => {
		unsubscribers.push(
			source.subscribe((value, oldValue) => {
				const values = sourceArray.map(s => s.get());
				const oldValues = sourceArray.map(s => oldValue);
				callback(
					Array.isArray(sources) ? values : values[0],
					Array.isArray(sources) ? oldValues : oldValues[0]
				);
			})
		);
	});

	return () => unsubscribers.forEach(unsub => unsub());
}

// StateMap - collection of named runes
export class StateMap {
	private readonly _runes: Map<string, Rune<unknown>> = new Map();

	set<T>(key: string, value: T): Rune<T> {
		const existing = this._runes.get(key);
		if (existing) {
			// Update existing Rune to preserve subscribers
			existing.set(value);
			return existing as Rune<T>;
		}
		const r = new Rune(value);
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
			result[key] = rune.get();
		});
		return result;
	}

	fromJSON(data: Record<string, unknown>): void {
		Object.entries(data).forEach(([key, value]) => {
			if (this._runes.has(key)) {
				(this._runes.get(key) as Rune<unknown>).set(value);
			} else {
				this.set(key, value);
			}
		});
	}

	/**
	 * Dispose all runes in the map.
	 */
	dispose(): void {
		this._runes.forEach(rune => {
			if ('dispose' in rune && typeof rune.dispose === 'function') {
				rune.dispose();
			}
		});
		this._runes.clear();
	}

	/**
	 * Check if the map has been disposed.
	 */
	isDisposed(): boolean {
		return this._runes.size === 0;
	}
}

// Create a state map
export function stateMap(): StateMap {
	return new StateMap();
}

// === Additional Svelte 5-like APIs ===

/**
 * untrack - Execute a function without tracking dependencies.
 * Useful when you need to read reactive values inside an effect without subscribing.
 */
export function untrack<T>(fn: () => T): T {
	const prevEffect = currentEffect;
	currentEffect = null;
	try {
		return fn();
	} finally {
		currentEffect = prevEffect;
	}
}

/**
 * PreEffect - Effect that runs BEFORE DOM updates.
 * Useful for reading DOM state that will be affected by a pending update.
 */
export class PreEffect extends Effect {
	private static _preEffects: PreEffect[] = [];
	private static _scheduled = false;

	constructor(fn: EffectFn) {
		super(fn);
		PreEffect._preEffects.push(this);
		PreEffect._scheduleFlush();
	}

	private static _scheduleFlush(): void {
		if (!PreEffect._scheduled) {
			PreEffect._scheduled = true;
			// Use queueMicrotask to run before DOM updates
			queueMicrotask(() => {
				PreEffect._scheduled = false;
				const effects = [...PreEffect._preEffects];
				PreEffect._preEffects = [];
				effects.forEach(e => e.notify());
			});
		}
	}

	override dispose(): void {
		const idx = PreEffect._preEffects.indexOf(this);
		if (idx > -1) PreEffect._preEffects.splice(idx, 1);
		super.dispose();
	}
}

/**
 * Create a pre-effect that runs before DOM updates.
 */
export function preEffect(fn: EffectFn): PreEffect {
	return new PreEffect(fn);
}

/**
 * RuneRaw - Shallow reactive state without deep proxying.
 * Updates require reassignment of the entire value.
 */
export class RuneRaw<T> implements Notifier {
	private _value: T;
	private readonly _id: number;
	private readonly _subscribers: Set<Subscriber<T>> = new Set();

	constructor(initialValue: T) {
		this._value = initialValue;
		this._id = ++runeId;
	}

	get value(): T {
		this.trackDependency();
		return this._value;
	}

	set value(newValue: T) {
		if (Object.is(this._value, newValue)) return;
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

	subscribe(fn: Subscriber<T>): Unsubscribe {
		this._subscribers.add(fn);
		return () => this._subscribers.delete(fn);
	}

	private _notifySubscribers(_oldValue: T): void {
		if (batchDepth > 0) {
			pendingNotifications.add(this);
			return;
		}
		// Auto-batch: schedule notification for next microtask if not already scheduled
		pendingNotifications.add(this);
		scheduleAutoBatch();
	}

	notify(): void {
		const value = this._value;
		this._subscribers.forEach(fn => fn(value, this._value));
	}

	private trackDependency(): void {
		if (currentEffect) {
			currentEffect.addDependency(this as unknown as Rune<unknown>);
		}
	}

	/**
	 * Create a snapshot - non-reactive plain copy.
	 * For objects/arrays, returns a shallow copy.
	 */
	snapshot(): T {
		const val = this._value;
		if (typeof val === 'object' && val !== null) {
			if (Array.isArray(val)) return [...val] as T;
			return { ...val } as T;
		}
		return val;
	}
}

/**
 * Create a raw (shallow) rune.
 */
export function runeRaw<T>(initialValue: T): RuneRaw<T> {
	return new RuneRaw(initialValue);
}

/**
 * snapshot - Create a non-reactive plain copy of a value.
 * Works with Rune, RuneRaw, or plain values.
 */
export function snapshot<T>(value: T | Rune<T> | RuneRaw<T>): T {
	if (value instanceof RuneRaw) {
		return value.snapshot();
	}
	if (value instanceof Rune) {
		const val = value.get();
		if (typeof val === 'object' && val !== null) {
			if (Array.isArray(val)) return [...val] as T;
			return { ...val } as T;
		}
		return val;
	}
	// Plain value - return as-is or shallow copy
	if (typeof value === 'object' && value !== null) {
		if (Array.isArray(value)) return [...value] as T;
		return { ...value } as T;
	}
	return value;
}

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
 * Returns a cleanup function.
 */
export function effectRoot(fn: EffectFn): () => void {
	const root = new EffectRoot(fn);
	return () => root.dispose();
}

/**
 * tracking - Check if currently inside a reactive tracking context.
 */
export function tracking(): boolean {
	return currentEffect !== null;
}

// === Async Derived and Resource Patterns ===

export type ResourceStatus = 'idle' | 'pending' | 'success' | 'error';

/**
 * DerivedAsync - Async computed values with loading/error states.
 * Recomputes when dependencies change.
 */
export class DerivedAsync<T, E = Error> implements Notifier {
	private _value: T | undefined;
	private _error: E | undefined;
	private _status: ResourceStatus = 'pending';
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
		return this._status === 'pending';
	}

	get isSuccess(): boolean {
		return this._status === 'success';
	}

	get isError(): boolean {
		return this._status === 'error';
	}

	get(): T | undefined {
		return this.value;
	}

	subscribe(fn: Subscriber<T | undefined>): Unsubscribe {
		this._subscribers.add(fn);
		return () => this._subscribers.delete(fn);
	}

	private async _recompute(): Promise<void> {
		// Abort previous request
		if (this._abortController) {
			this._abortController.abort();
		}
		this._abortController = new AbortController();

		// Track dependencies
		const oldDeps = new Set(this._dependencies);
		this._dependencies.clear();

		const prevEffect = currentEffect;
		const collector = {
			addDependency: (rune: Rune<unknown>) => {
				this._dependencies.add(rune);
			}
		} as Effect;

		currentEffect = collector;
		let promise: Promise<T>;
		try {
			promise = this._compute();
			this._dirty = false;
		} finally {
			currentEffect = prevEffect;
		}

		// Subscribe to new dependencies
		this._dependencies.forEach(dep => {
			if (!oldDeps.has(dep)) {
				dep.subscribe(() => {
					this._dirty = true;
					this._recompute();
				});
			}
		});

		// Execute async computation
		this._status = 'pending';
		this._notifySubscribers();

		try {
			const result = await promise;
			if (this._abortController?.signal.aborted) return;
			this._value = result;
			this._error = undefined;
			this._status = 'success';
		} catch (err) {
			if (this._abortController?.signal.aborted) return;
			this._error = err as E;
			this._status = 'error';
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
		this._subscribers.forEach(fn => fn(value, this._value));
	}

	private trackDependency(): void {
		if (currentEffect) {
			currentEffect.addDependency(this as unknown as Rune<unknown>);
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

	/**
	 * Check if the derived async has been disposed.
	 */
	isDisposed(): boolean {
		return this._disposed;
	}
}

/**
 * Create an async derived value.
 */
export function derivedAsync<T, E = Error>(compute: () => Promise<T>): DerivedAsync<T, E> {
	return new DerivedAsync(compute);
}

/**
 * Resource - Wrap async data fetching with reactive status.
 * Similar to SvelteKit's loading stores and Runed's resource.
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
		this._status = new Rune<ResourceStatus>('idle');
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
		return this._status.get() === 'idle';
	}

	get isPending(): boolean {
		return this._status.get() === 'pending';
	}

	get isSuccess(): boolean {
		return this._status.get() === 'success';
	}

	get isError(): boolean {
		return this._status.get() === 'error';
	}

	/**
	 * Fetch or refetch the resource.
	 */
	async refetch(): Promise<void> {
		// Abort previous request
		if (this._abortController) {
			this._abortController.abort();
		}
		this._abortController = new AbortController();

		this._status.set('pending');
		this._error.set(undefined);

		try {
			const result = await this._fetcher();
			if (this._abortController?.signal.aborted) return;
			this._data.set(result);
			this._status.set('success');
		} catch (err) {
			if (this._abortController?.signal.aborted) return;
			this._error.set(err as E);
			this._status.set('error');
		}
	}

	/**
	 * Reset to idle state.
	 */
	reset(): void {
		if (this._abortController) {
			this._abortController.abort();
			this._abortController = null;
		}
		this._data.set(undefined);
		this._error.set(undefined);
		this._status.set('idle');
	}

	/**
	 * Dispose the resource, aborting any pending requests.
	 */
	dispose(): void {
		if (this._abortController) {
			this._abortController.abort();
			this._abortController = null;
		}
		this._data.dispose();
		this._error.dispose();
		this._status.dispose();
	}

	/**
	 * Check if the resource has been disposed.
	 */
	isDisposed(): boolean {
		return this._data.isDisposed() && this._error.isDisposed() && this._status.isDisposed();
	}
}

/**
 * Create a resource from an async fetcher.
 */
export function resource<T, E = Error>(fetcher: () => Promise<T>): Resource<T, E> {
	return new Resource(fetcher);
}

/**
 * Create a resource that auto-fetches when sources change.
 */
export function resourceReactive<T, E = Error>(
	sources: Rune<unknown> | Rune<unknown>[],
	fetcher: () => Promise<T>
): Resource<T, E> {
	const res = new Resource<T, E>(fetcher);
	const sourceArray = Array.isArray(sources) ? sources : [sources];

	// Auto-refetch when sources change
	sourceArray.forEach(source => {
		source.subscribe(() => {
			res.refetch();
		});
	});

	// Initial fetch
	res.refetch();

	return res;
}

// === Debug utilities - lazy loaded from separate module ===
// Re-exported for convenience, but tree-shaken in production
export { inspect, isDev, timing, memoryUsage, debugLog, createInspector } from './debug.ts';
export type { InspectType } from './debug.ts';

// === Deep Watch Path Support ===

/**
 * Get a nested property value by dot-separated path.
 */
function getByPath<T>(obj: T, path: string): unknown {
	const parts = path.split('.');
	let current: unknown = obj;

	for (const part of parts) {
		if (current === null || current === undefined) return undefined;
		if (typeof current !== 'object') return undefined;
		current = (current as Record<string, unknown>)[part];
	}

	return current;
}

/**
 * Watch a specific path in an object for changes.
 * Returns unsubscribe function.
 */
export function watchPath<T extends object>(
	obj: Rune<T>,
	path: string,
	callback: (value: unknown, oldValue: unknown) => void
): Unsubscribe {
	let oldValue: unknown;

	return obj.subscribe((current) => {
		const newValue = getByPath(current, path);
		if (newValue !== oldValue) {
			callback(newValue, oldValue);
			oldValue = newValue;
		}
	});
}

/**
 * Create a derived value from a specific path in a reactive object.
 */
export function derivedPath<T extends object>(
	obj: Rune<T>,
	path: string
): Derived<unknown> {
	return new Derived(() => getByPath(obj.get(), path));
}
