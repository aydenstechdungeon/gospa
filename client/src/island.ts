/**
 * Client-side island runtime for GoSPA.
 * Handles island detection, hydration orchestration, and priority-based loading.
 */

// Island hydration modes
export type IslandHydrationMode = 'immediate' | 'visible' | 'idle' | 'interaction' | 'lazy';

// Island priority levels
export type IslandPriority = 'high' | 'normal' | 'low';

// Island data from DOM
export interface IslandElementData {
	id: string;
	name: string;
	mode: IslandHydrationMode;
	priority: IslandPriority;
	props?: Record<string, unknown>;
	state?: Record<string, unknown>;
	threshold?: number;
	defer?: number;
	clientOnly?: boolean;
	serverOnly?: boolean;
	element: Element;
}

// Island hydration result
export interface IslandHydrationResult {
	id: string;
	name: string;
	success: boolean;
	error?: Error;
}

// Island module loader type
export type IslandModuleLoader = (name: string) => Promise<IslandModule | null>;

// Island module interface
export interface IslandModule {
	default?: {
		hydrate?: (element: Element, props: Record<string, unknown>, state: Record<string, unknown>) => void | Promise<void>;
		mount?: (element: Element, props: Record<string, unknown>, state: Record<string, unknown>) => void | Promise<void>;
	};
	hydrate?: (element: Element, props: Record<string, unknown>, state: Record<string, unknown>) => void | Promise<void>;
	mount?: (element: Element, props: Record<string, unknown>, state: Record<string, unknown>) => void | Promise<void>;
}

// Island manager configuration
export interface IslandManagerConfig {
	// Custom module loader
	moduleLoader?: IslandModuleLoader;
	// Base path for island modules
	moduleBasePath?: string;
	// Default hydration timeout
	defaultTimeout?: number;
	// Enable debug logging
	debug?: boolean;
}

// Hydration queue item
interface HydrationQueueItem {
	island: IslandElementData;
	priority: IslandPriority;
	resolve: (result: IslandHydrationResult) => void;
	reject: (error: Error) => void;
}

/**
 * IslandManager handles island detection and hydration orchestration.
 */
export class IslandManager {
	private islands: Map<string, IslandElementData> = new Map();
	private hydrated: Set<string> = new Set();
	private pending: Map<string, Promise<IslandHydrationResult>> = new Map();
	private queue: HydrationQueueItem[] = [];
	private processing = false;
	private moduleLoader: IslandModuleLoader;
	private moduleBasePath: string;
	private defaultTimeout: number;
	private debug: boolean;
	private observers: IntersectionObserver[] = [];
	private idleCallbacks: Map<string, number | ReturnType<typeof setTimeout>> = new Map();
	private interactionListeners: Map<string, () => void> = new Map();

	constructor(config: IslandManagerConfig = {}) {
		this.moduleLoader = config.moduleLoader ?? this.defaultModuleLoader;
		this.moduleBasePath = config.moduleBasePath ?? '/islands';
		this.defaultTimeout = config.defaultTimeout ?? 30000;
		this.debug = config.debug ?? false;

		// Auto-discover islands on DOMContentLoaded
		if (document.readyState === 'loading') {
			document.addEventListener('DOMContentLoaded', () => this.discoverIslands());
		} else {
			this.discoverIslands();
		}
	}

	/**
	 * Discover all islands in the DOM.
	 */
	discoverIslands(): IslandElementData[] {
		const elements = document.querySelectorAll('[data-gospa-island]');
		const discovered: IslandElementData[] = [];

		elements.forEach((element) => {
			const data = this.parseIslandElement(element);
			if (data && !this.islands.has(data.id)) {
				this.islands.set(data.id, data);
				discovered.push(data);
				this.log('Discovered island:', data.name, data.id);
			}
		});

		// Start hydration based on modes
		this.scheduleHydration(discovered);

		return discovered;
	}

	/**
	 * Parse island data from DOM element.
	 */
	private parseIslandElement(element: Element): IslandElementData | null {
		const id = element.id || this.generateId();
		const name = element.getAttribute('data-gospa-island');
		if (!name) return null;

		const mode = (element.getAttribute('data-gospa-mode') as IslandHydrationMode) || 'immediate';
		const priority = (element.getAttribute('data-gospa-priority') as IslandPriority) || 'normal';

		let props: Record<string, unknown> | undefined;
		let state: Record<string, unknown> | undefined;

		const propsAttr = element.getAttribute('data-gospa-props');
		if (propsAttr) {
			try {
				props = JSON.parse(propsAttr);
			} catch (e) {
				this.log('Failed to parse props for island:', name, e);
			}
		}

		const stateAttr = element.getAttribute('data-gospa-state');
		if (stateAttr) {
			try {
				state = JSON.parse(stateAttr);
			} catch (e) {
				this.log('Failed to parse state for island:', name, e);
			}
		}

		const thresholdAttr = element.getAttribute('data-gospa-threshold');
		const deferAttr = element.getAttribute('data-gospa-defer');

		return {
			id,
			name,
			mode,
			priority,
			props,
			state,
			threshold: thresholdAttr ? parseInt(thresholdAttr, 10) : undefined,
			defer: deferAttr ? parseInt(deferAttr, 10) : undefined,
			clientOnly: element.getAttribute('data-gospa-client-only') === 'true',
			serverOnly: element.getAttribute('data-gospa-server-only') === 'true',
			element,
		};
	}

	/**
	 * Schedule hydration based on island modes.
	 */
	private scheduleHydration(islands: IslandElementData[]): void {
		for (const island of islands) {
			if (this.hydrated.has(island.id) || this.pending.has(island.id)) {
				continue;
			}

			switch (island.mode) {
				case 'immediate':
					this.queueHydration(island);
					break;

				case 'visible':
					this.scheduleVisibleHydration(island);
					break;

				case 'idle':
					this.scheduleIdleHydration(island);
					break;

				case 'interaction':
					this.scheduleInteractionHydration(island);
					break;

				case 'lazy':
					// Lazy islands are hydrated on demand
					break;
			}
		}

		this.processQueue();
	}

	/**
	 * Queue island for hydration.
	 */
	private queueHydration(island: IslandElementData): Promise<IslandHydrationResult> {
		if (this.pending.has(island.id)) {
			return this.pending.get(island.id)!;
		}

		const promise = new Promise<IslandHydrationResult>((resolve, reject) => {
			this.queue.push({
				island,
				priority: island.priority,
				resolve,
				reject,
			});
		});

		this.pending.set(island.id, promise);
		return promise;
	}

	/**
	 * Process hydration queue in priority order.
	 */
	private async processQueue(): Promise<void> {
		if (this.processing) return;
		this.processing = true;

		while (this.queue.length > 0) {
			// Sort by priority
			this.queue.sort((a, b) => {
				const priorityOrder = { high: 0, normal: 1, low: 2 };
				return priorityOrder[a.priority] - priorityOrder[b.priority];
			});

			const item = this.queue.shift()!;
			try {
				const result = await this.hydrateIsland(item.island);
				item.resolve(result);
			} catch (error) {
				item.reject(error as Error);
			}
		}

		this.processing = false;
	}

	/**
	 * Hydrate a single island.
	 */
	async hydrateIsland(island: IslandElementData): Promise<IslandHydrationResult> {
		if (this.hydrated.has(island.id)) {
			return { id: island.id, name: island.name, success: true };
		}

		if (island.serverOnly) {
			this.log('Skipping server-only island:', island.name);
			return { id: island.id, name: island.name, success: true };
		}

		this.log('Hydrating island:', island.name, island.id);

		try {
			const module = await this.moduleLoader(island.name);
			if (!module) {
				throw new Error(`Island module not found: ${island.name}`);
			}

			// Get hydrate or mount function
			const hydrateFn = module.hydrate ?? module.default?.hydrate ?? module.mount ?? module.default?.mount;
			if (!hydrateFn) {
				throw new Error(`No hydrate or mount function found for island: ${island.name}`);
			}

			// Execute hydration
			await hydrateFn(island.element, island.props ?? {}, island.state ?? {});

			this.hydrated.add(island.id);
			this.log('Hydrated island:', island.name);

			return { id: island.id, name: island.name, success: true };
		} catch (error) {
			this.log('Failed to hydrate island:', island.name, error);
			return { id: island.id, name: island.name, success: false, error: error as Error };
		}
	}

	/**
	 * Schedule hydration when island becomes visible.
	 */
	private scheduleVisibleHydration(island: IslandElementData): void {
		if (!('IntersectionObserver' in window)) {
			// Fallback to immediate hydration
			this.queueHydration(island);
			this.processQueue();
			return;
		}

		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (entry.isIntersecting) {
						this.queueHydration(island);
						this.processQueue();
						observer.disconnect();
						this.observers = this.observers.filter((o) => o !== observer);
					}
				}
			},
			{
				rootMargin: `${island.threshold ?? 200}px`,
			}
		);

		observer.observe(island.element);
		this.observers.push(observer);
	}

	/**
	 * Schedule hydration during idle time.
	 */
	private scheduleIdleHydration(island: IslandElementData): void {
		if (typeof requestIdleCallback !== 'undefined') {
			const callbackId = requestIdleCallback(
				() => {
					this.queueHydration(island);
					this.processQueue();
					this.idleCallbacks.delete(island.id);
				},
				{ timeout: island.defer ?? 2000 }
			);
			this.idleCallbacks.set(island.id, callbackId);
		} else {
			// Fallback to setTimeout
			const timeoutId = setTimeout(() => {
				this.queueHydration(island);
				this.processQueue();
				this.idleCallbacks.delete(island.id);
			}, island.defer ?? 2000);
			this.idleCallbacks.set(island.id, timeoutId);
		}
	}

	/**
	 * Schedule hydration on first interaction.
	 */
	private scheduleInteractionHydration(island: IslandElementData): void {
		const events = ['mouseenter', 'touchstart', 'focusin', 'click'];

		const hydrateOnInteraction = () => {
			this.queueHydration(island);
			this.processQueue();

			// Remove all listeners
			for (const event of events) {
				island.element.removeEventListener(event, hydrateOnInteraction);
			}
			this.interactionListeners.delete(island.id);
		};

		for (const event of events) {
			island.element.addEventListener(event, hydrateOnInteraction, { passive: true, once: true });
		}

		this.interactionListeners.set(island.id, hydrateOnInteraction);
	}

	/**
	 * Default module loader - loads from module base path.
	 */
	private defaultModuleLoader: IslandModuleLoader = async (name: string) => {
		try {
			const module = await import(`${this.moduleBasePath}/${name}.js`);
			return module as IslandModule;
		} catch (error) {
			this.log('Failed to load island module:', name, error);
			return null;
		}
	};

	/**
	 * Generate unique ID.
	 */
	private generateId(): string {
		return `gospa-island-${Math.random().toString(36).substring(2, 11)}`;
	}

	/**
	 * Debug logging.
	 */
	private log(...args: unknown[]): void {
		if (this.debug) {
			console.log('[GoSPA Islands]', ...args);
		}
	}

	/**
	 * Get all discovered islands.
	 */
	getIslands(): IslandElementData[] {
		return Array.from(this.islands.values());
	}

	/**
	 * Get island by ID.
	 */
	getIsland(id: string): IslandElementData | undefined {
		return this.islands.get(id);
	}

	/**
	 * Check if island is hydrated.
	 */
	isHydrated(id: string): boolean {
		return this.hydrated.has(id);
	}

	/**
	 * Manually hydrate an island by ID or name.
	 */
	async hydrate(idOrName: string): Promise<IslandHydrationResult | null> {
		// Find by ID first, then by name
		let island = this.islands.get(idOrName);
		if (!island) {
			island = Array.from(this.islands.values()).find((i) => i.name === idOrName);
		}
		if (!island) {
			return null;
		}

		return this.hydrateIsland(island);
	}

	/**
	 * Cleanup observers and listeners.
	 */
	destroy(): void {
		// Disconnect intersection observers
		for (const observer of this.observers) {
			observer.disconnect();
		}
		this.observers = [];

		// Cancel idle callbacks
		for (const [id, callbackId] of this.idleCallbacks) {
			if ('cancelIdleCallback' in window) {
				(window as any).cancelIdleCallback(callbackId);
			} else {
				clearTimeout(callbackId);
			}
		}
		this.idleCallbacks.clear();

		// Remove interaction listeners
		for (const [id, listener] of this.interactionListeners) {
			const island = this.islands.get(id);
			if (island) {
				const events = ['mouseenter', 'touchstart', 'focusin', 'click'];
				for (const event of events) {
					island.element.removeEventListener(event, listener);
				}
			}
		}
		this.interactionListeners.clear();
	}
}

// Global island manager instance
let globalManager: IslandManager | null = null;

/**
 * Initialize the global island manager.
 */
export function initIslands(config?: IslandManagerConfig): IslandManager {
	if (globalManager) {
		return globalManager;
	}
	globalManager = new IslandManager(config);
	return globalManager;
}

/**
 * Get the global island manager.
 */
export function getIslandManager(): IslandManager | null {
	return globalManager;
}

/**
 * Hydrate a specific island.
 */
export async function hydrateIsland(idOrName: string): Promise<IslandHydrationResult | null> {
	if (!globalManager) {
		console.warn('Island manager not initialized. Call initIslands() first.');
		return null;
	}
	return globalManager.hydrate(idOrName);
}

// Auto-initialize if data-gospa-islands script tag is present
if (typeof document !== 'undefined') {
	const scriptTag = document.querySelector('[data-gospa-islands]');
	if (scriptTag) {
		initIslands();
	}
}

// Export for window global
if (typeof window !== 'undefined') {
	(window as any).__GOSPA_ISLAND_MANAGER__ = {
		init: initIslands,
		get: getIslandManager,
		hydrate: hydrateIsland,
		IslandManager,
	};
}
