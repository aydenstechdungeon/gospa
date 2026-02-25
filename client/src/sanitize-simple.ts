/**
 * Simple sanitizer that removes potentially dangerous HTML elements and attributes.
 * This is a lightweight alternative to DOMPurify for basic sanitization needs.
 * 
 * WARNING: For handling untrusted user input, consider using the full sanitizer
 * (sanitize.ts) which uses DOMPurify for more comprehensive protection.
 * 
 * Removes:
 * - script elements
 * - Event handler attributes (onclick, onerror, onload, etc.)
 * - Dangerous elements (iframe, object, embed, form, input, button, select, textarea)
 * - javascript: URLs in href and src attributes
 * - data: URLs in src attributes (can contain malicious content)
 * - SVG elements by default (can contain onload handlers) - can be enabled via allowSVGs
 */

// Event handler attribute pattern - matches on* attributes
const EVENT_HANDLER_PATTERN = /^on/i;

// Dangerous URL schemes
const DANGEROUS_URL_PATTERN = /^\s*(javascript|data|vbscript):/i;

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
]);

// Elements that can be optionally allowed (SVG/math - removed by default for security)
const OPTIONALLY_DANGEROUS_ELEMENTS = new Set(['svg', 'math']);

// Attributes that can contain URLs and need checking
const URL_ATTRIBUTES = new Set(['href', 'src', 'action', 'formaction', 'xlink:href']);

/** Options for creating a simple sanitizer */
export interface SimpleSanitizerOptions {
	/** Allow SVG elements (WARNING: SVGs can contain onload handlers - security risk for untrusted content) */
	allowSVGs?: boolean;
	/** Allow math elements (WARNING: math elements can contain scripts in some browsers) */
	allowMath?: boolean;
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
		div.innerHTML = html;

		// Remove dangerous elements
		for (const tagName of dangerousElements) {
			const elements = div.getElementsByTagName(tagName);
			// Iterate backwards since collection is live
			for (let i = elements.length - 1; i >= 0; i--) {
				elements[i].parentNode?.removeChild(elements[i]);
			}
		}

		// Clean all remaining elements
		const allElements = div.getElementsByTagName('*');
		for (let i = 0; i < allElements.length; i++) {
			const element = allElements[i];
			
			// Remove event handler attributes
			const attributes = Array.from(element.attributes);
			for (const attr of attributes) {
				// Remove on* event handlers
				if (EVENT_HANDLER_PATTERN.test(attr.name)) {
					element.removeAttribute(attr.name);
					continue;
				}
				
				// Check URL attributes for dangerous schemes
				if (URL_ATTRIBUTES.has(attr.name.toLowerCase())) {
					if (DANGEROUS_URL_PATTERN.test(attr.value)) {
						element.removeAttribute(attr.name);
					}
				}
			}
		}

		return div.innerHTML;
	};
}

/**
 * Default simple sanitizer - removes SVGs and math elements for maximum security.
 * Use createSimpleSanitizer() if you need to allow SVGs or math elements.
 */
export const simpleSanitizer = createSimpleSanitizer({ allowSVGs: false, allowMath: false });
