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
import { setupTransitions, fade, fly, slide } from './transition.ts';

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
}

// Runtime configuration
export interface RuntimeConfig {
	wsUrl?: string;
	debug?: boolean;
	performance?: boolean;
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
	console.log('[GoSPA DEBUG] handleServerMessage received:', JSON.stringify(message));
	
	switch (message.type) {
		case 'init':
			if (message.componentId && message.data) {
				const component = components.get(message.componentId);
				if (component) {
					component.states.fromJSON(message.data);
				}
			} else if (message.state) {
				// Global state HMR from server - iterate through components and apply updates based on keys
				for (const component of components.values()) {
					for (const [key, value] of Object.entries(message.state)) {
						if (component.states.get(key) !== undefined) {
							component.states.set(key, value);
						}
					}
				}
				globalState.fromJSON(message.state);
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
			console.log('[GoSPA DEBUG] sync message - data:', message.data, 'key:', (message as any).key, 'value:', (message as any).value);
			// Full state sync
			if (message.data) {
				globalState.fromJSON(message.data);
			} else if ((message as any).key !== undefined && (message as any).value !== undefined) {
				// Partial sync from server broadcast
				console.log('[GoSPA DEBUG] Processing partial sync. Components count:', components.size);
				for (const component of components.values()) {
					const existingRune = component.states.get((message as any).key);
					console.log('[GoSPA DEBUG] Component:', component.id, 'key:', (message as any).key, 'existingRune:', existingRune ? 'exists' : 'not found');
					if (existingRune !== undefined) {
						console.log('[GoSPA DEBUG] Setting state key:', (message as any).key, 'to value:', (message as any).value);
						component.states.set((message as any).key, (message as any).value);
					}
				}
				globalState.set((message as any).key, (message as any).value);
			}
			break;
		case 'error':
			if (config.debug) {
				console.error('Server error:', message.error);
			}
			break;
	}
}

// Create component instance
export function createComponent(def: ComponentDefinition, element?: Element): ComponentInstance {
	const instance: ComponentInstance = {
		id: def.id,
		definition: def,
		states: new StateMap(),
		derived: new Map(),
		unsubscribers: [],
		cleanup: [],
		element: element || null
	};

	// Initialize state
	for (const [key, value] of Object.entries(def.state)) {
		instance.states.set(key, value);
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

// Auto-initialize from DOM
export function autoInit(): void {
	// Find all components with data-gospa-component attribute
	const elements = document.querySelectorAll('[data-gospa-component]');

	for (const element of elements) {
		const id = element.getAttribute('data-gospa-component');
		const stateJson = element.getAttribute('data-gospa-state');

		if (!id) continue;

		const state = stateJson ? JSON.parse(stateJson) : {};

		createComponent({
			id,
			name: id,
			state
		}, element);
	}

	// Set up bindings
	setupBindings();
}

// Set up reactive bindings from DOM attributes
function setupBindings(): void {
	// DEBUG: Log all data-gospa-component elements found
	const allComponents = document.querySelectorAll('[data-gospa-component]');
	console.log('[GoSPA DEBUG] Found components with data-gospa-component:', allComponents.length);
	allComponents.forEach((el, i) => {
		console.log(`[GoSPA DEBUG] Component ${i}:`, el.getAttribute('data-gospa-component'), el);
	});

	// Find all elements with data-bind attribute
	const boundElements = document.querySelectorAll('[data-bind]');

	for (const element of boundElements) {
		const closestComponent = element.closest('[data-gospa-component]');
		const componentId = closestComponent?.getAttribute('data-gospa-component') || '';
		console.log('[GoSPA DEBUG] setupBindings - element:', element, 'closest component:', closestComponent, 'componentId:', componentId);

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
	const eventElements = document.querySelectorAll('[data-on]');

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
	const modelElements = document.querySelectorAll('[data-model]');

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
