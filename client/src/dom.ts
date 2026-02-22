// DOM update engine for reactive bindings

import { Rune, Derived, batch } from './state.ts';

// Binding types
export type BindingType = 'text' | 'html' | 'value' | 'checked' | 'class' | 'style' | 'attr' | 'prop';

// Binding configuration
export interface Binding {
	type: BindingType;
	key: string;
	element: Element;
	attribute?: string;
	transform?: (value: unknown) => unknown;
}

// Binding registry
const bindings = new Map<string, Set<Binding>>();
const elementBindings = new WeakMap<Element, Set<Binding>>();

// Create a unique binding ID
let bindingId = 0;
function nextBindingId(): string {
	return `binding-${++bindingId}`;
}

// Register a binding
export function registerBinding(binding: Binding): string {
	const id = nextBindingId();
	
	if (!bindings.has(id)) {
		bindings.set(id, new Set());
	}
	bindings.get(id)!.add(binding);
	
	if (!elementBindings.has(binding.element)) {
		elementBindings.set(binding.element, new Set());
	}
	elementBindings.get(binding.element)!.add(binding);
	
	return id;
}

// Unregister a binding
export function unregisterBinding(id: string): void {
	const bindingSet = bindings.get(id);
	if (bindingSet) {
		bindingSet.forEach(binding => {
			const elemBindings = elementBindings.get(binding.element);
			if (elemBindings) {
				elemBindings.delete(binding);
				if (elemBindings.size === 0) {
					elementBindings.delete(binding.element);
				}
			}
		});
		bindings.delete(id);
	}
}

// Update element based on binding type
function updateElement(binding: Binding, value: unknown): void {
	const { element, type, attribute, transform } = binding;
	const transformedValue = transform ? transform(value) : value;
	
	switch (type) {
		case 'text':
			if (element instanceof HTMLElement || element instanceof SVGElement) {
				element.textContent = String(transformedValue ?? '');
			}
			break;
			
		case 'html':
			if (element instanceof HTMLElement) {
				element.innerHTML = String(transformedValue ?? '');
			}
			break;
			
		case 'value':
			if (element instanceof HTMLInputElement || 
				element instanceof HTMLTextAreaElement || 
				element instanceof HTMLSelectElement) {
				if (element.value !== String(transformedValue ?? '')) {
					element.value = String(transformedValue ?? '');
				}
			}
			break;
			
		case 'checked':
			if (element instanceof HTMLInputElement) {
				element.checked = Boolean(transformedValue);
			}
			break;
			
		case 'class':
			if (element instanceof Element) {
				if (attribute) {
					// Toggle specific class
					if (transformedValue) {
						element.classList.add(attribute);
					} else {
						element.classList.remove(attribute);
					}
				} else if (typeof transformedValue === 'string') {
					// Set class string
					element.className = transformedValue;
				} else if (Array.isArray(transformedValue)) {
					// Set class array
					element.className = transformedValue.join(' ');
				} else if (typeof transformedValue === 'object' && transformedValue !== null) {
					// Toggle classes by object
					Object.entries(transformedValue as Record<string, boolean>).forEach(([cls, enabled]) => {
						if (enabled) {
							element.classList.add(cls);
						} else {
							element.classList.remove(cls);
						}
					});
				}
			}
			break;
			
		case 'style':
			if (element instanceof HTMLElement || element instanceof SVGElement) {
				if (attribute) {
					// Set specific style property
					(element.style as unknown as Record<string, string>)[attribute] = 
						String(transformedValue ?? '');
				} else if (typeof transformedValue === 'string') {
					// Set style string
					element.setAttribute('style', transformedValue);
				} else if (typeof transformedValue === 'object' && transformedValue !== null) {
					// Set styles by object
					Object.entries(transformedValue as Record<string, string>).forEach(([prop, val]) => {
						(element.style as unknown as Record<string, string>)[prop] = val;
					});
				}
			}
			break;
			
		case 'attr':
			if (attribute) {
				if (transformedValue === null || transformedValue === undefined || transformedValue === false) {
					element.removeAttribute(attribute);
				} else if (transformedValue === true) {
					element.setAttribute(attribute, '');
				} else {
					element.setAttribute(attribute, String(transformedValue));
				}
			}
			break;
			
		case 'prop':
			if (attribute && element instanceof HTMLElement) {
				(element as unknown as Record<string, unknown>)[attribute] = transformedValue;
			}
			break;
	}
}

// Bind a rune to an element
export function bindElement<T>(
	element: Element,
	rune: Rune<T>,
	options: Partial<Binding> = {}
): () => void {
	const binding: Binding = {
		type: options.type || 'text',
		key: options.key || '',
		element,
		attribute: options.attribute,
		transform: options.transform
	};
	
	const id = registerBinding(binding);
	
	// Initial update
	updateElement(binding, rune.get());
	
	// Subscribe to changes
	const unsubscribe = rune.subscribe((value) => {
		updateElement(binding, value);
	});
	
	// Return cleanup function
	return () => {
		unsubscribe();
		unregisterBinding(id);
	};
}

// Bind a derived value to an element
export function bindDerived<T>(
	element: Element,
	derived: Derived<T>,
	options: Partial<Binding> = {}
): () => void {
	const binding: Binding = {
		type: options.type || 'text',
		key: options.key || '',
		element,
		attribute: options.attribute,
		transform: options.transform
	};
	
	const id = registerBinding(binding);
	
	// Initial update
	updateElement(binding, derived.get());
	
	// Subscribe to changes
	const unsubscribe = derived.subscribe((value) => {
		updateElement(binding, value);
	});
	
	// Return cleanup function
	return () => {
		unsubscribe();
		unregisterBinding(id);
	};
}

// Create two-way binding for form elements
export function bindTwoWay<T extends string | number | boolean>(
	element: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement,
	rune: Rune<T>
): () => void {
	const isCheckbox = element instanceof HTMLInputElement && element.type === 'checkbox';
	const isRadio = element instanceof HTMLInputElement && element.type === 'radio';
	const isNumber = element instanceof HTMLInputElement && element.type === 'number';
	
	// Initial value
	if (isCheckbox) {
		element.checked = Boolean(rune.get());
	} else {
		element.value = String(rune.get() ?? '');
	}
	
	// Subscribe to rune changes
	const unsubscribe = rune.subscribe((value) => {
		if (isCheckbox) {
			element.checked = Boolean(value);
		} else {
			if (element.value !== String(value ?? '')) {
				element.value = String(value ?? '');
			}
		}
	});
	
	// Listen to input changes
	const inputHandler = () => {
		let newValue: string | number | boolean;
		
		if (isCheckbox) {
			newValue = element.checked;
		} else if (isNumber) {
			newValue = element.value ? parseFloat(element.value) : 0;
		} else {
			newValue = element.value;
		}
		
		batch(() => {
			rune.set(newValue as T);
		});
	};
	
	element.addEventListener('input', inputHandler);
	element.addEventListener('change', inputHandler);
	
	// Return cleanup function
	return () => {
		unsubscribe();
		element.removeEventListener('input', inputHandler);
		element.removeEventListener('change', inputHandler);
	};
}

// Query selector helper with reactive updates
export function querySelector(selector: string): Element | null {
	return document.querySelector(selector);
}

export function querySelectorAll(selector: string): NodeListOf<Element> {
	return document.querySelectorAll(selector);
}

// Create element with bindings
export function createElement<K extends keyof HTMLElementTagNameMap>(
	tag: K,
	attrs: Record<string, unknown> = {},
	children?: (Element | string)[]
): HTMLElementTagNameMap[K] {
	const element = document.createElement(tag);
	
	Object.entries(attrs).forEach(([key, value]) => {
		if (key.startsWith('on') && typeof value === 'function') {
			// Event listener
			const eventName = key.slice(2).toLowerCase();
			element.addEventListener(eventName, value as EventListener);
		} else if (key === 'class') {
			// Class
			if (typeof value === 'string') {
				element.className = value;
			} else if (Array.isArray(value)) {
				element.className = value.join(' ');
			} else if (typeof value === 'object' && value !== null) {
				Object.entries(value as Record<string, boolean>).forEach(([cls, enabled]) => {
					if (enabled) element.classList.add(cls);
				});
			}
		} else if (key === 'style' && typeof value === 'object') {
			// Style object
			Object.entries(value as Record<string, string>).forEach(([prop, val]) => {
				(element.style as unknown as Record<string, string>)[prop] = val;
			});
		} else if (value instanceof Rune) {
			// Reactive binding
			bindElement(element, value, { type: 'attr', attribute: key });
		} else {
			// Static attribute
			element.setAttribute(key, String(value));
		}
	});
	
	if (children) {
		children.forEach(child => {
			if (typeof child === 'string') {
				element.appendChild(document.createTextNode(child));
			} else {
				element.appendChild(child);
			}
		});
	}
	
	return element;
}

// Conditional rendering helper
export function renderIf<T>(
	condition: Rune<boolean> | Derived<boolean>,
	trueRender: () => T,
	falseRender?: () => T
): { element: T | null; cleanup: () => void } {
	let current: T | null = null;
	
	const update = (value: boolean) => {
		if (value) {
			if (!current) {
				current = trueRender();
			}
		} else {
			if (current && falseRender) {
				current = falseRender();
			} else {
				current = null;
			}
		}
	};
	
	// Initial render
	update(condition.get());
	
	// Subscribe to changes
	const unsubscribe = condition.subscribe(update);
	
	return {
		element: current,
		cleanup: () => {
			unsubscribe();
		}
	};
}

// List rendering helper with key tracking
export function renderList<T, K>(
	items: Rune<T[]> | Derived<T[]>,
	render: (item: T, index: number) => Element,
	getKey: (item: T, index: number) => K
): { container: Element; cleanup: () => void } {
	const container = document.createDocumentFragment();
	const containerElement = document.createElement('div');
	container.appendChild(containerElement);
	
	const itemMap = new Map<K, { element: Element; index: number }>();
	
	const update = (newItems: T[]) => {
		const newKeys = new Set<K>();
		
		// Add or update items
		newItems.forEach((item, index) => {
			const key = getKey(item, index);
			newKeys.add(key);
			
			if (!itemMap.has(key)) {
				const element = render(item, index);
				itemMap.set(key, { element, index });
				containerElement.appendChild(element);
			} else {
				const existing = itemMap.get(key)!;
				existing.index = index;
				// Reorder if needed
				if (containerElement.children[index] !== existing.element) {
					containerElement.insertBefore(existing.element, containerElement.children[index] || null);
				}
			}
		});
		
		// Remove items not in new list
		itemMap.forEach((value, key) => {
			if (!newKeys.has(key)) {
				value.element.remove();
				itemMap.delete(key);
			}
		});
	};
	
	// Initial render
	update(items.get());
	
	// Subscribe to changes
	const unsubscribe = items.subscribe(update);
	
	return {
		container: containerElement,
		cleanup: () => {
			unsubscribe();
			itemMap.clear();
		}
	};
}
