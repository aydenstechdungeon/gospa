/**
 * Priority-based selective hydration for GoSPA islands.
 * Manages intelligent loading order based on component importance.
 */

import type { IslandHydrationMode } from './island';

/**
 * Numeric priority levels for hydration ordering.
 */
export type PriorityLevel = number;

export const PRIORITY_CRITICAL: PriorityLevel = 100;
export const PRIORITY_HIGH: PriorityLevel = 75;
export const PRIORITY_NORMAL: PriorityLevel = 50;
export const PRIORITY_LOW: PriorityLevel = 25;
export const PRIORITY_DEFERRED: PriorityLevel = 10;

/**
 * Priority island configuration.
 */
export interface PriorityIsland {
	id: string;
	name: string;
	priority: PriorityLevel;
	mode: IslandHydrationMode;
	dependencies?: string[];
	state?: unknown;
	script?: string;
	position: number;
	metadata?: Record<string, unknown>;
}

/**
 * Hydration plan from server.
 */
export interface HydrationPlan {
	immediate: PriorityIsland[];
	idle: PriorityIsland[];
	visible: PriorityIsland[];
	interaction: PriorityIsland[];
	lazy: PriorityIsland[];
	preload: string[];
}

/**
 * Priority scheduler configuration.
 */
export interface PriorityConfig {
	maxConcurrent: number;
	idleTimeout: number;
	intersectionThreshold: number;
	intersectionRootMargin: string;
	enablePreload: boolean;
}

const DEFAULT_CONFIG: PriorityConfig = {
	maxConcurrent: 3,
	idleTimeout: 2000,
	intersectionThreshold: 0.1,
	intersectionRootMargin: '50px',
	enablePreload: true,
};

/**
 * Island hydration state.
 */
type HydrationState = 'pending' | 'hydrating' | 'hydrated' | 'error';

/**
 * Tracked island with state.
 */
interface TrackedIsland extends PriorityIsland {
	state: HydrationState;
	error?: Error;
	element?: Element;
}

/**
 * PriorityScheduler manages priority-based island hydration.
 */
export class PriorityScheduler {
	private config: PriorityConfig;
	private islands: Map<string, TrackedIsland> = new Map();
	private hydrationQueue: TrackedIsland[] = [];
	private activeHydrations = 0;
	private observers: Map<string, IntersectionObserver> = new Map();
	private idleCallbacks: Map<string, number> = new Map();
	private interactionHandlers: Map<string, EventListener[]> = new Map();

	constructor(config: Partial<PriorityConfig> = {}) {
		this.config = { ...DEFAULT_CONFIG, ...config };
	}

	/**
	 * Register islands from a hydration plan.
	 */
	registerPlan(plan: HydrationPlan): void {
		// Preload high-priority scripts
		if (this.config.enablePreload) {
			this.preloadScripts(plan.preload);
		}

		// Register all islands
		for (const island of plan.immediate) {
			this.registerIsland(island, 'immediate');
		}
		for (const island of plan.idle) {
			this.registerIsland(island, 'idle');
		}
		for (const island of plan.visible) {
			this.registerIsland(island, 'visible');
		}
		for (const island of plan.interaction) {
			this.registerIsland(island, 'interaction');
		}
		for (const island of plan.lazy) {
			this.registerIsland(island, 'lazy');
		}

		// Start processing
		this.processQueue();
	}

	/**
	 * Register a single island.
	 */
	registerIsland(island: PriorityIsland, mode: IslandHydrationMode): void {
		const tracked: TrackedIsland = {
			...island,
			state: 'pending',
			mode,
		};

		this.islands.set(island.id, tracked);

		// Find DOM element
		const element = document.querySelector(`[data-island-id="${island.id}"]`);
		if (element) {
			tracked.element = element;
		}

		// Setup hydration triggers based on mode
		this.setupHydrationTrigger(tracked);
	}

	/**
	 * Setup hydration trigger based on island mode.
	 */
	private setupHydrationTrigger(island: TrackedIsland): void {
		switch (island.mode) {
			case 'immediate':
				// Add to immediate queue
				this.hydrationQueue.push(island);
				break;

			case 'idle':
				this.setupIdleHydration(island);
				break;

			case 'visible':
				this.setupVisibleHydration(island);
				break;

			case 'interaction':
				this.setupInteractionHydration(island);
				break;

			case 'lazy':
				// Lazy islands are hydrated last, on explicit trigger
				break;
		}
	}

	/**
	 * Setup idle callback hydration.
	 */
	private setupIdleHydration(island: TrackedIsland): void {
		if ('requestIdleCallback' in window) {
			const callbackId = requestIdleCallback(
				() => {
					this.hydrationQueue.push(island);
					this.processQueue();
				},
				{ timeout: this.config.idleTimeout }
			);
			this.idleCallbacks.set(island.id, callbackId);
		} else {
			// Fallback: use setTimeout
			setTimeout(() => {
				this.hydrationQueue.push(island);
				this.processQueue();
			}, this.config.idleTimeout);
		}
	}

	/**
	 * Setup intersection observer for visible hydration.
	 */
	private setupVisibleHydration(island: TrackedIsland): void {
		if (!island.element) {
			// Element not found, hydrate immediately
			this.hydrationQueue.push(island);
			this.processQueue();
			return;
		}

		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (entry.isIntersecting) {
						this.hydrationQueue.push(island);
						this.processQueue();
						observer.disconnect();
						this.observers.delete(island.id);
					}
				}
			},
			{
				threshold: this.config.intersectionThreshold,
				rootMargin: this.config.intersectionRootMargin,
			}
		);

		observer.observe(island.element);
		this.observers.set(island.id, observer);
	}

	/**
	 * Setup interaction-based hydration.
	 */
	private setupInteractionHydration(island: TrackedIsland): void {
		if (!island.element) {
			// Element not found, add to queue
			this.hydrationQueue.push(island);
			this.processQueue();
			return;
		}

		const events = ['click', 'focus', 'mouseenter', 'touchstart'];
		const handlers: EventListener[] = [];

		const hydrateOnInteraction = (event: Event) => {
			// Remove all listeners
			for (let i = 0; i < events.length; i++) {
				island.element!.removeEventListener(events[i], handlers[i]);
			}

			// Hydrate
			this.hydrationQueue.push(island);
			this.processQueue();
		};

		for (const eventType of events) {
			const handler = hydrateOnInteraction;
			handlers.push(handler);
			island.element.addEventListener(eventType, handler, { passive: true, once: true });
		}

		this.interactionHandlers.set(island.id, handlers);
	}

	/**
	 * Process the hydration queue.
	 */
	private processQueue(): void {
		// Sort by priority (highest first)
		this.hydrationQueue.sort((a, b) => {
			if (a.priority !== b.priority) {
				return b.priority - a.priority;
			}
			return a.position - b.position;
		});

		// Hydrate up to maxConcurrent
		while (
			this.activeHydrations < this.config.maxConcurrent &&
			this.hydrationQueue.length > 0
		) {
			const island = this.hydrationQueue.shift();
			if (island && island.state === 'pending') {
				this.hydrateIsland(island);
			}
		}
	}

	/**
	 * Hydrate a single island.
	 */
	private async hydrateIsland(island: TrackedIsland): Promise<void> {
		island.state = 'hydrating';
		this.activeHydrations++;

		try {
			// Wait for dependencies first
			await this.waitForDependencies(island);

			// Dispatch hydration event
			const event = new CustomEvent('gospa:hydrate', {
				detail: {
					id: island.id,
					name: island.name,
					state: island.state,
				},
			});
			document.dispatchEvent(event);

			// Mark as hydrated
			island.state = 'hydrated';

			// Dispatch hydrated event
			const hydratedEvent = new CustomEvent('gospa:hydrated', {
				detail: { id: island.id, name: island.name },
			});
			document.dispatchEvent(hydratedEvent);
		} catch (error) {
			island.state = 'error';
			island.error = error instanceof Error ? error : new Error(String(error));

			// Dispatch error event
			const errorEvent = new CustomEvent('gospa:hydration-error', {
				detail: { id: island.id, error: island.error },
			});
			document.dispatchEvent(errorEvent);
		} finally {
			this.activeHydrations--;
			this.processQueue();
		}
	}

	/**
	 * Wait for island dependencies to be hydrated.
	 */
	private async waitForDependencies(island: TrackedIsland): Promise<void> {
		if (!island.dependencies || island.dependencies.length === 0) {
			return;
		}

		const promises = island.dependencies.map((depId) => {
			return new Promise<void>((resolve) => {
				const dep = this.islands.get(depId);
				if (!dep || dep.state === 'hydrated') {
					resolve();
					return;
				}

				// Listen for hydration
				const handler = (event: Event) => {
					const customEvent = event as CustomEvent;
					if (customEvent.detail.id === depId) {
						document.removeEventListener('gospa:hydrated', handler);
						resolve();
					}
				};
				document.addEventListener('gospa:hydrated', handler);
			});
		});

		await Promise.all(promises);
	}

	/**
	 * Preload scripts for faster hydration.
	 */
	private preloadScripts(scripts: string[]): void {
		for (const src of scripts) {
			const link = document.createElement('link');
			link.rel = 'preload';
			link.as = 'script';
			link.href = src;
			document.head.appendChild(link);
		}
	}

	/**
	 * Force hydrate an island by ID.
	 */
	forceHydrate(id: string): void {
		const island = this.islands.get(id);
		if (island && island.state === 'pending') {
			// Cancel any pending triggers
			this.cancelTriggers(id);

			// Add to queue and process
			this.hydrationQueue.push(island);
			this.processQueue();
		}
	}

	/**
	 * Cancel pending hydration triggers.
	 */
	private cancelTriggers(id: string): void {
		// Cancel idle callback
		const idleCallback = this.idleCallbacks.get(id);
		if (idleCallback !== undefined) {
			cancelIdleCallback(idleCallback);
			this.idleCallbacks.delete(id);
		}

		// Disconnect observer
		const observer = this.observers.get(id);
		if (observer) {
			observer.disconnect();
			this.observers.delete(id);
		}

		// Remove interaction handlers
		const handlers = this.interactionHandlers.get(id);
		if (handlers) {
			const island = this.islands.get(id);
			if (island?.element) {
				const events = ['click', 'focus', 'mouseenter', 'touchstart'];
				for (let i = 0; i < events.length; i++) {
					island.element.removeEventListener(events[i], handlers[i]);
				}
			}
			this.interactionHandlers.delete(id);
		}
	}

	/**
	 * Get island state.
	 */
	getIslandState(id: string): HydrationState | undefined {
		return this.islands.get(id)?.state;
	}

	/**
	 * Get all pending islands.
	 */
	getPendingIslands(): TrackedIsland[] {
		return Array.from(this.islands.values()).filter((i) => i.state === 'pending');
	}

	/**
	 * Get all hydrated islands.
	 */
	getHydratedIslands(): TrackedIsland[] {
		return Array.from(this.islands.values()).filter((i) => i.state === 'hydrated');
	}

	/**
	 * Get statistics.
	 */
	getStats(): {
		total: number;
		pending: number;
		hydrating: number;
		hydrated: number;
		errors: number;
	} {
		const islands = Array.from(this.islands.values());
		return {
			total: islands.length,
			pending: islands.filter((i) => i.state === 'pending').length,
			hydrating: islands.filter((i) => i.state === 'hydrating').length,
			hydrated: islands.filter((i) => i.state === 'hydrated').length,
			errors: islands.filter((i) => i.state === 'error').length,
		};
	}

	/**
	 * Cleanup and destroy the scheduler.
	 */
	destroy(): void {
		// Cancel all idle callbacks
		for (const callbackId of this.idleCallbacks.values()) {
			cancelIdleCallback(callbackId);
		}
		this.idleCallbacks.clear();

		// Disconnect all observers
		for (const observer of this.observers.values()) {
			observer.disconnect();
		}
		this.observers.clear();

		// Clear interaction handlers
		this.interactionHandlers.clear();

		// Clear islands
		this.islands.clear();
		this.hydrationQueue = [];
	}
}

// Global scheduler instance
let globalScheduler: PriorityScheduler | null = null;

/**
 * Get or create the global priority scheduler.
 */
export function getPriorityScheduler(config?: Partial<PriorityConfig>): PriorityScheduler {
	if (!globalScheduler) {
		globalScheduler = new PriorityScheduler(config);
	}
	return globalScheduler;
}

/**
 * Initialize priority-based hydration from a plan.
 */
export function initPriorityHydration(plan: HydrationPlan): PriorityScheduler {
	const scheduler = getPriorityScheduler();
	scheduler.registerPlan(plan);
	return scheduler;
}

// Export for module usage
export default PriorityScheduler;
