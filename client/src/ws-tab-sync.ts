// GoSPA WebSocket Tab Synchronization
// Uses BroadcastChannel to share WebSocket state across browser tabs
// This reduces server load and provides instant state sync between tabs

import { Rune, batch } from './state.ts';

/**
 * Tab sync message types
 */
type TabSyncMessageType = 
	| 'state-update'
	| 'state-sync'
	| 'ws-connected'
	| 'ws-disconnected'
	| 'action'
	| 'ping'
	| 'pong';

/**
 * Tab sync message structure
 */
interface TabSyncMessage {
	type: TabSyncMessageType;
	tabId: string;
	timestamp: number;
	payload?: unknown;
}

/**
 * Tab sync configuration
 */
export interface TabSyncConfig {
	/** Channel name for BroadcastChannel */
	channelName?: string;
	/** Enable tab sync (default: true) */
	enabled?: boolean;
	/** Ping interval to detect active tabs (default: 5000ms) */
	pingInterval?: number;
	/** Timeout for considering a tab dead (default: 10000ms) */
	tabTimeout?: number;
}

/**
 * Tab information
 */
interface TabInfo {
	id: string;
	lastSeen: number;
	isLeader: boolean;
}

/**
 * WebSocket Tab Sync Manager
 * Coordinates WebSocket connections across browser tabs
 */
export class WSTabSync {
	private channel: BroadcastChannel | null = null;
	private tabId: string;
	private tabs: Map<string, TabInfo> = new Map();
	private isLeader: boolean = false;
	private pingTimer: ReturnType<typeof setInterval> | null = null;
	private config: Required<TabSyncConfig>;
	private stateRunes: Map<string, Rune<unknown>> = new Map();
	private onStateUpdate: ((key: string, value: unknown) => void) | null = null;
	private onAction: ((action: string, payload: unknown) => void) | null = null;

	constructor(config: TabSyncConfig = {}) {
		this.tabId = `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
		this.config = {
			channelName: config.channelName ?? 'gospa-ws-sync',
			enabled: config.enabled ?? true,
			pingInterval: config.pingInterval ?? 5000,
			tabTimeout: config.tabTimeout ?? 10000,
		};

		if (this.config.enabled && typeof BroadcastChannel !== 'undefined') {
			this.init();
		}
	}

	/**
	 * Initialize the BroadcastChannel
	 */
	private init(): void {
		try {
			this.channel = new BroadcastChannel(this.config.channelName);
			this.channel.onmessage = (event) => this.handleMessage(event.data);

			// Announce ourselves
			this.broadcast({
				type: 'ping',
				tabId: this.tabId,
				timestamp: Date.now(),
			});

			// Start ping interval
			this.pingTimer = setInterval(() => {
				this.broadcast({
					type: 'ping',
					tabId: this.tabId,
					timestamp: Date.now(),
				});
				this.cleanupDeadTabs();
			}, this.config.pingInterval);

			// Handle tab close
			window.addEventListener('beforeunload', () => {
				this.broadcast({
					type: 'ws-disconnected',
					tabId: this.tabId,
					timestamp: Date.now(),
				});
			});

			console.log(`[GoSPA Tab Sync] Initialized with tab ID: ${this.tabId}`);
		} catch (error) {
			console.warn('[GoSPA Tab Sync] BroadcastChannel not available:', error);
		}
	}

	/**
	 * Handle incoming messages from other tabs
	 */
	private handleMessage(message: TabSyncMessage): void {
		if (message.tabId === this.tabId) return; // Ignore own messages

		// Update tab info
		this.tabs.set(message.tabId, {
			id: message.tabId,
			lastSeen: Date.now(),
			isLeader: false,
		});

		switch (message.type) {
			case 'ping':
				// Respond with pong
				this.broadcast({
					type: 'pong',
					tabId: this.tabId,
					timestamp: Date.now(),
				});
				this.electLeader();
				break;

			case 'pong':
				// Tab is alive
				this.electLeader();
				break;

			case 'state-update':
				// Apply state update from another tab
				if (message.payload && typeof message.payload === 'object') {
					const { key, value } = message.payload as { key: string; value: unknown };
					const rune = this.stateRunes.get(key);
					if (rune) {
						batch(() => {
							rune.set(value);
						});
					}
					this.onStateUpdate?.(key, value);
				}
				break;

			case 'state-sync':
				// Full state sync from leader
				if (message.payload && typeof message.payload === 'object') {
					const state = message.payload as Record<string, unknown>;
					batch(() => {
						for (const [key, value] of Object.entries(state)) {
							const rune = this.stateRunes.get(key);
							if (rune) {
								rune.set(value);
							}
						}
					});
				}
				break;

			case 'action':
				// Forward action to handler
				if (message.payload && typeof message.payload === 'object') {
					const { action, payload } = message.payload as { action: string; payload: unknown };
					this.onAction?.(action, payload);
				}
				break;

			case 'ws-connected':
				// Another tab connected to WebSocket
				console.log(`[GoSPA Tab Sync] Tab ${message.tabId} connected`);
				this.electLeader();
				break;

			case 'ws-disconnected':
				// Another tab disconnected
				this.tabs.delete(message.tabId);
				this.electLeader();
				break;
		}
	}

	/**
	 * Broadcast a message to all tabs
	 */
	private broadcast(message: TabSyncMessage): void {
		if (this.channel) {
			try {
				this.channel.postMessage(message);
			} catch (error) {
				console.warn('[GoSPA Tab Sync] Failed to broadcast:', error);
			}
		}
	}

	/**
	 * Elect a leader tab (oldest tab becomes leader)
	 */
	private electLeader(): void {
		const now = Date.now();
		let oldestTab: TabInfo | null = null;

		// Include ourselves
		const allTabs: TabInfo[] = [
			{ id: this.tabId, lastSeen: now, isLeader: false },
			...Array.from(this.tabs.values()),
		];

		for (const tab of allTabs) {
			if (!oldestTab || tab.lastSeen < oldestTab.lastSeen) {
				oldestTab = tab;
			}
		}

		const wasLeader = this.isLeader;
		this.isLeader = oldestTab?.id === this.tabId;

		if (this.isLeader && !wasLeader) {
			console.log('[GoSPA Tab Sync] This tab is now the leader');
			// Sync state to other tabs
			this.syncStateToTabs();
		}
	}

	/**
	 * Clean up tabs that haven't been seen recently
	 */
	private cleanupDeadTabs(): void {
		const now = Date.now();
		for (const [tabId, tab] of this.tabs) {
			if (now - tab.lastSeen > this.config.tabTimeout) {
				this.tabs.delete(tabId);
				console.log(`[GoSPA Tab Sync] Removed dead tab: ${tabId}`);
			}
		}
		this.electLeader();
	}

	/**
	 * Sync current state to all tabs
	 */
	private syncStateToTabs(): void {
		const state: Record<string, unknown> = {};
		for (const [key, rune] of this.stateRunes) {
			state[key] = rune.get();
		}

		this.broadcast({
			type: 'state-sync',
			tabId: this.tabId,
			timestamp: Date.now(),
			payload: state,
		});
	}

	/**
	 * Register a state rune for synchronization
	 */
	registerState<T>(key: string, rune: Rune<T>): void {
		this.stateRunes.set(key, rune as Rune<unknown>);

		// Subscribe to changes and broadcast
		rune.subscribe((value) => {
			if (!this.isLeader) return; // Only leader broadcasts state changes

			this.broadcast({
				type: 'state-update',
				tabId: this.tabId,
				timestamp: Date.now(),
				payload: { key, value },
			});
		});
	}

	/**
	 * Unregister a state rune
	 */
	unregisterState(key: string): void {
		this.stateRunes.delete(key);
	}

	/**
	 * Set callback for state updates from other tabs
	 */
	onStateChange(callback: (key: string, value: unknown) => void): void {
		this.onStateUpdate = callback;
	}

	/**
	 * Set callback for actions from other tabs
	 */
	onActionReceived(callback: (action: string, payload: unknown) => void): void {
		this.onAction = callback;
	}

	/**
	 * Broadcast an action to all tabs
	 */
	broadcastAction(action: string, payload: unknown = {}): void {
		this.broadcast({
			type: 'action',
			tabId: this.tabId,
			timestamp: Date.now(),
			payload: { action, payload },
		});
	}

	/**
	 * Check if this tab is the leader
	 */
	getIsLeader(): boolean {
		return this.isLeader;
	}

	/**
	 * Get the tab ID
	 */
	getTabId(): string {
		return this.tabId;
	}

	/**
	 * Get count of active tabs
	 */
	getActiveTabCount(): number {
		return this.tabs.size + 1; // +1 for ourselves
	}

	/**
	 * Destroy the tab sync manager
	 */
	destroy(): void {
		if (this.pingTimer) {
			clearInterval(this.pingTimer);
			this.pingTimer = null;
		}

		if (this.channel) {
			this.broadcast({
				type: 'ws-disconnected',
				tabId: this.tabId,
				timestamp: Date.now(),
			});
			this.channel.close();
			this.channel = null;
		}

		this.tabs.clear();
		this.stateRunes.clear();
	}
}

/**
 * Create a tab sync manager
 */
export function createTabSync(config?: TabSyncConfig): WSTabSync {
	return new WSTabSync(config);
}

/**
 * Global tab sync instance
 */
let globalTabSync: WSTabSync | null = null;

/**
 * Get or create the global tab sync instance
 */
export function getTabSync(config?: TabSyncConfig): WSTabSync {
	if (!globalTabSync) {
		globalTabSync = new WSTabSync(config);
	}
	return globalTabSync;
}

/**
 * Destroy the global tab sync instance
 */
export function destroyTabSync(): void {
	if (globalTabSync) {
		globalTabSync.destroy();
		globalTabSync = null;
	}
}
