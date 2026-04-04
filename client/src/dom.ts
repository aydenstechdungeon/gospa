import { Rune, batch } from "./state.ts";
import type { Derived } from "./state.ts";
import { handlers } from "./dom/handlers.ts";

/**
 * Declare global debug constant for build-time stripping.
 */
declare global {
  var GOSPA_DEBUG: boolean; // eslint-disable-line no-var
}

// === RAF-batched DOM Updates ===
let pendingDOMUpdates: (() => void)[] = [];
let rafScheduled = false;
let rafId: number | null = null;

function scheduleDOMUpdate(update: () => void): void {
  pendingDOMUpdates.push(update);
  if (!rafScheduled) {
    rafScheduled = true;
    rafId = requestAnimationFrame(flushDOMUpdates);
  }
}

function flushDOMUpdates(): void {
  const updates = pendingDOMUpdates;
  pendingDOMUpdates = [];
  rafScheduled = false;
  rafId = null;

  for (const update of updates) {
    try {
      update();
    } catch (error) {
      if (typeof GOSPA_DEBUG !== "undefined" && GOSPA_DEBUG) {
        console.error("[GoSPA] DOM update failed:", error);
      }
    }
  }
}

export function cancelPendingDOMUpdates(): void {
  if (rafId !== null) {
    cancelAnimationFrame(rafId);
    rafId = null;
  }
  pendingDOMUpdates = [];
  rafScheduled = false;
}

export function flushDOMUpdatesNow(): void {
  if (rafScheduled) {
    if (rafId !== null) {
      cancelAnimationFrame(rafId);
      rafId = null;
    }
    flushDOMUpdates();
  }
}

// Binding types
export type BindingType = string; // Modular registry allows any string

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
const elementVersions = new WeakMap<Element, number>();

let bindingId = 0;
function nextBindingId(): string {
  return `binding-${++bindingId}`;
}

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

export function unregisterBinding(id: string): void {
  const bindingSet = bindings.get(id);
  if (bindingSet) {
    bindingSet.forEach((binding) => {
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

async function updateElement(binding: Binding, value: unknown): Promise<void> {
  const { element, type, attribute, transform } = binding;
  const transformedValue = transform ? transform(value) : value;

  const version = (elementVersions.get(element) || 0) + 1;
  elementVersions.set(element, version);

  const handler = handlers[type];
  if (handler) {
    scheduleDOMUpdate(() => {
      const result = handler(
        element,
        transformedValue,
        attribute,
        version,
        elementVersions,
      );
      if (result instanceof Promise) {
        result.catch((error) => {
          if (typeof GOSPA_DEBUG !== "undefined" && GOSPA_DEBUG) {
            console.error(`[GoSPA] Binding '${type}' failed:`, error);
          }
        });
      }
    });
  } else if (typeof GOSPA_DEBUG !== "undefined" && GOSPA_DEBUG) {
    console.warn(`[GoSPA] No handler registered for binding type: ${type}`);
  }
}

export function bindElement<T>(
  element: Element,
  rune: Rune<T>,
  options: Partial<Binding> = {},
): () => void {
  const binding: Binding = {
    type: options.type || "text",
    key: options.key || "",
    element,
    attribute: options.attribute,
    transform: options.transform,
  };

  const id = registerBinding(binding);
  updateElement(binding, rune.get());

  const unsubscribe = rune.subscribe((value) => {
    updateElement(binding, value);
  });

  return () => {
    unsubscribe();
    unregisterBinding(id);
  };
}

export function bindDerived<T>(
  element: Element,
  derived: Derived<T>,
  options: Partial<Binding> = {},
): () => void {
  const binding: Binding = {
    type: options.type || "text",
    key: options.key || "",
    element,
    attribute: options.attribute,
    transform: options.transform,
  };

  const id = registerBinding(binding);
  updateElement(binding, derived.get());

  const unsubscribe = derived.subscribe((value) => {
    updateElement(binding, value);
  });

  return () => {
    unsubscribe();
    unregisterBinding(id);
  };
}

export function bindTwoWay<T extends string | number | boolean>(
  element: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement,
  rune: Rune<T>,
): () => void {
  const isCheckbox =
    element instanceof HTMLInputElement && element.type === "checkbox";
  const isNumber =
    element instanceof HTMLInputElement && element.type === "number";

  if (isCheckbox) {
    element.checked = Boolean(rune.get());
  } else {
    element.value = String(rune.get() ?? "");
  }

  const unsubscribe = rune.subscribe((value) => {
    if (isCheckbox) {
      element.checked = Boolean(value);
    } else if (element.value !== String(value ?? "")) {
      element.value = String(value ?? "");
    }
  });

  const inputHandler = () => {
    let newValue: string | number | boolean;
    if (isCheckbox) newValue = element.checked;
    else if (isNumber) newValue = element.value ? parseFloat(element.value) : 0;
    else newValue = element.value;

    batch(() => rune.set(newValue as T));
  };

  element.addEventListener("input", inputHandler);
  element.addEventListener("change", inputHandler);

  return () => {
    unsubscribe();
    element.removeEventListener("input", inputHandler);
    element.removeEventListener("change", inputHandler);
  };
}

// Re-export list rendering utilities
export { renderIf, renderList } from "./dom/lists.ts";

/**
 * Create element with bindings.
 */
export function createElement<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  attrs: Record<string, unknown> = {},
  children?: (Element | string)[],
): HTMLElementTagNameMap[K] {
  const element = document.createElement(tag);

  Object.entries(attrs).forEach(([key, value]) => {
    if (key.startsWith("on") && typeof value === "function") {
      const eventName = key.slice(2).toLowerCase();
      element.addEventListener(eventName, value as EventListener);
    } else if (key === "class") {
      handlers.class(element, value);
    } else if (key === "style") {
      handlers.style(element, value);
    } else if (value instanceof Rune) {
      bindElement(element, value, { type: "attr", attribute: key });
    } else {
      element.setAttribute(key, String(value));
    }
  });

  if (children) {
    children.forEach((child) => {
      if (typeof child === "string") {
        element.appendChild(document.createTextNode(child));
      } else {
        element.appendChild(child);
      }
    });
  }

  return element;
}

// Note: addClass, removeClass, attr, data, querySelectorAll, find, findAll
// have been removed as they are native Web API "sugar".
