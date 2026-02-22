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

// Check if a link is internal (same origin)
function isInternalLink(link: HTMLAnchorElement): boolean {
	const href = link.getAttribute('href');
	if (!href || href.startsWith('#') || href.startsWith('javascript:')) {
		return false;
	}
	
	// Check for external links
	if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('//')) {
		try {
			const url = new URL(href, window.location.origin);
			return url.origin === window.location.origin;
		} catch {
			return false;
		}
	}
	
	// Check for special attributes that disable SPA nav
	if (link.hasAttribute('data-external') || 
		link.hasAttribute('download') || 
		link.getAttribute('target') === '_blank') {
		return false;
	}
	
	return true;
}

// Page data type
interface PageData {
	content: string;
	title: string;
	head: string;
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
		
		// Extract content from main element or body
		const mainEl = doc.querySelector('main');
		const content = mainEl ? mainEl.innerHTML : doc.body.innerHTML;
		
		// Extract title
		const title = doc.querySelector('title')?.textContent || '';
		
		// Extract head elements (for head management)
		const headEl = doc.querySelector('head');
		const head = headEl ? headEl.innerHTML : '';
		
		return { content, title, head };
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

// Update the DOM with new content
function updateDOM(data: PageData): void {
	// Update title
	if (data.title) {
		document.title = data.title;
	}
	
	// Update main content
	const mainEl = document.querySelector('main');
	if (mainEl) {
		mainEl.innerHTML = data.content;
	} else {
		// Fallback: replace body content
		document.body.innerHTML = data.content;
	}
	
	// Update head (managed head elements)
	updateHead(data.head);
	
	// Re-initialize runtime for new content
	initNewContent();
}

// Update head elements
function updateHead(headHtml: string): void {
	// Parse head HTML to extract elements
	const parser = new DOMParser();
	const doc = parser.parseFromString(`<html><head>${headHtml}</head></html>`, 'text/html');
	const newHead = doc.querySelector('head');
	
	if (!newHead) return;
	
	// Get elements with data-gospa-head attribute (managed by GoSPA)
	const managedElements = document.head.querySelectorAll('[data-gospa-head]');
	
	// Remove old managed elements
	managedElements.forEach(el => el.remove());
	
	// Add new managed elements
	newHead.querySelectorAll('title, meta, link, style, script').forEach(el => {
		// Skip certain elements
		if (el.tagName === 'SCRIPT' && !el.hasAttribute('data-gospa-head')) {
			return; // Don't auto-inject scripts unless marked
		}
		
		// Mark as managed
		el.setAttribute('data-gospa-head', 'true');
		
		// Clone and append
		document.head.appendChild(el.cloneNode(true));
	});
}

// Initialize new content (re-run runtime setup)
function initNewContent(): void {
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
		
		newElement.addEventListener(eventType, () => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ action }));
			}
		});
	});
	
	// Setup bindings
	boundElements.forEach((element) => {
		const attr = element.getAttribute('data-bind');
		if (!attr) return;
		
		const [bindingType, stateKey] = attr.split(':');
		if (!bindingType || !stateKey) return;
		
		const rune = gospa?.state?.get(stateKey);
		if (!rune) return;
		
		// Bind element to rune
		const update = (value: any) => {
			switch (bindingType) {
				case 'text':
					element.textContent = value;
					break;
				case 'html':
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
		
		update(rune.get());
		rune.subscribe(update);
	});
}

// Navigate to a new path
export async function navigate(path: string, options: NavigationOptions = {}): Promise<boolean> {
	// Don't navigate if already at this path
	if (path === state.currentPath && !options.replace) {
		return false;
	}
	
	// Wait for any pending navigation
	if (state.pendingNavigation) {
		await state.pendingNavigation;
	}
	
	state.isNavigating = true;
	
	// Notify before navigation
	beforeNavCallbacks.forEach(cb => cb(path));
	
	try {
		state.pendingNavigation = (async () => {
			const data = await getPageData(path);
			
			if (!data) {
				// Fallback to full page load
				window.location.href = path;
				return false;
			}
			
			// Update browser history
			if (options.replace) {
				window.history.replaceState({ path }, '', path);
			} else {
				window.history.pushState({ path }, '', path);
			}
			
			// Update state
			state.currentPath = path;
			
			// Update DOM
			updateDOM(data);
			
			// Scroll to top if requested
			if (options.scrollToTop !== false) {
				window.scrollTo(0, 0);
			}
			
			// Notify after navigation
			afterNavCallbacks.forEach(cb => cb(path));
			
			return true;
		})();
		
		return await state.pendingNavigation;
	} finally {
		state.isNavigating = false;
		state.pendingNavigation = null;
	}
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
			updateDOM(data);
			afterNavCallbacks.forEach(cb => cb(path));
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
	
	// Mark as SPA-enabled
	document.documentElement.setAttribute('data-gospa-spa', 'true');
	
	console.log('[GoSPA] SPA navigation initialized');
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
