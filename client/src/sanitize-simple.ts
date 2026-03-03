/**
 * Simple sanitizer that removes potentially dangerous HTML elements and attributes.
 * This is a lightweight alternative to DOMPurify for basic sanitization needs.
 * 
 * WARNING: For handling untrusted user input, consider using the full sanitizer
 * (sanitize.ts) which uses DOMPurify for more comprehensive protection.
 * 
 * SECURITY NOTES:
 * - This sanitizer provides BASIC protection only
 * - For production use with untrusted content, use the DOMPurify-based sanitizer
 * - SVG elements are disabled by default due to XSS risks (animate events, foreignObject)
 * - MathML is disabled by default due to script injection risks in some browsers
 * 
 * Removes:
 * - script elements (including inside template elements)
 * - Event handler attributes (onclick, onerror, onload, etc.)
 * - Dangerous elements (iframe, object, embed, form, input, button, select, textarea)
 * - javascript: URLs in href and src attributes
 * - data: URLs in src attributes (can contain malicious content)
 * - SVG elements by default (can contain onload handlers) - can be enabled via allowSVGs
 */

// Event handler attribute pattern - matches on* attributes
const EVENT_HANDLER_PATTERN = /^on/i;

// Dangerous URL schemes
const DANGEROUS_URL_PATTERN = /^(javascript|data|vbscript):/i;

// HTML whitespace for normalization
const HTML_WHITESPACE = /[\u0000-\u0020\u00A0\u1680\u180E\u2000-\u2029\u205F\u3000]/g;

// Elements that should always be removed (security-critical)
const ALWAYS_DANGEROUS_ELEMENTS = new Set([
	'script',
	'iframe',
	'object',
	'embed',
	'applet',
	'form',
	'input',
	'button',
	'select',
	'textarea',
	'meta',
	'link',
	'style',
	'base',
	'template', // Templates can contain hidden scripts
	'portal',   // Experimental element that can load remote content
]);

// Elements that can be optionally allowed (SVG/math - removed by default for security)
const OPTIONALLY_DANGEROUS_ELEMENTS = new Set(['svg', 'math']);

// Attributes that can contain URLs and need checking
const URL_ATTRIBUTES = new Set([
	'href', 'src', 'action', 'formaction', 'xlink:href', 'data', 'poster'
]);

// Dangerous SVG attributes (animation events)
const DANGEROUS_SVG_ATTRS = new Set([
	'onbegin', 'onend', 'onrepeat', 'onload', 'onerror', 'onabort', 'onresize', 'onscroll', 'onunload'
]);

/** Options for creating a simple sanitizer */
export interface SimpleSanitizerOptions {
	/** 
	 * Allow SVG elements (WARNING: SVGs can contain onload handlers - security risk for untrusted content)
	 * Even when enabled, animation event handlers are stripped
	 */
	allowSVGs?: boolean;
	/** 
	 * Allow math elements (WARNING: math elements can contain scripts in some browsers)
	 */
	allowMath?: boolean;
}

/**
 * Normalizes a URL for comparison by decoding entities and lowercasing
 */
function normalizeUrl(url: string): string {
	// Create a temporary element to decode HTML entities
	const textarea = document.createElement('textarea');
	textarea.innerHTML = url;
	return textarea.value.replace(HTML_WHITESPACE, '').toLowerCase();
}

/**
 * Checks if an attribute value contains a dangerous URL
 */
function isDangerousUrl(value: string): boolean {
	const normalized = normalizeUrl(value);
	return DANGEROUS_URL_PATTERN.test(normalized);
}

/**
 * Creates a simple sanitizer function with the specified options.
 * @param options Configuration options for the sanitizer
 * @returns A sanitizer function that takes HTML string and returns sanitized HTML
 */
export function createSimpleSanitizer(options: SimpleSanitizerOptions = {}) {
	const { allowSVGs = false, allowMath = false } = options;

	// Build the set of dangerous elements based on options
	const dangerousElements = new Set(ALWAYS_DANGEROUS_ELEMENTS);
	if (!allowSVGs) {
		dangerousElements.add('svg');
	}
	if (!allowMath) {
		dangerousElements.add('math');
	}

	return function sanitize(html: string): string {
		const div = document.createElement('div');
		
		// SECURITY: Wrap in a template to prevent script execution during parsing
		// This prevents mXSS attacks that exploit innerHTML behavior
		const template = document.createElement('template');
		template.innerHTML = html;
		
		// If template content exists, work with it; otherwise fall back to div
		const container = template.content || div;
		if (!template.content) {
			div.innerHTML = html;
		}

		// Helper function to recursively process nodes
		function processNode(node: Element): void {
			const tagName = node.tagName.toLowerCase();
			
			// Remove dangerous elements
			if (dangerousElements.has(tagName)) {
				node.remove();
				return;
			}

			// Process attributes
			const attributes = Array.from(node.attributes);
			for (const attr of attributes) {
				const attrName = attr.name.toLowerCase();
				const attrValue = attr.value;

				// Remove event handler attributes
				if (EVENT_HANDLER_PATTERN.test(attrName)) {
					node.removeAttribute(attr.name);
					continue;
				}

				// Remove dangerous SVG animation event attributes
				if (DANGEROUS_SVG_ATTRS.has(attrName)) {
					node.removeAttribute(attr.name);
					continue;
				}

				// Check URL attributes for dangerous schemes
				if (URL_ATTRIBUTES.has(attrName)) {
					if (isDangerousUrl(attrValue)) {
						node.removeAttribute(attr.name);
						continue;
					}
				}

				// Remove attributes that could be used for DOM Clobbering
				if (attrName === 'name' || attrName === 'form') {
					node.removeAttribute(attr.name);
					continue;
				}
			}

			// Recursively process child elements
			// Use Array.from to avoid live collection issues when removing
			const children = Array.from(node.children);
			for (const child of children) {
				processNode(child);
			}
		}

		// Process all elements in the container
		const topLevelElements = Array.from(container.children);
		for (const element of topLevelElements) {
			processNode(element);
		}

		// Also process template elements' content
		const templates = container.querySelectorAll('template');
		for (const temp of templates) {
			const tempContent = temp.content;
			if (tempContent) {
				const tempElements = Array.from(tempContent.children);
				for (const element of tempElements) {
					processNode(element);
				}
			}
		}

		// Return sanitized HTML
		if (template.content) {
			return template.innerHTML;
		}
		return div.innerHTML;
	};
}

/**
 * Default simple sanitizer - removes SVGs and math elements for maximum security.
 * Use createSimpleSanitizer() if you need to allow SVGs or math elements.
 */
export const simpleSanitizer = createSimpleSanitizer({ allowSVGs: false, allowMath: false });

/**
 * Sanitize HTML using the default simple sanitizer.
 * This is a convenience function equivalent to simpleSanitizer(html).
 * @param html - Raw HTML string to sanitize
 * @returns Sanitized HTML string
 */
export function sanitize(html: string): string {
	return simpleSanitizer(html);
}
