/**
 * Registry-based DOM binding handlers to allow tree-shaking unused handlers.
 */

export type BindingHandler = (
  element: Element,
  value: unknown,
  attribute?: string,
  version?: number,
  elementVersions?: WeakMap<Element, number>,
) => void | Promise<void>;

import { toHTMLString } from "../html-policy.ts";

export const handlers: Record<string, BindingHandler> = {
  text: (element, value) => {
    if (element instanceof HTMLElement || element instanceof SVGElement) {
      element.textContent = String(value ?? "");
    }
  },

  html: (element, value, _attr, version, elementVersions) => {
    if (element instanceof HTMLElement) {
      if (!elementVersions || elementVersions.get(element) === version) {
        element.innerHTML = toHTMLString(value);
      }
    }
  },

  value: (element, value) => {
    if (
      element instanceof HTMLInputElement ||
      element instanceof HTMLTextAreaElement ||
      element instanceof HTMLSelectElement
    ) {
      if (element.value !== String(value ?? "")) {
        element.value = String(value ?? "");
      }
    }
  },

  checked: (element, value) => {
    if (element instanceof HTMLInputElement) {
      element.checked = Boolean(value);
    }
  },

  class: (element, value, attribute) => {
    if (element instanceof Element) {
      if (attribute) {
        if (value) {
          element.classList.add(attribute);
        } else {
          element.classList.remove(attribute);
        }
      } else if (typeof value === "string") {
        element.className = value;
      } else if (Array.isArray(value)) {
        element.className = value.join(" ");
      } else if (typeof value === "object" && value !== null) {
        Object.entries(value as Record<string, boolean>).forEach(
          ([cls, enabled]) => {
            if (enabled) {
              element.classList.add(cls);
            } else {
              element.classList.remove(cls);
            }
          },
        );
      }
    }
  },

  style: (element, value, attribute) => {
    if (element instanceof HTMLElement || element instanceof SVGElement) {
      if (attribute) {
        (element.style as unknown as Record<string, string>)[attribute] =
          String(value ?? "");
      } else if (typeof value === "string") {
        element.setAttribute("style", value);
      } else if (typeof value === "object" && value !== null) {
        Object.entries(value as Record<string, string>).forEach(
          ([prop, val]) => {
            (element.style as unknown as Record<string, string>)[prop] = val;
          },
        );
      }
    }
  },

  attr: (element, value, attribute) => {
    if (attribute) {
      if (value === null || value === undefined || value === false) {
        element.removeAttribute(attribute);
      } else if (value === true) {
        element.setAttribute(attribute, "");
      } else {
        element.setAttribute(attribute, String(value));
      }
    }
  },

  prop: (element, value, attribute) => {
    if (attribute && element instanceof HTMLElement) {
      (element as unknown as Record<string, unknown>)[attribute] = value;
    }
  },
};

/**
 * Register a custom binding handler.
 */
export function registerHandler(type: string, handler: BindingHandler): void {
  handlers[type] = handler;
}
