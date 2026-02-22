// Event handling system for reactive bindings

import { Rune } from './state.ts';

// Event handler types
export type EventHandler<E = Event> = (event: E) => void | Promise<void>;
export type ModifierHandler<E = Event> = (event: E, handler: EventHandler<E>) => void | Promise<void>;

// Event modifiers
export type EventModifier = 'prevent' | 'stop' | 'capture' | 'once' | 'passive' | 'self';

// Modifier implementations
const modifiers: Record<EventModifier, ModifierHandler> = {
	prevent: (event, handler) => {
		event.preventDefault();
		return handler(event);
	},
	stop: (event, handler) => {
		event.stopPropagation();
		return handler(event);
	},
	capture: (event, handler) => handler(event),
	once: (event, handler) => handler(event),
	passive: (event, handler) => handler(event),
	self: (event, handler) => {
		if (event.target === event.currentTarget) {
			return handler(event);
		}
	}
};

// Event configuration
export interface EventConfig {
	event: string;
	handler: EventHandler;
	modifiers?: EventModifier[];
	options?: AddEventListenerOptions;
}

// Event listener registry for cleanup
const listenerRegistry = new WeakMap<EventTarget, Map<string, Set<EventListener>>>();

// Create wrapped handler with modifiers
function createWrappedHandler<E extends Event>(
	handler: EventHandler<E>,
	mods: EventModifier[]
): EventHandler<E> {
	return (event: E) => {
		// Apply modifiers in order
		for (const mod of mods) {
			if (mod === 'capture' || mod === 'once' || mod === 'passive') {
				continue; // These are handled by addEventListener options
			}
			const modHandler = modifiers[mod];
			modHandler(event, handler as EventHandler);
		}
		
		// Call handler directly if no active modifiers
		const activeMods = mods.filter(m => !['capture', 'once', 'passive'].includes(m));
		if (activeMods.length === 0) {
			return handler(event);
		}
	};
}

// Parse event string like "click:prevent:stop"
export function parseEventString(eventStr: string): { event: string; modifiers: EventModifier[] } {
	const parts = eventStr.split(':');
	const event = parts[0];
	const mods = parts.slice(1) as EventModifier[];
	
	return { event, modifiers: mods };
}

// Add event listener with modifiers
export function on<K extends keyof HTMLElementEventMap>(
	target: EventTarget,
	eventStr: string,
	handler: EventHandler<HTMLElementEventMap[K] extends Event ? HTMLElementEventMap[K] : Event>
): () => void {
	const { event, modifiers: mods } = parseEventString(eventStr);
	
	// Build options from modifiers
	const options: AddEventListenerOptions = {
		capture: mods.includes('capture'),
		once: mods.includes('once'),
		passive: mods.includes('passive')
	};
	
	// Create wrapped handler
	const wrappedHandler = createWrappedHandler(handler as EventHandler, mods);
	
	// Add listener
	target.addEventListener(event, wrappedHandler as EventListener, options);
	
	// Track for cleanup
	if (!listenerRegistry.has(target)) {
		listenerRegistry.set(target, new Map());
	}
	const targetMap = listenerRegistry.get(target)!;
	if (!targetMap.has(eventStr)) {
		targetMap.set(eventStr, new Set());
	}
	targetMap.get(eventStr)!.add(wrappedHandler as EventListener);
	
	// Return cleanup function
	return () => {
		target.removeEventListener(event, wrappedHandler as EventListener, options);
		const set = targetMap.get(eventStr);
		if (set) {
			set.delete(wrappedHandler as EventListener);
			if (set.size === 0) {
				targetMap.delete(eventStr);
			}
		}
	};
}

// Remove all listeners for a target
export function offAll(target: EventTarget): void {
	const targetMap = listenerRegistry.get(target);
	if (!targetMap) return;
	
	for (const [eventStr, listeners] of targetMap) {
		const { event, modifiers: mods } = parseEventString(eventStr);
		const options: AddEventListenerOptions = {
			capture: mods.includes('capture')
		};
		
		for (const listener of listeners) {
			target.removeEventListener(event, listener, options);
		}
	}
	
	listenerRegistry.delete(target);
}

// Debounce event handler
export function debounce<E extends Event>(
	handler: EventHandler<E>,
	wait: number
): EventHandler<E> {
	let timeoutId: ReturnType<typeof setTimeout> | null = null;
	
	return (event: E) => {
		if (timeoutId) {
			clearTimeout(timeoutId);
		}
		
		timeoutId = setTimeout(() => {
			handler(event);
			timeoutId = null;
		}, wait);
	};
}

// Throttle event handler
export function throttle<E extends Event>(
	handler: EventHandler<E>,
	limit: number
): EventHandler<E> {
	let inThrottle = false;
	
	return (event: E) => {
		if (!inThrottle) {
			handler(event);
			inThrottle = true;
			setTimeout(() => {
				inThrottle = false;
			}, limit);
		}
	};
}

// Bind event to rune update
export function bindEvent<E extends Event>(
	target: EventTarget,
	eventStr: string,
	rune: Rune<unknown>,
	transformer: (event: E) => unknown
): () => void {
	return on(target, eventStr, (event) => {
		const value = transformer(event as E);
		rune.set(value);
	});
}

// Common event transformers
export const transformers = {
	value: (event: Event) => (event.target as HTMLInputElement).value,
	checked: (event: Event) => (event.target as HTMLInputElement).checked,
	numberValue: (event: Event) => Number((event.target as HTMLInputElement).value),
	files: (event: Event) => (event.target as HTMLInputElement).files,
	formData: (event: Event) => {
		event.preventDefault();
		return new FormData(event.target as HTMLFormElement);
	}
};

// Event delegation helper
export function delegate(
	root: EventTarget,
	selector: string,
	eventStr: string,
	handler: EventHandler
): () => void {
	const { event, modifiers: mods } = parseEventString(eventStr);
	
	const delegatedHandler = (e: Event) => {
		const target = e.target as Element;
		const matched = target.closest(selector);
		
		if (matched) {
			const wrappedHandler = createWrappedHandler(handler, mods);
			wrappedHandler(e);
		}
	};
	
	const options: AddEventListenerOptions = {
		capture: mods.includes('capture'),
		passive: mods.includes('passive')
	};
	
	root.addEventListener(event, delegatedHandler, options);
	
	return () => {
		root.removeEventListener(event, delegatedHandler, options);
	};
}

// Keyboard event helpers
export function onKey(
	keys: string | string[],
	handler: EventHandler<KeyboardEvent>,
	options?: { preventDefault?: boolean }
): EventHandler<KeyboardEvent> {
	const keyArray = Array.isArray(keys) ? keys : [keys];
	
	return (event: KeyboardEvent) => {
		if (keyArray.includes(event.key)) {
			if (options?.preventDefault) {
				event.preventDefault();
			}
			handler(event);
		}
	};
}

// Common key shortcuts
export const keys = {
	enter: 'Enter',
	escape: 'Escape',
	tab: 'Tab',
	space: ' ',
	arrowUp: 'ArrowUp',
	arrowDown: 'ArrowDown',
	arrowLeft: 'ArrowLeft',
	arrowRight: 'ArrowRight'
};
