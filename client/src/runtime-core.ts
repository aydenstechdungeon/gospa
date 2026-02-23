// GoSPA Core Runtime - Minimal core (~15KB target)
// Only includes essential state, DOM bindings, and events

import { Rune, Derived, Effect, StateMap, batch, effect, watch, type Unsubscribe } from './state.ts';
import { bindElement, bindTwoWay, renderIf, renderList, registerBinding, unregisterBinding } from './dom.ts';
import { on, offAll, debounce, throttle, delegate, onKey, keys, transformers } from './events.ts';
import type { StateMessage } from './websocket.ts';

// Re-export StateMessage for convenience
export type { StateMessage };

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
	isLocal: boolean;
}

// Runtime configuration (minimal)
export interface RuntimeConfig {
	wsUrl?: string;
	debug?: boolean;
	onConnectionError?: (error: Error) => void;
	hydration?: {
		mode?: 'immediate' | 'lazy' | 'visible' | 'idle';
		timeout?: number;
	};
}

// Global component registry
const components = new Map<string, ComponentInstance>();
const globalState = new StateMap();

// Runtime state
let isInitialized = false;
let config: RuntimeConfig = {};

// Lazy-loaded modules
let wsModule: Promise<typeof import('./websocket.ts')> | null = null;
let navModule: Promise<typeof import('./navigation.ts')> | null = null;
let transitionModule: Promise<typeof import('./transition.ts')> | null = null;

// Initialize runtime
export function init(options: RuntimeConfig = {}): void {
	if (isInitialized) {
		console.warn('GoSPA runtime already initialized');
		return;
	}

	config = options;
	isInitialized = true;

	// Initialize WebSocket if URL provided (lazy load)
	if (config.wsUrl) {
		wsModule = import('./websocket.ts').then(mod => {
			const ws = mod.initWebSocket({
				url: config.wsUrl!,
				onMessage: handleServerMessage
			});
			ws.connect().catch(err => {
				if (config.onConnectionError) {
					config.onConnectionError(err);
				} else if (config.debug) {
					console.error('WebSocket connection failed:', err);
				}
			});
			return mod;
		});
	}

	// Set up global error handler
	window.addEventListener('error', (event) => {
		if (config.debug) console.error('Runtime error:', event.error);
	});

	// Expose to window for debugging
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
		autoInit
	};
}

// Handle messages from server
function handleServerMessage(message: StateMessage): void {
	switch (message.type) {
		case 'init':
			if (message.componentId && message.data) {
				const component = components.get(message.componentId);
				if (component) component.states.fromJSON(message.data);
			} else if (message.state) {
				const stateObj = message.state as Record<string, unknown>;
				for (const [scopedKey, value] of Object.entries(stateObj)) {
					const dotIndex = scopedKey.indexOf('.');
					if (dotIndex > 0) {
						const componentId = scopedKey.substring(0, dotIndex);
						const stateKey = scopedKey.substring(dotIndex + 1);
						const component = components.get(componentId);
						if (component) component.states.set(stateKey, value);
					} else {
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
				if (component) component.states.fromJSON(message.diff);
			}
			break;
		case 'sync':
			if (message.data) {
				globalState.fromJSON(message.data);
			}
			break;
		case 'error':
			if (config.debug) console.error('Server error:', message.error);
			break;
		// Ignore ping/pong/action messages - handled by websocket module
	}
}

// LocalStorage key prefix
const LOCAL_STORAGE_PREFIX = 'gospa_local_';

function loadLocalState(componentId: string): Record<string, unknown> | null {
	try {
		const stored = localStorage.getItem(LOCAL_STORAGE_PREFIX + componentId);
		return stored ? JSON.parse(stored) : null;
	} catch {
		return null;
	}
}

function saveLocalState(componentId: string, state: Record<string, unknown>): void {
	try {
		localStorage.setItem(LOCAL_STORAGE_PREFIX + componentId, JSON.stringify(state));
	} catch {
		// Ignore storage errors
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

	// Load from localStorage for local components
	if (isLocal) {
		const savedState = loadLocalState(def.id);
		if (savedState) {
			for (const [key, value] of Object.entries(savedState)) {
				instance.states.set(key, value);
			}
		}
	}

	// Initialize state
	for (const [key, value] of Object.entries(def.state)) {
		if (!isLocal || instance.states.get(key) === undefined) {
			instance.states.set(key, value);
		}
	}

	// Persist local state changes
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
			instance.derived.set(key, new Derived(compute));
		}
	}

	// Initialize watchers
	if (def.watch) {
		for (const [key, callback] of Object.entries(def.watch)) {
			const state = instance.states.get(key);
			if (state) {
				instance.unsubscribers.push(watch(state, callback));
			}
		}
	}

	// Call mount hook
	if (def.mount) {
		const cleanup = def.mount();
		if (cleanup) instance.cleanup.push(cleanup);
	}

	components.set(def.id, instance);
	return instance;
}

// Destroy component instance
export function destroyComponent(id: string): void {
	const instance = components.get(id);
	if (!instance) return;

	if (instance.definition.destroy) instance.definition.destroy();
	for (const cleanup of instance.cleanup) cleanup();
	for (const unsub of instance.unsubscribers) unsub();
	for (const derivedVal of instance.derived.values()) derivedVal.dispose();
	components.delete(id);
}

// Get component instance
export function getComponent(id: string): ComponentInstance | undefined {
	return components.get(id);
}

// Get state from component
export function getState(componentId: string, key: string): Rune<unknown> | undefined {
	return components.get(componentId)?.states.get(key);
}

// Set state value
export async function setState(componentId: string, key: string, value: unknown): Promise<void> {
	const component = components.get(componentId);
	if (!component) return;

	component.states.set(key, value);

	// Send update to server for synced components
	if (!component.isLocal && wsModule) {
		const mod = await wsModule;
		const ws = mod.getWebSocketClient();
		if (ws?.isConnected) {
			ws.send({ type: 'update', componentId, payload: { key, value } });
		}
	}
}

// Call component action
export function callAction(componentId: string, action: string, ...args: unknown[]): unknown {
	const component = components.get(componentId);
	if (!component?.definition.actions?.[action]) {
		throw new Error(`Action "${action}" not found on component "${componentId}"`);
	}
	return component.definition.actions[action](...args);
}

// Bind element to state
export function bind(
	componentId: string,
	element: Element,
	binding: string,
	key: string,
	options?: { twoWay?: boolean; transform?: (value: any) => any }
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
		type: binding as any,
		transform: options?.transform
	});
}

// Auto-initialize from DOM
export function autoInit(): void {
	const elements = document.querySelectorAll('[data-gospa-component]');

	for (const element of elements) {
		const id = element.getAttribute('data-gospa-component');
		const stateJson = element.getAttribute('data-gospa-state');
		const isLocal = element.hasAttribute('data-gospa-local');
		if (!id) continue;

		let state: Record<string, unknown> = {};
		try {
			state = stateJson ? JSON.parse(stateJson) : {};
		} catch {
			// Ignore parse errors
		}

		const hydrate = element.getAttribute('data-gospa-hydrate') || config.hydration?.mode || 'immediate';

		const initComponent = () => {
			createComponent({ id, name: id, state }, element, isLocal);
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
			initComponent();
		}
	}
}

// Set up reactive bindings from DOM attributes
function setupBindings(root: Element | Document = document): void {
	// Data-bind elements
	for (const element of root.querySelectorAll('[data-bind]')) {
		const componentId = element.closest('[data-gospa-component]')?.getAttribute('data-gospa-component') || '';
		const bindingSpec = element.getAttribute('data-bind');
		const transformName = element.getAttribute('data-transform');
		if (!bindingSpec) continue;

		const [key, binding = 'text'] = bindingSpec.split(':').map(s => s.trim());
		let transform: ((v: unknown) => unknown) | undefined;
		if (transformName && typeof (window as any)[transformName] === 'function') {
			transform = (window as any)[transformName];
		}

		bind(componentId, element, binding, key, { transform });
	}

	// Event handlers
	for (const element of root.querySelectorAll('[data-on]')) {
		const componentId = element.closest('[data-gospa-component]')?.getAttribute('data-gospa-component') || '';
		const eventSpec = element.getAttribute('data-on');
		if (!eventSpec) continue;

		const [event, action, argsStr] = eventSpec.split(':').map(s => s.trim());
		const args = argsStr ? argsStr.split(',').map(s => s.trim()) : [];

		on(element, event, async () => {
			try {
				callAction(componentId, action, ...args);
			} catch {
				if (wsModule) {
					const mod = await wsModule;
					const ws = mod.getWebSocketClient();
					if (ws?.isConnected) ws.sendAction(action);
				}
			}
		});
	}

	// Two-way bindings
	for (const element of root.querySelectorAll('[data-model]')) {
		const componentId = element.closest('[data-gospa-component]')?.getAttribute('data-gospa-component') || '';
		const key = element.getAttribute('data-model');
		if (key) bind(componentId, element, 'value', key, { twoWay: true });
	}
}

// Export core APIs
export {
	Rune, Derived, Effect, StateMap, batch, effect, watch,
	bindElement, bindTwoWay, renderIf, renderList, registerBinding, unregisterBinding,
	on, offAll, debounce, throttle, delegate, onKey, keys, transformers
};

// Lazy module loaders
export async function getWebSocket() {
	if (!wsModule) wsModule = import('./websocket.ts');
	return wsModule;
}

export async function getNavigation() {
	if (!navModule) navModule = import('./navigation.ts');
	return navModule;
}

export async function getTransitions() {
	if (!transitionModule) transitionModule = import('./transition.ts');
	return transitionModule;
}

// Auto-initialize on DOM ready
if (typeof document !== 'undefined') {
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', () => {
			if (document.documentElement.hasAttribute('data-gospa-auto')) autoInit();
		});
	} else if (document.documentElement.hasAttribute('data-gospa-auto')) {
		autoInit();
	}
}


