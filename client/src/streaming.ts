/**
 * GoSPA Streaming SSR Runtime
 * Handles progressive hydration and streaming content updates
 */

// Stream chunk types
interface StreamChunk {
	type: 'html' | 'island' | 'script' | 'state' | 'error';
	id: string;
	content: string;
	data: Record<string, unknown>;
}

// Island data from server
interface IslandData {
	id: string;
	name: string;
	mode: 'immediate' | 'visible' | 'idle' | 'interaction' | 'lazy';
	priority: 'high' | 'normal' | 'low';
	props: Record<string, unknown>;
	state: Record<string, unknown>;
}

// Hydration queue item
interface HydrationQueueItem {
	island: IslandData;
	resolve: () => void;
	reject: (error: Error) => void;
}

// Streaming manager options
interface StreamingManagerOptions {
	enableLogging?: boolean;
	hydrationTimeout?: number;
}

/**
 * StreamingManager handles streaming SSR content and progressive hydration
 */
export class StreamingManager {
	private islands: IslandData[] = [];
	private hydrationQueue: HydrationQueueItem[] = [];
	private hydratedIslands = new Set<string>();
	private isHydrating = false;
	private options: StreamingManagerOptions;

	constructor(options: StreamingManagerOptions = {}) {
		this.options = {
			enableLogging: false,
			hydrationTimeout: 30000,
			...options,
		};

		// Set up global stream handler
		this.setupStreamHandler();
	}

	/**
	 * Set up the global stream handler for incoming chunks
	 */
	private setupStreamHandler(): void {
		// Extend the global stream function
		const existingHandler = (globalThis as unknown as Record<string, unknown>).__GOSPA_STREAM__;
		
		(globalThis as unknown as Record<string, unknown>).__GOSPA_STREAM__ = (chunk: StreamChunk) => {
			// Call existing handler first
			if (typeof existingHandler === 'function') {
				(existingHandler as (chunk: StreamChunk) => void)(chunk);
			}
			
			// Process chunk
			this.processChunk(chunk);
		};
	}

	/**
	 * Process an incoming stream chunk
	 */
	private processChunk(chunk: StreamChunk): void {
		if (this.options.enableLogging) {
			console.log('[GoSPA Stream]', chunk.type, chunk.id || '', chunk);
		}

		switch (chunk.type) {
			case 'html':
				this.handleHtmlChunk(chunk);
				break;
			case 'island':
				this.handleIslandChunk(chunk);
				break;
			case 'script':
				this.handleScriptChunk(chunk);
				break;
			case 'state':
				this.handleStateChunk(chunk);
				break;
			case 'error':
				this.handleErrorChunk(chunk);
				break;
		}
	}

	/**
	 * Handle HTML chunk - replace content in DOM
	 */
	private handleHtmlChunk(chunk: StreamChunk): void {
		const element = document.getElementById(chunk.id);
		if (element) {
			element.innerHTML = chunk.content;
			
			// Dispatch custom event for HTML update
			element.dispatchEvent(new CustomEvent('gospa:html-update', {
				detail: { id: chunk.id, content: chunk.content },
			}));
		}
	}

	/**
	 * Handle island chunk - queue for hydration
	 */
	private handleIslandChunk(chunk: StreamChunk): void {
		const islandData = chunk.data as unknown as IslandData;
		if (!islandData || !islandData.id) {
			console.error('[GoSPA Stream] Invalid island data:', chunk);
			return;
		}

		this.islands.push(islandData);
		
		// Queue for hydration based on mode
		this.queueHydration(islandData);
	}

	/**
	 * Handle script chunk - execute script
	 */
	private handleScriptChunk(chunk: StreamChunk): void {
		const script = document.createElement('script');
		script.textContent = chunk.content;
		document.head.appendChild(script);
	}

	/**
	 * Handle state chunk - update state
	 */
	private handleStateChunk(chunk: StreamChunk): void {
		// Store state globally
		const gospaState = (globalThis as unknown as Record<string, Record<string, unknown>>).__GOSPA_STATE__ ||= {};
		gospaState[chunk.id] = chunk.data;

		// Dispatch state update event
		document.dispatchEvent(new CustomEvent('gospa:state-update', {
			detail: { id: chunk.id, state: chunk.data },
		}));
	}

	/**
	 * Handle error chunk
	 */
	private handleErrorChunk(chunk: StreamChunk): void {
		console.error('[GoSPA Stream Error]', chunk.content);
		
		// Dispatch error event
		document.dispatchEvent(new CustomEvent('gospa:stream-error', {
			detail: { error: chunk.content },
		}));
	}

	/**
	 * Queue an island for hydration based on its mode
	 */
	private queueHydration(island: IslandData): void {
		switch (island.mode) {
			case 'immediate':
				this.hydrateImmediate(island);
				break;
			case 'visible':
				this.hydrateOnVisible(island);
				break;
			case 'idle':
				this.hydrateOnIdle(island);
				break;
			case 'interaction':
				this.hydrateOnInteraction(island);
				break;
			case 'lazy':
				this.hydrateLazy(island);
				break;
			default:
				this.hydrateImmediate(island);
		}
	}

	/**
	 * Hydrate immediately
	 */
	private hydrateImmediate(island: IslandData): void {
		this.addToHydrationQueue(island, 'high');
	}

	/**
	 * Hydrate when element is visible
	 */
	private hydrateOnVisible(island: IslandData): void {
		const element = document.querySelector(`[data-gospa-island="${island.id}"]`);
		if (!element) {
			// Element not found, hydrate immediately as fallback
			this.hydrateImmediate(island);
			return;
		}

		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (entry.isIntersecting) {
						observer.disconnect();
						this.addToHydrationQueue(island, 'normal');
					}
				}
			},
			{ rootMargin: '100px' }
		);

		observer.observe(element);
	}

	/**
	 * Hydrate when browser is idle
	 */
	private hydrateOnIdle(island: IslandData): void {
		if ('requestIdleCallback' in globalThis) {
			(globalThis as unknown as { requestIdleCallback: (cb: () => void) => void }).requestIdleCallback(() => {
				this.addToHydrationQueue(island, 'low');
			});
		} else {
			// Fallback for browsers without requestIdleCallback
			setTimeout(() => {
				this.addToHydrationQueue(island, 'low');
			}, 100);
		}
	}

	/**
	 * Hydrate on user interaction
	 */
	private hydrateOnInteraction(island: IslandData): void {
		const element = document.querySelector(`[data-gospa-island="${island.id}"]`);
		if (!element) {
			this.hydrateImmediate(island);
			return;
		}

		const events = ['mouseenter', 'touchstart', 'focusin', 'click'];
		const handler = () => {
			events.forEach((event) => element.removeEventListener(event, handler));
			this.addToHydrationQueue(island, 'high');
		};

		events.forEach((event) => {
			element.addEventListener(event, handler, { once: true, passive: true });
		});
	}

	/**
	 * Hydrate lazily (lowest priority)
	 */
	private hydrateLazy(island: IslandData): void {
		// Wait for page load and then some idle time
		if (document.readyState === 'complete') {
			this.hydrateOnIdle(island);
		} else {
			globalThis.addEventListener('load', () => {
				setTimeout(() => {
					this.hydrateOnIdle(island);
				}, 500);
			});
		}
	}

	/**
	 * Add island to hydration queue with priority
	 */
	private addToHydrationQueue(island: IslandData, priority: 'high' | 'normal' | 'low'): void {
		if (this.hydratedIslands.has(island.id)) {
			return; // Already hydrated
		}

		const queueItem: HydrationQueueItem = {
			island,
			resolve: () => {},
			reject: () => {},
		};

		// Insert based on priority
		if (priority === 'high') {
			this.hydrationQueue.unshift(queueItem);
		} else {
			this.hydrationQueue.push(queueItem);
		}

		this.processQueue();
	}

	/**
	 * Process the hydration queue
	 */
	private processQueue(): void {
		if (this.isHydrating || this.hydrationQueue.length === 0) {
			return;
		}

		this.isHydrating = true;
		const item = this.hydrationQueue.shift();

		if (item) {
			this.hydrateIsland(item.island)
				.then(() => {
					this.hydratedIslands.add(item.island.id);
					this.isHydrating = false;
					this.processQueue();
				})
				.catch((error) => {
					console.error('[GoSPA] Hydration error:', error);
					this.isHydrating = false;
					this.processQueue();
				});
		}
	}

	/**
	 * Hydrate a single island
	 */
	private async hydrateIsland(island: IslandData): Promise<void> {
		const element = document.querySelector(`[data-gospa-island="${island.id}"]`);
		if (!element) {
			if (this.options.enableLogging) {
				console.warn('[GoSPA] Island element not found:', island.id);
			}
			return;
		}

		// Check if island manager is available
		const islandManager = (globalThis as unknown as Record<string, unknown>).__GOSPA_ISLAND_MANAGER__;
		if (islandManager && typeof (islandManager as { hydrate: (id: string, data: IslandData) => Promise<void> }).hydrate === 'function') {
			await (islandManager as { hydrate: (id: string, data: IslandData) => Promise<void> }).hydrate(island.id, island);
		}

		// Dispatch hydration event
		element.dispatchEvent(new CustomEvent('gospa:hydrated', {
			detail: { island },
		}));

		if (this.options.enableLogging) {
			console.log('[GoSPA] Hydrated island:', island.id, island.name);
		}
	}

	/**
	 * Get all registered islands
	 */
	getIslands(): IslandData[] {
		return [...this.islands];
	}

	/**
	 * Get hydrated island IDs
	 */
	getHydratedIslands(): Set<string> {
		return new Set(this.hydratedIslands);
	}

	/**
	 * Check if an island is hydrated
	 */
	isHydrated(islandId: string): boolean {
		return this.hydratedIslands.has(islandId);
	}

	/**
	 * Manually trigger hydration for an island
	 */
	async hydrate(islandId: string): Promise<void> {
		const island = this.islands.find((i) => i.id === islandId);
		if (island) {
			await this.hydrateIsland(island);
		}
	}
}

// Global type extensions
declare global {
	interface Window {
		__GOSPA_STREAM__?: (chunk: StreamChunk) => void;
		__GOSPA_STATE__?: Record<string, Record<string, unknown>>;
		__GOSPA_ISLAND_MANAGER__?: unknown;
	}
}

// Create singleton instance
let streamingManager: StreamingManager | null = null;

/**
 * Initialize the streaming manager
 */
export function initStreaming(options?: StreamingManagerOptions): StreamingManager {
	if (!streamingManager) {
		streamingManager = new StreamingManager(options);
	}
	return streamingManager;
}

/**
 * Get the streaming manager instance
 */
export function getStreamingManager(): StreamingManager | null {
	return streamingManager;
}

// Auto-initialize if in browser
if (typeof window !== 'undefined') {
	// Defer initialization to allow for configuration
	setTimeout(() => {
		if (!streamingManager) {
			initStreaming();
		}
	}, 0);
}
