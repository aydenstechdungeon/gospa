// GoSPA Micro Runtime - Ultra-minimal (~5KB target)
// Only includes essential reactive primitives - no DOM, events, or lazy loaders
// Use this for: SSR hydration, minimal state management, progressive enhancement
//
// For more features, use:
// - runtime-core.ts (~15KB): DOM bindings, events, lazy-loaded modules
// - runtime-simple.ts (~18KB): + simple sanitizer
// - runtime.ts (~20KB): + full DOMPurify sanitizer

import { Rune, Derived, Effect, StateMap, batch, effect, watch, type Unsubscribe } from './state.ts';

// Minimal component definition for micro runtime
export interface MicroComponent {
	id: string;
	state: StateMap;
	cleanup: (() => void)[];
}

// Global component registry (minimal)
const components = new Map<string, MicroComponent>();

// Create a minimal component with reactive state
export function createMicroComponent(id: string, initialState: Record<string, unknown> = {}): MicroComponent {
	const state = new StateMap();
	
	for (const [key, value] of Object.entries(initialState)) {
		state.set(key, value);
	}
	
	const component: MicroComponent = {
		id,
		state,
		cleanup: []
	};
	
	components.set(id, component);
	return component;
}

// Destroy a micro component
export function destroyMicroComponent(id: string): void {
	const component = components.get(id);
	if (!component) return;
	
	for (const cleanup of component.cleanup) cleanup();
	component.state.dispose();
	components.delete(id);
}

// Get a micro component
export function getMicroComponent(id: string): MicroComponent | undefined {
	return components.get(id);
}

// Get state from component
export function getMicroState(componentId: string, key: string): Rune<unknown> | undefined {
	return components.get(componentId)?.state.get(key);
}

// Set state value
export function setMicroState(componentId: string, key: string, value: unknown): void {
	const component = components.get(componentId);
	if (component) {
		component.state.set(key, value);
	}
}

// Batch state updates
export { batch };

// Reactive primitives
export { Rune, Derived, Effect, StateMap, effect, watch };

// Types
export type { Unsubscribe };

// Utility: Create a reactive value with automatic cleanup
export function reactive<T>(initialValue: T): {
	get: () => T;
	set: (value: T) => void;
	subscribe: (callback: (value: T) => void) => () => void;
	dispose: () => void;
} {
	const rune = new Rune(initialValue);
	let disposed = false;
	
	return {
		get: () => rune.get(),
		set: (value: T) => {
			if (!disposed) rune.set(value);
		},
		subscribe: (callback: (value: T) => void) => {
			if (disposed) return () => {};
			return watch(rune, (value) => {
				// Handle both single value and array from watch
				const v = Array.isArray(value) ? value[0] : value;
				callback(v as T);
			});
		},
		dispose: () => {
			if (!disposed) {
				disposed = true;
				rune.dispose();
			}
		}
	};
}

// Utility: Create a computed value with automatic cleanup
export function computed<T>(compute: () => T): {
	get: () => T;
	dispose: () => void;
} {
	const derived = new Derived(compute);
	let disposed = false;
	
	return {
		get: () => derived.get(),
		dispose: () => {
			if (!disposed) {
				disposed = true;
				derived.dispose();
			}
		}
	};
}

// Utility: Create a side effect with automatic cleanup
export function sideEffect(effectFn: () => void | (() => void)): () => void {
	const e = new Effect(effectFn);
	return () => e.dispose();
}

// Export disposal utilities for cleanup
export { disposeAll, enableDisposalTracking, getActiveDisposableCount } from './state.ts';
