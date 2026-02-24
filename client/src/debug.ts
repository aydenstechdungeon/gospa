// Debug utilities - tree-shaken in production builds
// This module is lazy-loaded only when debug features are used

import { Effect } from './state.ts';

export type InspectType = 'init' | 'update';

/**
 * Check if running in development mode
 */
export function isDev(): boolean {
	return typeof window !== 'undefined' &&
		(window as unknown as { __GOSPA_DEV__?: boolean }).__GOSPA_DEV__ !== false;
}

/**
 * $inspect - Debug helper for observing state changes (dev only).
 * In production, this becomes a no-op.
 */
export function inspect<T>(...values: (() => T)[] | T[]): {
	with: (callback: (type: InspectType, value: T[]) => void) => void
} {
	if (!isDev()) {
		return { with: () => { } };
	}

	let firstRun = true;
	const callbacks: Array<(type: InspectType, value: T[]) => void> = [];

	// Log initial values
	const getValues = (): T[] => values.map(v => typeof v === 'function' ? (v as () => T)() : v);

	const logValues = (type: InspectType) => {
		const currentValues = getValues();
		console.log(`%c[${type}]`, 'color: #888', ...currentValues);
		callbacks.forEach(cb => cb(type, currentValues));
	};

	// Set up effect to track changes
	new Effect(() => {
		// Read all values to track them
		getValues();

		if (firstRun) {
			firstRun = false;
			logValues('init');
		} else {
			logValues('update');
		}
	});

	return {
		with: (callback: (type: InspectType, value: T[]) => void) => {
			callbacks.push(callback);
		}
	};
}

/**
 * $inspect.trace - Log which dependencies triggered an effect.
 */
inspect.trace = (label?: string) => {
	if (!isDev()) return;

	console.log(`%c[trace]${label ? ` ${label}` : ''}`, 'color: #666; font-style: italic');
};

/**
 * Performance timing helper for development
 */
export function timing(name: string) {
	if (!isDev()) {
		return { end: () => { } };
	}

	const start = performance.now();
	return {
		end: () => {
			const duration = performance.now() - start;
			console.log(`%c[timing] ${name}: ${duration.toFixed(2)}ms`, 'color: #0a0');
		}
	};
}

/**
 * Memory usage helper for development
 */
export function memoryUsage(label: string) {
	if (!isDev()) return;

	if ('memory' in performance && (performance as unknown as { memory?: { usedJSHeapSize: number } }).memory) {
		const memory = (performance as unknown as { memory: { usedJSHeapSize: number } }).memory;
		const mb = (memory.usedJSHeapSize / 1024 / 1024).toFixed(2);
		console.log(`%c[memory] ${label}: ${mb}MB`, 'color: #a0a');
	}
}

/**
 * Debug logger that only logs in development
 */
export function debugLog(...args: unknown[]): void {
	if (!isDev()) return;
	console.log('%c[debug]', 'color: #888', ...args);
}

/**
 * Create a debug inspector for reactive state
 */
export function createInspector<T>(name: string, state: { get: () => T; subscribe: (fn: (v: T) => void) => () => void }) {
	if (!isDev()) {
		return { log: () => { }, dispose: () => { } };
	}

	console.log(`%c[inspector] ${name} created`, 'color: #08f');

	const unsub = state.subscribe((value) => {
		console.log(`%c[${name}]`, 'color: #08f', value);
	});

	return {
		log: () => {
			console.log(`%c[${name}]`, 'color: #08f', state.get());
		},
		dispose: () => {
			unsub();
			console.log(`%c[inspector] ${name} disposed`, 'color: #888');
		}
	};
}
