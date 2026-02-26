/**
 * Lazy-loaded DOMPurify sanitizer module
 * 
 * DOMPurify is only loaded when sanitization is actually needed,
 * reducing initial bundle size by ~20KB minified.
 */

// Purify config - defined upfront, no runtime cost
const PURIFY_CONFIG = {
	ALLOWED_TAGS: [
		// Semantic HTML5
		'header', 'footer', 'nav', 'main', 'section', 'article', 'aside', 'figure', 'figcaption',
		// Interactive elements
		'button', 'details', 'summary', 'dialog',
		// Standard elements
		'a', 'b', 'br', 'code', 'div', 'em', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
		'i', 'img', 'li', 'ol', 'p', 'pre', 'span', 'strong', 'table', 'tbody',
		'td', 'th', 'thead', 'tr', 'u', 'ul', 'blockquote', 'hr', 'sub', 'sup',
		'small', 'mark', 'del', 'ins', 'abbr', 'cite', 'dfn', 'kbd', 'samp', 'var',
		'svg', 'path', 'rect', 'circle', 'line', 'polyline', 'polygon', 'ellipse', 'style',
		// Additional useful elements
		'input', 'textarea', 'select', 'option', 'optgroup', 'label', 'form', 'fieldset', 'legend',
		'dl', 'dt', 'dd', 'address', 'time', 'picture', 'source', 'audio', 'video', 'track',
		'canvas', 'data', 'meter', 'progress', 'output'
	],
	ALLOWED_ATTR: [
		'href', 'src', 'alt', 'title', 'class', 'id', 'target', 'rel', 'style',
		'xmlns', 'viewBox', 'fill', 'stroke', 'stroke-width', 'stroke-linecap', 'stroke-linejoin',
		'd', 'x', 'y', 'width', 'height', 'rx', 'ry', 'points', 'cx', 'cy', 'r'
	],
	ALLOW_DATA_ATTR: true,
	FORBID_TAGS: ['script', 'iframe', 'object', 'embed', 'form', 'meta', 'link', 'base', 'applet', 'frame', 'frameset'],
	FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur', 'formaction', 'xlink:href'],
	ADD_ATTR: ['target'],
	FORCE_BODY: true
};

// Cached DOMPurify instance - lazy loaded
let domPurifyInstance: typeof import('dompurify').default | null = null;
let domPurifyPromise: Promise<typeof import('dompurify').default> | null = null;

/**
 * Get DOMPurify instance (lazy loaded)
 * Returns a promise that resolves to the DOMPurify instance
 */
async function getDOMPurify(): Promise<typeof import('dompurify').default> {
	if (domPurifyInstance) {
		return domPurifyInstance;
	}

	if (domPurifyPromise) {
		return domPurifyPromise;
	}

	domPurifyPromise = import('dompurify').then(module => {
		domPurifyInstance = module.default;
		return domPurifyInstance;
	});

	return domPurifyPromise;
}

/**
 * Async sanitization - preferred for most use cases
 * Loads DOMPurify on first call, then caches for subsequent calls
 */
export async function sanitize(html: string): Promise<string> {
	const purify = await getDOMPurify();
	return purify.sanitize(html, PURIFY_CONFIG) as string;
}

/**
 * Sync sanitization - only use if DOMPurify is already loaded
 * Returns original HTML if DOMPurify not yet loaded (use with caution)
 */
export function sanitizeSync(html: string): string {
	if (!domPurifyInstance) {
		console.warn('[gospa] sanitizeSync: DOMPurify not loaded');
		// Escape basic HTML entities as fallback
		return html.replace(/[&<>"']/g, (m) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m]!));
	}
	return domPurifyInstance.sanitize(html, PURIFY_CONFIG) as string;
}

/**
 * Check if DOMPurify is loaded and ready
 */
export function isSanitizerReady(): boolean {
	return domPurifyInstance !== null;
}

/**
 * Preload DOMPurify - call during idle time for faster first sanitization
 */
export function preloadSanitizer(): void {
	if (!domPurifyInstance && !domPurifyPromise) {
		getDOMPurify().catch(err => {
			console.error('[gospa] Failed to preload DOMPurify:', err);
		});
	}
}

/**
 * Async sanitizer for use with dom.ts setSanitizer
 * This is the preferred sanitizer function - it always properly sanitizes
 * by waiting for DOMPurify to load if needed.
 */
export function domPurifySanitizer(html: string): string | Promise<string> {
	// If DOMPurify is already loaded, use sync path for performance
	if (domPurifyInstance) {
		return domPurifyInstance.sanitize(html, PURIFY_CONFIG) as string;
	}
	// Otherwise use async path - this ensures proper sanitization
	return sanitize(html);
}

// Export types for consumers
export type { PURIFY_CONFIG as PurifyConfigType };
