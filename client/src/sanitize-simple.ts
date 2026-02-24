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
 * - SVG elements (can contain onload handlers)
 */

// Event handler attribute pattern - matches on* attributes
const EVENT_HANDLER_PATTERN = /^on/i;

// Dangerous URL schemes
const DANGEROUS_URL_PATTERN = /^\s*(javascript|data|vbscript):/i;

// Elements that should be completely removed
const DANGEROUS_ELEMENTS = new Set([
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
	'svg',
	'math',
]);

// Attributes that can contain URLs and need checking
const URL_ATTRIBUTES = new Set(['href', 'src', 'action', 'formaction', 'xlink:href']);

export function simpleSanitizer(html: string): string {
	const div = document.createElement('div');
	div.innerHTML = html;

	// Remove dangerous elements
	for (const tagName of DANGEROUS_ELEMENTS) {
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
}
