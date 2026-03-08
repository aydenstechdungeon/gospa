// GoSPA Client-side Navigation
// Enables SPA-style navigation without full page reloads

// Navigation state
const state = {
	currentPath: window.location.pathname,
	isNavigating: false,
	pendingNavigation: null as Promise<boolean> | null,
};

// Navigation options
export interface NavigationOptions {
	replace?: boolean;
	scrollToTop?: boolean;
	preserveState?: boolean;
}

// Navigation event handlers
type NavigationCallback = (path: string) => void;
const beforeNavCallbacks: Set<NavigationCallback> = new Set();
const afterNavCallbacks: Set<NavigationCallback> = new Set();

// Register callbacks
export function onBeforeNavigate(cb: NavigationCallback): () => void {
	beforeNavCallbacks.add(cb);
	return () => beforeNavCallbacks.delete(cb);
}

export function onAfterNavigate(cb: NavigationCallback): () => void {
	afterNavCallbacks.add(cb);
	return () => afterNavCallbacks.delete(cb);
}

// Common file extensions that should be ignored by the SPA router
let IGNORED_EXTENSIONS = [
	// Text, Documents & E-books
	'txt', 'md', 'pdf', 'csv', 'tsv', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'rtf', 'ods', 'odt', 'odp', 'epub', 'mobi', 'azw3', 'djvu',
	// Data, Config & Logs
	'json', 'xml', 'sql', 'sqlite', 'db', 'yaml', 'yml', 'toml', 'ini', 'log', 'env', 'bak', 'tmp', 'swp', 'lock',
	// Images
	'png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'avif', 'heic', 'heif', 'ico', 'bmp', 'tif', 'tiff', 'apng', 'jp2', 'jxl',
	// Audio & Playlists
	'mp3', 'wav', 'ogg', 'm4a', 'flac', 'aac', 'opus', 'm3u', 'm3u8', 'mid', 'midi', 'aif', 'aiff',
	// Video
	'mp4', 'webm', 'mov', 'mkv', 'avi', 'wmv', 'ogv', 'm4v', '3gp', '3g2', 'ts',
	// 3D & AR
	'glb', 'gltf', 'obj', 'stl', 'usdz',
	// Archives
	'zip', 'tar', 'gz', '7z', 'rar', 'bz2', 'xz', 'tgz', 'br', 'zst', 'lz', 'lzma', 'cab', 'ar', 'cpio',
	// Executables, Packages & Scripts
	'iso', 'dmg', 'bin', 'exe', 'msi', 'appimage', 'deb', 'rpm', 'apk', 'jar', 'sh', 'bat', 'cmd', 'ps1',
	// Web Assets & PWA
	'js', 'jsx', 'mjs', 'cjs', 'ts', 'tsx', 'css', 'scss', 'sass', 'less', 'wasm', 'map', 'webmanifest',
	// Fonts
	'woff', 'woff2', 'ttf', 'otf', 'eot',
	// Certs & Security
	'pem', 'crt', 'cer', 'der', 'key',
	// Events & Contacts
	'ics', 'vcf'
];

let IGNORED_EXTENSIONS_SET = new Set<string>();

/**
 * Configure the ignored extensions for the SPA router
 */
export function setIgnoredExtensions(extensions: string[]): void {
	if (!extensions || extensions.length === 0) return;
	IGNORED_EXTENSIONS = extensions;

	IGNORED_EXTENSIONS_SET.clear();
	for (const ext of extensions) {
		// Handle legacy regex-like shorthand (e.g., 'docx?', 'jpe?g') for backwards compatibility
		const i = ext.indexOf('?');
		if (i !== -1) {
			// e.g. 'jpe?g' -> 'jpeg' (withChar) & 'jpg' (withoutChar)
			const withChar = ext.slice(0, i) + ext.slice(i + 1);
			const withoutChar = ext.slice(0, i - 1) + ext.slice(i + 1);
			IGNORED_EXTENSIONS_SET.add(withChar.toLowerCase());
			IGNORED_EXTENSIONS_SET.add(withoutChar.toLowerCase());
		} else {
			IGNORED_EXTENSIONS_SET.add(ext.toLowerCase());
		}
	}
}

// Initialize the optimized lookup set
setIgnoredExtensions(IGNORED_EXTENSIONS);

/**
 * Append to the current list of ignored extensions
 */
export function appendIgnoredExtensions(extensions: string[]): void {
	if (!extensions || extensions.length === 0) return;
	setIgnoredExtensions([...IGNORED_EXTENSIONS, ...extensions]);
}

// Check if a link is internal (same origin)
function isInternalLink(link: HTMLAnchorElement): boolean {
	const href = link.getAttribute('href');

	// Skip obviously non-navigable schemas and hashes instantly
	if (!href ||
		href.startsWith('#') ||
		href.startsWith('javascript:') ||
		href.startsWith('mailto:') ||
		href.startsWith('tel:') ||
		href.startsWith('sms:') ||
		href.startsWith('blob:') ||
		href.startsWith('data:')
	) {
		return false;
	}

	let urlObj: URL;
	try {
		// Parse the URL exactly once
		urlObj = new URL(href, window.location.origin);
	} catch {
		return false;
	}

	// 1. External origin check
	if (urlObj.origin !== window.location.origin) {
		return false;
	}

	// 2. Target & Download attributes
	if (link.hasAttribute('data-external') ||
		link.hasAttribute('download') ||
		link.getAttribute('target') === '_blank') {
		return false;
	}

	// 3. File extension check (optimized trailing segment lookahead)
	const pathname = urlObj.pathname;
	const lastSegment = pathname.slice(pathname.lastIndexOf('/') + 1);

	const dot = lastSegment.lastIndexOf('.');
	if (dot !== -1 && dot < lastSegment.length - 1) {
		const ext = lastSegment.slice(dot + 1).toLowerCase();
		if (IGNORED_EXTENSIONS_SET.has(ext)) {
			return false;
		}
	}

	return true;
}

// Page data type
interface PageData {
	content: string;
	title: string;
	head: string;
	isDocsPage: boolean;
}

// Check if path is a docs page
function isDocsPage(path: string): boolean {
	return path.startsWith('/docs');
}

// Prefetch cache
const prefetchCache = new Map<string, PageData>();

// Fetch page content from server
async function fetchPageFromServer(path: string): Promise<PageData | null> {
	try {
		const response = await fetch(path, {
			headers: {
				'X-Requested-With': 'GoSPA-Navigate',
				'Accept': 'text/html',
			},
		});

		if (!response.ok) {
			console.error('[GoSPA] Navigation failed:', response.status);
			return null;
		}

		const html = await response.text();

		// Parse the HTML response
		const parser = new DOMParser();
		const doc = parser.parseFromString(html, 'text/html');

		// Check if this is a docs page
		const isDocsPage = !!doc.querySelector('[data-gospa-docs-content]');

		// Check if current page is docs
		const currentIsDocsPage = !!document.querySelector('[data-gospa-docs-content]');

		// Extract content based on page type transition
		let content: string;

		if (isDocsPage && currentIsDocsPage) {
			// Docs -> Docs: Only extract inner content (sidebar persists)
			const docsContentEl = doc.querySelector('[data-gospa-docs-content]');
			content = docsContentEl ? docsContentEl.innerHTML : '';
		} else if (isDocsPage && !currentIsDocsPage) {
			// Non-docs -> Docs: Extract FULL page content including sidebar
			const pageContentEl = doc.querySelector('[data-gospa-page-content]');
			content = pageContentEl ? pageContentEl.innerHTML : doc.body.innerHTML;
		} else if (!isDocsPage && currentIsDocsPage) {
			// Docs -> Non-docs: Extract FULL page content (remove sidebar)
			const pageContentEl = doc.querySelector('[data-gospa-page-content]');
			content = pageContentEl ? pageContentEl.innerHTML : doc.body.innerHTML;
		} else {
			// Non-docs -> Non-docs: Standard content replacement
			const contentEl = doc.querySelector('[data-gospa-page-content]');
			const rootEl = doc.querySelector('[data-gospa-root]');
			const mainEl = doc.querySelector('main');
			content = contentEl ? contentEl.innerHTML :
				(rootEl ? rootEl.innerHTML : (mainEl ? mainEl.innerHTML : doc.body.innerHTML));
		}

		// Extract title
		const title = doc.querySelector('title')?.textContent || '';

		// Extract head elements (for head management)
		const headEl = doc.querySelector('head');
		const head = headEl ? headEl.innerHTML : '';

		return { content, title, head, isDocsPage };
	} catch (error) {
		console.error('[GoSPA] Navigation error:', error);
		return null;
	}
}

// Get page data (from cache or server)
async function getPageData(path: string): Promise<PageData | null> {
	const cached = prefetchCache.get(path);
	if (cached) {
		prefetchCache.delete(path);
		return cached;
	}
	return fetchPageFromServer(path);
}

// Content is trusted - Templ auto-escapes on the server
// For user-generated content, use 'gospa/runtime-secure' which includes DOMPurify
async function prepareContent(html: string): Promise<string> {
	// Return HTML as-is - server is trusted, CSP provides XSS protection
	return html;
}

// Update the DOM with new content
async function updateDOM(data: PageData): Promise<void> {
	// Update title
	if (data.title) {
		document.title = data.title;
	}

	const pageContent = await prepareContent(data.content);

	// Check current page type
	const currentIsDocsPage = !!document.querySelector('[data-gospa-docs-content]');

	// Handle page transitions
	if (data.isDocsPage && currentIsDocsPage) {
		// Docs -> Docs: Only replace the inner docs content (sidebar persists)
		const docsContentEl = document.querySelector('[data-gospa-docs-content]');
		if (docsContentEl) {
			docsContentEl.innerHTML = pageContent;
		}
	} else if ((data.isDocsPage && !currentIsDocsPage) || (!data.isDocsPage && currentIsDocsPage)) {
		// Cross-type transition: Replace FULL page content (includes sidebar addition/removal)
		const contentEl = document.querySelector('[data-gospa-page-content]');
		if (contentEl) {
			contentEl.innerHTML = pageContent;
		}
	} else {
		// Non-docs -> Non-docs: Standard full content replacement
		const contentEl = document.querySelector('[data-gospa-page-content]');
		const rootEl = document.querySelector('[data-gospa-root]');

		if (contentEl) {
			// Preferred: update only the content region, preserving header/footer
			contentEl.innerHTML = pageContent;
		} else if (rootEl) {
			// Fallback: update entire root (legacy behavior)
			rootEl.innerHTML = pageContent;
		} else {
			// Last resort: update main or body
			const mainEl = document.querySelector('main');
			if (mainEl) {
				mainEl.innerHTML = pageContent;
			} else {
				document.body.innerHTML = pageContent;
			}
		}
	}

	// Update head (managed head elements)
	updateHead(data.head);

	// Re-initialize runtime for new content
	await initNewContent();
}

// Update head elements - smart reconciliation to avoid CSS flashes
// and clean up elements that are no longer needed
function updateHead(headHtml: string): void {
	// Parse head HTML to extract elements
	const parser = new DOMParser();
	const doc = parser.parseFromString(`<html><head>${headHtml}</head></html>`, 'text/html');
	const newHead = doc.querySelector('head');

	if (!newHead) return;

	// 1. Update title explicitly if it changed
	const newTitle = doc.querySelector('title')?.textContent;
	if (newTitle && newTitle !== document.title) {
		document.title = newTitle;
	}

	// Track which GoSPA-managed elements are still needed
	const neededSelectors = new Set<string>();

	// 2. Smart reconciliation for link tags (CSS)
	// Never remove existing stylesheets to avoid FOUC (Flash of Unstyled Content)
	const newLinkElements = Array.from(newHead.querySelectorAll('link'));

	newLinkElements.forEach(newEl => {
		const href = newEl.getAttribute('href');
		const rel = newEl.getAttribute('rel');

		// Build a unique selector for tracking
		const selector = href ? `link[href="${href}"]` : null;
		if (selector) neededSelectors.add(selector);

		// Check if this link already exists in the document
		const existingEl = selector ? document.head.querySelector(selector) : null;

		if (!existingEl) {
			// Only add if it doesn't exist
			const clone = newEl.cloneNode(true) as HTMLElement;
			clone.setAttribute('data-gospa-head', 'true');
			document.head.appendChild(clone);
		}
	});

	// 3. Handle meta tags - update existing or add new
	const newMetaElements = Array.from(newHead.querySelectorAll('meta'));

	newMetaElements.forEach(newEl => {
		const name = newEl.getAttribute('name');
		const property = newEl.getAttribute('property');
		const httpEquiv = newEl.getAttribute('http-equiv');

		// Build selector to find existing meta and for tracking
		let selector = '';
		if (name) selector = `meta[name="${name}"]`;
		else if (property) selector = `meta[property="${property}"]`;
		else if (httpEquiv) selector = `meta[http-equiv="${httpEquiv}"]`;

		if (selector) neededSelectors.add(selector);

		const existingEl = selector ? document.head.querySelector(selector) : null;

		if (existingEl) {
			// Update content attribute only
			const content = newEl.getAttribute('content');
			if (content) existingEl.setAttribute('content', content);
		} else {
			// Add new meta tag
			const clone = newEl.cloneNode(true) as HTMLElement;
			clone.setAttribute('data-gospa-head', 'true');
			document.head.appendChild(clone);
		}
	});

	// 4. Handle style tags - only add new ones, don't remove existing
	const newStyleElements = Array.from(newHead.querySelectorAll('style'));

	newStyleElements.forEach(newEl => {
		const id = newEl.id;
		const selector = id ? `style#${id}` : null;

		if (selector) neededSelectors.add(selector);

		const existingEl = selector ? document.head.querySelector(selector) : null;

		if (!existingEl) {
			const clone = newEl.cloneNode(true) as HTMLElement;
			clone.setAttribute('data-gospa-head', 'true');
			document.head.appendChild(clone);
		}
	});

	// 5. Handle scripts separately if marked
	newHead.querySelectorAll('script[data-gospa-head]').forEach(el => {
		const src = el.getAttribute('src');
		const selector = src ? `script[src="${src}"]` : `script`;

		neededSelectors.add(selector);

		const existingEl = src ? document.head.querySelector(`script[src="${src}"]`) : null;

		if (!existingEl) {
			const script = document.createElement('script');
			Array.from(el.attributes).forEach(attr => script.setAttribute(attr.name, attr.value));
			script.textContent = el.textContent;
			document.head.appendChild(script);
		}
	});

	// 6. Clean up old GoSPA-managed head elements that are no longer needed
	// This prevents memory leaks and DOM bloat during long SPA sessions
	const existingGoSPAElements = document.head.querySelectorAll('[data-gospa-head]');
	existingGoSPAElements.forEach(el => {
		let shouldRemove = true;

		// Check if this element matches any of the needed selectors
		for (const needed of neededSelectors) {
			if (el.matches(needed)) {
				shouldRemove = false;
				break;
			}
		}

		// For link and meta elements, also check by attribute patterns
		if (el.matches('link[href]')) {
			const href = el.getAttribute('href');
			if (href && neededSelectors.has(`link[href="${href}"]`)) {
				shouldRemove = false;
			}
		} else if (el.matches('meta[name]')) {
			const name = el.getAttribute('name');
			if (name && neededSelectors.has(`meta[name="${name}"]`)) {
				shouldRemove = false;
			}
		} else if (el.matches('meta[property]')) {
			const property = el.getAttribute('property');
			if (property && neededSelectors.has(`meta[property="${property}"]`)) {
				shouldRemove = false;
			}
		} else if (el.matches('meta[http-equiv]')) {
			const httpEquiv = el.getAttribute('http-equiv');
			if (httpEquiv && neededSelectors.has(`meta[http-equiv="${httpEquiv}"]`)) {
				shouldRemove = false;
			}
		} else if (el.matches('style[id]')) {
			const id = el.id;
			if (id && neededSelectors.has(`style#${id}`)) {
				shouldRemove = false;
			}
		} else if (el.matches('script[data-gospa-head]')) {
			const src = el.getAttribute('src');
			if (src && neededSelectors.has(`script[src="${src}"]`)) {
				shouldRemove = false;
			}
		}

		if (shouldRemove) {
			el.remove();
		}
	});
}

// Initialize new content (re-run runtime setup)
async function initNewContent(): Promise<void> {
	// Re-setup event handlers and bindings for new DOM content
	const eventElements = document.querySelectorAll('[data-on]');
	const boundElements = document.querySelectorAll('[data-bind]');

	// Get WebSocket from global context
	const gospa = (window as any).__gospa__;
	const ws = gospa?._ws;

	// Setup event handlers
	eventElements.forEach((element) => {
		const attr = element.getAttribute('data-on');
		if (!attr) return;

		const [eventType, action] = attr.split(':');
		if (!eventType || !action) return;

		// Remove old listener if any (using clone technique)
		const newElement = element.cloneNode(true) as Element;
		element.parentNode?.replaceChild(newElement, element);

		newElement.addEventListener(eventType, async () => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: 'action', action }));
				return;
			}

			const websocketModule = await import('./websocket.ts');
			websocketModule.sendAction(action);
		});
	});

	// Setup bindings
	for (const element of boundElements) {
		const attr = element.getAttribute('data-bind');
		if (!attr) continue;

		const [bindingType, stateKey] = attr.split(':');
		if (!bindingType || !stateKey) continue;

		const rune = gospa?.state?.get(stateKey);
		if (!rune) continue;

		// Bind element to rune
		const update = async (value: any) => {
			switch (bindingType) {
				case 'text':
					element.textContent = value;
					break;
				case 'html':
					// HTML content is trusted - Templ auto-escapes on the server
					// For user-generated content, use 'gospa/runtime-secure' with DOMPurify
					element.innerHTML = value;
					break;
				case 'value':
					(element as HTMLInputElement).value = value;
					break;
				case 'checked':
					(element as HTMLInputElement).checked = value;
					break;
				case 'show':
					(element as HTMLElement).style.display = value ? '' : 'none';
					break;
			}
		};

		await update(rune.get());
		rune.subscribe((value: any) => update(value));
	}
}

// Navigate to a new path
export async function navigate(path: string, options: NavigationOptions = {}): Promise<boolean> {
	// Don't navigate if already at this path
	if (path === state.currentPath && !options.replace) {
		return false;
	}

	// Serialize navigations: chain onto the previous promise so concurrent
	// calls don't interleave. Each call awaits the previous one before starting,
	// eliminating the TOCTOU gap in the old nullable-pendingNavigation pattern.
	const previous = state.pendingNavigation ?? Promise.resolve(true);
	const current: Promise<boolean> = previous.then(async () => {
		// Re-check path after waiting — a preceding navigation may have already served it
		if (path === state.currentPath && !options.replace) {
			return false;
		}

		state.isNavigating = true;
		beforeNavCallbacks.forEach(cb => cb(path));

		try {
			const data = await getPageData(path);

			if (!data) {
				window.location.href = path;
				return false;
			}

			if (options.replace) {
				window.history.replaceState({ path }, '', path);
			} else {
				window.history.pushState({ path }, '', path);
			}

			state.currentPath = path;
			await updateDOM(data);

			if (options.scrollToTop !== false) {
				if (data.isDocsPage && document.querySelector('[data-gospa-docs-content]')) {
					// For docs navigation, scroll the main content area, not the whole page
					// This preserves sidebar scroll position
					window.scrollTo(0, 0);
				} else {
					window.scrollTo(0, 0);
				}
			}

			afterNavCallbacks.forEach(cb => cb(path));
			document.dispatchEvent(new CustomEvent('gospa:navigated', { detail: { path } }));

			return true;
		} catch (error) {
			// BUG FIX: Ensure navigation state is cleared on error
			// Otherwise isNavigating flag gets stuck, blocking future navigations
			console.error('[GoSPA] Navigation error:', error);
			state.isNavigating = false;
			state.pendingNavigation = null;
			return false;
		} finally {
			state.isNavigating = false;
			// Only clear the pending reference if it still points to this request
			if (state.pendingNavigation === current) {
				state.pendingNavigation = null;
			}
		}
	});

	// Store the chained promise so the next call can serialize onto it
	state.pendingNavigation = current;
	return current;
}

// Go back in history
export function back(): void {
	window.history.back();
}

// Go forward in history
export function forward(): void {
	window.history.forward();
}

// Go to specific position in history
export function go(delta: number): void {
	window.history.go(delta);
}

// Get current path
export function getCurrentPath(): string {
	return state.currentPath;
}

// Check if currently navigating
export function isNavigating(): boolean {
	return state.isNavigating;
}

// Handle popstate (back/forward button)
function handlePopState(event: PopStateEvent): void {
	const path = window.location.pathname;

	// Notify before navigation
	beforeNavCallbacks.forEach(cb => cb(path));

	// Fetch and update
	getPageData(path).then(data => {
		if (data) {
			state.currentPath = path;
			updateDOM(data).then(() => {
				afterNavCallbacks.forEach(cb => cb(path));
				document.dispatchEvent(new CustomEvent('gospa:navigated', { detail: { path } }));
			});
		} else {
			// Fallback to reload
			window.location.reload();
		}
	});
}

// Handle link clicks
function handleLinkClick(event: MouseEvent): void {
	// Only handle left clicks without modifiers
	if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
		return;
	}

	// Find the closest anchor element
	const target = event.target as Element;
	const link = target.closest('a[href]');

	if (!link) return;

	// Check if it's an internal link
	if (!isInternalLink(link as HTMLAnchorElement)) {
		return;
	}

	// Prevent default navigation
	event.preventDefault();

	// Get the href
	const href = link.getAttribute('href');
	if (!href) return;

	// Navigate
	navigate(href);
}

// Initialize navigation
export function initNavigation(): void {
	// Setup link click handler
	document.addEventListener('click', handleLinkClick);

	// Setup popstate handler
	window.addEventListener('popstate', handlePopState);

	// Check for global configuration
	const config = (window as any).__GOSPA_CONFIG__;
	if (config) {
		if (config.ignoredExtensions) {
			setIgnoredExtensions(config.ignoredExtensions);
		}
		if (config.appendExtensions) {
			appendIgnoredExtensions(config.appendExtensions);
		}
	}

	// Mark as SPA-enabled
	document.documentElement.setAttribute('data-gospa-spa', 'true');
}

// Cleanup navigation
export function destroyNavigation(): void {
	document.removeEventListener('click', handleLinkClick);
	window.removeEventListener('popstate', handlePopState);
	document.documentElement.removeAttribute('data-gospa-spa');
}

// Prefetch a page for faster navigation
export async function prefetch(path: string): Promise<void> {
	if (prefetchCache.has(path)) return;

	const data = await fetchPageFromServer(path);
	if (data) {
		prefetchCache.set(path, data);

		// Clean up cache after 30 seconds
		setTimeout(() => prefetchCache.delete(path), 30000);
	}
}

// Export navigation state as reactive
export function createNavigationState() {
	return {
		get path() {
			return state.currentPath;
		},
		get isNavigating() {
			return state.isNavigating;
		},
		navigate,
		back,
		forward,
		go,
		prefetch,
	};
}

// Auto-initialize when DOM is ready
if (typeof document !== 'undefined') {
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', initNavigation);
	} else {
		initNavigation();
	}
}

// Extend window type
declare global {
	interface Window {
		__gospa__?: {
			state: Map<string, any>;
			_ws?: WebSocket;
		};
	}
}
