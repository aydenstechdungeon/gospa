// GoSPA Client Runtime - Main entry point
// A lightweight runtime for reactive SPAs with Go/Fiber/Templ

import { Rune, Derived, Effect, StateMap, batch, effect, watch, type Unsubscribe } from './state.ts';
import { bindElement, bindTwoWay, renderIf, renderList, registerBinding, unregisterBinding } from './dom.ts';
import { on, offAll, debounce, throttle, delegate, onKey, keys, transformers } from './events.ts';
import { WSClient, initWebSocket, getWebSocketClient, sendAction, syncedRune, applyStateUpdate, type StateMessage } from './websocket.ts';
import {
	navigate,
	back,
	forward,
	go,
	prefetch,
	getCurrentPath,
	isNavigating,
	onBeforeNavigate,
	onAfterNavigate,
	initNavigation,
	destroyNavigation,
	createNavigationState,
	type NavigationOptions
} from './navigation.ts';
import { setupTransitions, fade, fly, slide, scale, blur, crossfade } from './transition.ts';

// Component definition
export interface ComponentDefinition {
	id: string;
	name: string;
	state: Record<string, unknown>;
	actions?: Record<string, (...args: unknown[]) => unknown>;
	computed?: Record<string, () => unknown>;
	watch?: Record<string, (value: unknown, oldValue: unknown) => void>;
	mount?: () => void | (() => void);
	destroy?: () => void;
}

// Component instance
export interface ComponentInstance {
	id: string;
	definition: ComponentDefinition;
	states: StateMap;
	derived: Map<string, Derived<unknown>>;
	unsubscribers: Unsubscribe[];
	cleanup: (() => void)[];
	element: Element | null;
	isLocal: boolean; // true = localStorage persisted, no server sync
}

// Runtime configuration
export interface RuntimeConfig {
	wsUrl?: string;

	// WebSocket Options
	ws?: {
		reconnect?: boolean;
		reconnectDelay?: number;
		reconnectMaxAttempts?: number;
		heartbeat?: number;
		protocols?: string[];
	};

	// Hydration Options
	hydration?: {
		mode?: 'immediate' | 'lazy' | 'visible' | 'idle';
		timeout?: number;
		priorityAttribute?: string;
	};

	// State Options
	state?: {
		batchUpdates?: boolean;
		batchDelay?: number;
		persistLocal?: boolean;
		storageKey?: string;
	};

	// Performance Options
	performance?: boolean | {
		trackMetrics?: boolean;
		logSlowUpdates?: number;
		profileEffects?: boolean;
	};

	// Debug Options
	debug?: boolean | {
		enabled?: boolean;
		logState?: boolean;
		logEffects?: boolean;
		logWebSocket?: boolean;
		traceUpdates?: boolean;
	};
}

// Global component registry
const components = new Map<string, ComponentInstance>();
const globalState = new StateMap();

// Runtime state
let isInitialized = false;
let config: RuntimeConfig = {};

// Initialize runtime
export function init(options: RuntimeConfig = {}): void {
	if (isInitialized) {
		console.warn('GoSPA runtime already initialized');
		return;
	}

	config = options;
	isInitialized = true;

	// Global transition engine via MutationObserver
	setupTransitions();

	// Initialize WebSocket if URL provided
	if (config.wsUrl) {
		const ws = initWebSocket({
			url: config.wsUrl,
			onMessage: handleServerMessage
		});
		ws.connect().catch(err => {
			if (config.debug) {
				console.error('WebSocket connection failed:', err);
			}
		});
	}

	// Set up global error handler
	window.addEventListener('error', (event) => {
		if (config.debug) {
			console.error('Runtime error:', event.error);
		}
	});

	if (config.debug) {
		console.log('GoSPA runtime initialized');
	}

	// Expose to window for debugging and manual triggers
	(window as any).__GOSPA__ = {
		config,
		components,
		globalState,
		init,
		createComponent,
		destroyComponent,
		getComponent,
		getState,
		setState,
		callAction,
		bind,
		autoInit,
		sendAction,
		getWebSocketClient,
		navigate
	};
}

// Handle messages from server
function handleServerMessage(message: StateMessage): void {
	switch (message.type) {
		case 'init':
			if (message.componentId && message.data) {
				const component = components.get(message.componentId);
				if (component) {
					component.states.fromJSON(message.data);
				}
			} else if (message.state) {
				// Global state from server - parse component-scoped keys (e.g., "counter.count")
				const stateObj = message.state as Record<string, unknown>;
				for (const [scopedKey, value] of Object.entries(stateObj)) {
					// Check if key is component-scoped (contains a dot)
					const dotIndex = scopedKey.indexOf('.');
					if (dotIndex > 0) {
						const componentId = scopedKey.substring(0, dotIndex);
						const stateKey = scopedKey.substring(dotIndex + 1);
						const component = components.get(componentId);
						if (component) {
							component.states.set(stateKey, value);
						}
					} else {
						// Non-scoped key - apply to all components that have this key
						for (const component of components.values()) {
							if (component.states.get(scopedKey) !== undefined) {
								component.states.set(scopedKey, value);
							}
						}
						globalState.set(scopedKey, value);
					}
				}
			}
			break;
		case 'update':
			if (message.componentId && message.diff) {
				const component = components.get(message.componentId);
				if (component) {
					component.states.fromJSON(message.diff);
				}
			}
			break;
		case 'sync':
			// Full state sync
			if (message.data) {
				globalState.fromJSON(message.data);
			} else if ((message as any).key !== undefined && (message as any).value !== undefined) {
				const key = (message as any).key;
				const value = (message as any).value;
				const componentId = (message as any).componentId;

				// If componentId is specified, update only that component
				if (componentId) {
					const component = components.get(componentId);
					if (component) {
						component.states.set(key, value);
					}
				} else {
					// No componentId - update all components that have this key
					for (const component of components.values()) {
						const existingRune = component.states.get(key);
						if (existingRune !== undefined) {
							component.states.set(key, value);
						}
					}
					globalState.set(key, value);
				}
			}
			break;
		case 'error':
			if (config.debug) {
				console.error('Server error:', message.error);
			}
			break;
	}
}

// LocalStorage key prefix for local state
const LOCAL_STORAGE_PREFIX = 'gospa_local_';

// Load local state from localStorage
function loadLocalState(componentId: string): Record<string, unknown> | null {
	try {
		const key = LOCAL_STORAGE_PREFIX + componentId;
		const stored = localStorage.getItem(key);
		if (stored) {
			return JSON.parse(stored);
		}
	} catch (e) {
		if (config.debug) {
			console.warn(`[GoSPA] Failed to load local state for ${componentId}:`, e);
		}
	}
	return null;
}

// Save local state to localStorage
function saveLocalState(componentId: string, state: Record<string, unknown>): void {
	try {
		const key = LOCAL_STORAGE_PREFIX + componentId;
		localStorage.setItem(key, JSON.stringify(state));
	} catch (e) {
		if (config.debug) {
			console.warn(`[GoSPA] Failed to save local state for ${componentId}:`, e);
		}
	}
}

// Create component instance
export function createComponent(def: ComponentDefinition, element?: Element, isLocal = false): ComponentInstance {
	const instance: ComponentInstance = {
		id: def.id,
		definition: def,
		states: new StateMap(),
		derived: new Map(),
		unsubscribers: [],
		cleanup: [],
		element: element || null,
		isLocal
	};

	// For local components, try to load from localStorage first
	if (isLocal) {
		const savedState = loadLocalState(def.id);
		if (savedState) {
			for (const [key, value] of Object.entries(savedState)) {
				instance.states.set(key, value);
			}
		}
	}

	// Initialize state (localStorage values take precedence for local components)
	for (const [key, value] of Object.entries(def.state)) {
		if (!isLocal || instance.states.get(key) === undefined) {
			instance.states.set(key, value);
		}
	}

	// For local components, subscribe to state changes and persist to localStorage
	if (isLocal) {
		for (const key of Object.keys(def.state)) {
			const rune = instance.states.get(key);
			if (rune) {
				const unsub = watch(rune, () => {
					saveLocalState(def.id, instance.states.toJSON());
				});
				instance.unsubscribers.push(unsub);
			}
		}
	}

	// Initialize computed properties
	if (def.computed) {
		for (const [key, compute] of Object.entries(def.computed)) {
			const derivedVal = new Derived(compute);
			instance.derived.set(key, derivedVal);
		}
	}

	// Initialize watchers
	if (def.watch) {
		for (const [key, callback] of Object.entries(def.watch)) {
			const state = instance.states.get(key);
			if (state) {
				const unsub = watch(state, callback);
				instance.unsubscribers.push(unsub);
			}
		}
	}

	// Call mount hook
	if (def.mount) {
		const cleanup = def.mount();
		if (cleanup) {
			instance.cleanup.push(cleanup);
		}
	}

	components.set(def.id, instance);

	if (config.debug) {
		console.log(`Component "${def.name}" created with id "${def.id}"`);
	}

	return instance;
}

// Destroy component instance
export function destroyComponent(id: string): void {
	const instance = components.get(id);
	if (!instance) return;

	// Call destroy hook
	if (instance.definition.destroy) {
		instance.definition.destroy();
	}

	// Run cleanup functions
	for (const cleanup of instance.cleanup) {
		cleanup();
	}

	// Run unsubscribers
	for (const unsub of instance.unsubscribers) {
		unsub();
	}

	// Dispose derived
	for (const derivedVal of instance.derived.values()) {
		derivedVal.dispose();
	}

	// Remove from registry
	components.delete(id);

	if (config.debug) {
		console.log(`Component "${id}" destroyed`);
	}
}

// Get component instance
export function getComponent(id: string): ComponentInstance | undefined {
	return components.get(id);
}

// Get state from component
export function getState(componentId: string, key: string): Rune<unknown> | undefined {
	const component = components.get(componentId);
	return component?.states.get(key);
}

// Set state value
export function setState(componentId: string, key: string, value: unknown): void {
	const component = components.get(componentId);
	if (component) {
		component.states.set(key, value);

		// For synced (non-local) components, send update to server for persistence
		if (!component.isLocal) {
			const ws = getWebSocketClient();
			if (ws?.isConnected) {
				ws.send({
					type: 'update',
					componentId,
					payload: { key, value }
				});
			}
		}
	}
}

// Call component action
export function callAction(componentId: string, action: string, ...args: unknown[]): unknown {
	const component = components.get(componentId);
	if (!component || !component.definition.actions) {
		throw new Error(`Action "${action}" not found on component "${componentId}"`);
	}

	const fn = component.definition.actions[action];
	if (!fn) {
		throw new Error(`Action "${action}" not found on component "${componentId}"`);
	}

	return fn(...args);
}

// Bind element to state
export function bind(
	componentId: string,
	element: Element,
	binding: string,
	key: string,
	options?: { twoWay?: boolean; event?: string; transform?: (value: any) => any }
): () => void {
	const state = getState(componentId, key);
	if (!state) {
		console.warn(`State "${key}" not found on component "${componentId}"`);
		return () => { };
	}

	if (options?.twoWay) {
		return bindTwoWay(
			element as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement,
			state as Rune<string | number | boolean>
		);
	}

	return bindElement(element, state as Rune<unknown>, {
		type: binding as 'text' | 'html' | 'value' | 'checked' | 'class' | 'style' | 'attr' | 'prop',
		transform: options?.transform
	});
}

// Validate parsed state is a valid object
function validateState(state: unknown): Record<string, unknown> {
	if (state === null || typeof state !== 'object') {
		return {};
	}
	if (Array.isArray(state)) {
		return {};
	}
	return state as Record<string, unknown>;
}

// Safely parse JSON with validation
function safeParseJson(json: string): Record<string, unknown> | null {
	try {
		const parsed = JSON.parse(json);
		return validateState(parsed);
	} catch {
		if (config.debug) {
			console.warn('[GoSPA] Failed to parse state JSON');
		}
		return null;
	}
}

// Auto-initialize from DOM
export function autoInit(): void {
	// Find all components with data-gospa-component attribute
	const elements = document.querySelectorAll('[data-gospa-component]');

	for (const element of elements) {
		const id = element.getAttribute('data-gospa-component');
		const stateJson = element.getAttribute('data-gospa-state');
		const isLocal = element.hasAttribute('data-gospa-local');

		if (!id) continue;

		const state = stateJson ? safeParseJson(stateJson) ?? {} : {};

		const hydrate = element.getAttribute('data-gospa-hydrate') || (config.hydration?.mode) || 'immediate';

		const initComponent = () => {
			createComponent({
				id,
				name: id,
				state
			}, element, isLocal);
			setupBindings(element);
		};

		if (hydrate === 'visible') {
			const observer = new IntersectionObserver((entries) => {
				if (entries[0].isIntersecting) {
					observer.disconnect();
					initComponent();
				}
			});
			observer.observe(element);
		} else if (hydrate === 'idle' && 'requestIdleCallback' in window) {
			(window as any).requestIdleCallback(() => initComponent(), { timeout: config.hydration?.timeout || 2000 });
		} else {
			// immediate or fallback
			initComponent();
		}
	}
}

// Set up reactive bindings from DOM attributes
function setupBindings(root: Element | Document = document): void {
	// Find all elements with data-bind attribute
	const boundElements = root.querySelectorAll('[data-bind]');

	for (const element of boundElements) {
		const closestComponent = element.closest('[data-gospa-component]');
		const componentId = closestComponent?.getAttribute('data-gospa-component') || '';

		const bindingSpec = element.getAttribute('data-bind');
		const transformName = element.getAttribute('data-transform');
		if (!bindingSpec) continue;

		// Parse binding spec: "key:binding" or "key" (defaults to text)
		const [key, binding = 'text'] = bindingSpec.split(':').map(s => s.trim());

		// Resolve transform if present
		let transform: ((v: unknown) => unknown) | undefined;
		if (transformName) {
			transform = (window as any)[transformName];
			if (typeof transform !== 'function' && config.debug) {
				console.warn(`[GoSPA] Transform "${transformName}" not found or not a function`);
			}
		}

		bind(componentId, element, binding, key, { transform });
	}

	// Find all elements with data-on attribute (event handlers)
	const eventElements = root.querySelectorAll('[data-on]');

	for (const element of eventElements) {
		const componentId = element.closest('[data-gospa-component]')?.getAttribute('data-gospa-component') || '';

		const eventSpec = element.getAttribute('data-on');
		if (!eventSpec) continue;

		// Parse event spec: "event:action" or "event:action:arg1,arg2"
		const [event, action, argsStr] = eventSpec.split(':').map(s => s.trim());
		const args = argsStr ? argsStr.split(',').map(s => s.trim()) : [];

		on(element, event, () => {
			try {
				// Try client action first
				callAction(componentId, action, ...args);
			} catch (e) {
				// Fallback to server action if client action not found
				const ws = getWebSocketClient();
				if (ws?.isConnected) {
					// Extract payload if args are key=value pairs
					const payload: Record<string, unknown> = {};
					args.forEach(arg => {
						const [k, v] = arg.split('=');
						if (v !== undefined) {
							payload[k] = v;
						}
					});
					ws.sendAction(action, Object.keys(payload).length > 0 ? payload : undefined);
				} else if (config.debug) {
					console.error(`[GoSPA] Failed to execute action "${action}":`, e);
				}
			}
		});
	}

	// Find all elements with data-model attribute (two-way binding)
	const modelElements = root.querySelectorAll('[data-model]');

	for (const element of modelElements) {
		const componentId = element.closest('[data-gospa-component]')?.getAttribute('data-gospa-component') || '';

		const key = element.getAttribute('data-model');
		if (!key) continue;

		bind(componentId, element, 'value', key, { twoWay: true });
	}
}

// Export all public APIs
export {
	// State
	Rune,
	Derived,
	Effect,
	StateMap,
	batch,
	effect,
	watch,

	// DOM
	bindElement,
	bindTwoWay,
	renderIf,
	renderList,
	registerBinding,
	unregisterBinding,

	// Events
	on,
	offAll,
	debounce,
	throttle,
	delegate,
	onKey,
	keys,
	transformers,

	// WebSocket
	WSClient,
	initWebSocket,
	getWebSocketClient,
	syncedRune,
	applyStateUpdate,

	// Transitions
	fade,
	fly,
	slide,
	setupTransitions,

	// Navigation
	navigate,
	back,
	forward,
	go,
	prefetch,
	getCurrentPath,
	isNavigating,
	onBeforeNavigate,
	onAfterNavigate,
	initNavigation,
	destroyNavigation,
	createNavigationState
};

// Export types
export type { NavigationOptions };

// Auto-initialize on DOM ready if data-gospa-auto is present
if (typeof document !== 'undefined') {
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', () => {
			if (document.documentElement.hasAttribute('data-gospa-auto')) {
				autoInit();
			}
		});
	} else if (document.documentElement.hasAttribute('data-gospa-auto')) {
		autoInit();
	}
}

// Default export
export default {
	init,
	createComponent,
	destroyComponent,
	getComponent,
	getState,
	setState,
	callAction,
	bind,
	autoInit,

	// Re-exported for convenience
	Rune,
	Derived,
	Effect,
	StateMap,
	batch,
	effect,
	watch,
	on,
	debounce,
	throttle,

	// Transitions
	fade,
	fly,
	slide,
	scale,
	blur,
	crossfade,
	setupTransitions,

	// Navigation
	navigate,
	back,
	forward,
	go,
	prefetch,
	getCurrentPath,
	isNavigating,
	onBeforeNavigate,
	onAfterNavigate,
	initNavigation,
	destroyNavigation,
	createNavigationState
};
