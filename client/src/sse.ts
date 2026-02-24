/**
 * Client-side Server-Sent Events (SSE) support for GoSPA
 * Provides real-time server-to-client push notifications with automatic reconnection
 */

// SSE Event types
export interface SSEEvent<T = unknown> {
	id?: string;
	event?: string;
	data: T;
	retry?: number;
}

// SSE Connection state
export type SSEConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error';

// SSE Configuration
export interface SSEConfig {
	/** URL endpoint for SSE connection */
	url: string;
	/** Enable automatic reconnection on disconnect */
	autoReconnect?: boolean;
	/** Maximum reconnection attempts (0 = unlimited) */
	maxRetries?: number;
	/** Initial reconnection delay in ms */
	reconnectDelay?: number;
	/** Maximum reconnection delay in ms */
	maxReconnectDelay?: number;
	/** Backoff multiplier for reconnection delay */
	backoffMultiplier?: number;
	/** Connection timeout in ms */
	timeout?: number;
	/** Custom headers */
	headers?: Record<string, string>;
	/** Enable debug logging */
	debug?: boolean;
	/** Last event ID for resumption */
	lastEventId?: string;
	/** Heartbeat interval to detect dead connections (ms) */
	heartbeatInterval?: number;
	/** Missed heartbeats before considering connection dead */
	missedHeartbeatsLimit?: number;
}

// SSE Event Handler
export type SSEEventHandler<T = unknown> = (event: SSEEvent<T>) => void;

// SSE Error Handler
export type SSEErrorHandler = (error: Error, attempt: number) => void;

// SSE State Change Handler
export type SSEStateHandler = (state: SSEConnectionState) => void;

/**
 * SSEClient manages a single SSE connection with automatic reconnection
 */
export class SSEClient {
	private config: Required<SSEConfig>;
	private eventSource: EventSource | null = null;
	private reconnectAttempts = 0;
	private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
	private connectionState: SSEConnectionState = 'disconnected';
	private eventHandlers: Map<string, Set<SSEEventHandler>> = new Map();
	private errorHandlers: Set<SSEErrorHandler> = new Set();
	private stateHandlers: Set<SSEStateHandler> = new Set();
	private lastEventId: string | null = null;
	private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
	private missedHeartbeats = 0;
	private isIntentionallyClosed = false;

	constructor(config: SSEConfig) {
		this.config = {
			url: config.url,
			autoReconnect: config.autoReconnect ?? true,
			maxRetries: config.maxRetries ?? 5,
			reconnectDelay: config.reconnectDelay ?? 1000,
			maxReconnectDelay: config.maxReconnectDelay ?? 30000,
			backoffMultiplier: config.backoffMultiplier ?? 2,
			timeout: config.timeout ?? 0,
			headers: config.headers ?? {},
			debug: config.debug ?? false,
			lastEventId: config.lastEventId ?? '',
			heartbeatInterval: config.heartbeatInterval ?? 30000,
			missedHeartbeatsLimit: config.missedHeartbeatsLimit ?? 3,
		};

		// Set initial last event ID if provided
		if (this.config.lastEventId) {
			this.lastEventId = this.config.lastEventId;
		}
	}

	/**
	 * Connect to the SSE endpoint
	 */
	connect(): void {
		if (this.eventSource) {
			this.log('Already connected or connecting');
			return;
		}

		this.isIntentionallyClosed = false;
		this.setState('connecting');
		this.createConnection();
	}

	/**
	 * Disconnect from the SSE endpoint
	 */
	disconnect(): void {
		this.isIntentionallyClosed = true;
		this.cleanup();
		this.setState('disconnected');
		this.reconnectAttempts = 0;
	}

	/**
	 * Reconnect to the SSE endpoint
	 */
	reconnect(): void {
		this.cleanup();
		this.connect();
	}

	/**
	 * Subscribe to events
	 * @param event Event name (use 'message' for default events)
	 * @param handler Event handler
	 * @returns Unsubscribe function
	 */
	on<T = unknown>(event: string, handler: SSEEventHandler<T>): () => void {
		if (!this.eventHandlers.has(event)) {
			this.eventHandlers.set(event, new Set());
		}
		
		const handlers = this.eventHandlers.get(event)!;
		handlers.add(handler as SSEEventHandler);

		return () => {
			handlers.delete(handler as SSEEventHandler);
			if (handlers.size === 0) {
				this.eventHandlers.delete(event);
			}
		};
	}

	/**
	 * Subscribe to all messages (default event)
	 * @param handler Event handler
	 * @returns Unsubscribe function
	 */
	onMessage<T = unknown>(handler: SSEEventHandler<T>): () => void {
		return this.on('message', handler);
	}

	/**
	 * Subscribe to errors
	 * @param handler Error handler
	 * @returns Unsubscribe function
	 */
	onError(handler: SSEErrorHandler): () => void {
		this.errorHandlers.add(handler);
		return () => {
			this.errorHandlers.delete(handler);
		};
	}

	/**
	 * Subscribe to connection state changes
	 * @param handler State handler
	 * @returns Unsubscribe function
	 */
	onStateChange(handler: SSEStateHandler): () => void {
		this.stateHandlers.add(handler);
		return () => {
			this.stateHandlers.delete(handler);
		};
	}

	/**
	 * Get current connection state
	 */
	getState(): SSEConnectionState {
		return this.connectionState;
	}

	/**
	 * Check if connected
	 */
	isConnected(): boolean {
		return this.connectionState === 'connected';
	}

	/**
	 * Get last event ID
	 */
	getLastEventId(): string | null {
		return this.lastEventId;
	}

	/**
	 * Create the EventSource connection
	 */
	private createConnection(): void {
		try {
			// Build URL with last event ID for resumption
			const url = new URL(this.config.url, window.location.origin);
			if (this.lastEventId) {
				url.searchParams.set('lastEventId', this.lastEventId);
			}

			// EventSource doesn't support custom headers, so we use query params
			// for authentication if needed
			Object.entries(this.config.headers).forEach(([key, value]) => {
				if (key.toLowerCase() === 'authorization' || key.toLowerCase() === 'x-api-key') {
					url.searchParams.set(key, value);
				}
			});

			this.eventSource = new EventSource(url.toString());

			// Connection opened
			this.eventSource.onopen = () => {
				this.log('Connection opened');
				this.setState('connected');
				this.reconnectAttempts = 0;
				this.missedHeartbeats = 0;
				this.startHeartbeatMonitor();
			};

			// Generic message handler
			this.eventSource.onmessage = (event: MessageEvent) => {
				this.handleEvent('message', event);
			};

			// Error handler
			this.eventSource.onerror = (event: Event) => {
				this.log('Connection error:', event);
				this.setState('error');
				this.handleError(new Error('SSE connection error'));
			};

			// Listen for custom events
			this.setupCustomEventListeners();

		} catch (error) {
			this.log('Failed to create connection:', error);
			this.handleError(error instanceof Error ? error : new Error(String(error)));
		}
	}

	/**
	 * Setup listeners for custom event types
	 */
	private setupCustomEventListeners(): void {
		if (!this.eventSource) return;

		// Common SSE event types
		const customEvents = ['update', 'notification', 'ping', 'heartbeat', 'data'];
		
		customEvents.forEach(eventType => {
			this.eventSource!.addEventListener(eventType, (event: Event) => {
				this.handleEvent(eventType, event as MessageEvent);
			});
		});
	}

	/**
	 * Handle an incoming SSE event
	 */
	private handleEvent(eventType: string, event: MessageEvent): void {
		// Update last event ID
		if (event.lastEventId) {
			this.lastEventId = event.lastEventId;
		}

		// Reset heartbeat counter on any event
		this.missedHeartbeats = 0;

		// Parse data
		let data: unknown;
		try {
			data = event.data ? JSON.parse(event.data) : null;
		} catch {
			data = event.data;
		}

		// Handle heartbeat/ping events
		if (eventType === 'ping' || eventType === 'heartbeat') {
			this.log('Heartbeat received');
			return;
		}

		const sseEvent: SSEEvent = {
			id: event.lastEventId || undefined,
			event: eventType,
			data,
		};

		this.log(`Event received [${eventType}]:`, sseEvent);

		// Emit to handlers
		const handlers = this.eventHandlers.get(eventType);
		if (handlers) {
			handlers.forEach(handler => {
				try {
					handler(sseEvent);
				} catch (error) {
					this.log('Handler error:', error);
				}
			});
		}

		// Also emit to wildcard handlers
		const wildcardHandlers = this.eventHandlers.get('*');
		if (wildcardHandlers) {
			wildcardHandlers.forEach(handler => {
				try {
					handler(sseEvent);
				} catch (error) {
					this.log('Wildcard handler error:', error);
				}
			});
		}
	}

	/**
	 * Handle connection errors
	 */
	private handleError(error: Error): void {
		// Notify error handlers
		this.errorHandlers.forEach(handler => {
			try {
				handler(error, this.reconnectAttempts);
			} catch (e) {
				this.log('Error handler failed:', e);
			}
		});

		// Attempt reconnection
		if (this.config.autoReconnect && !this.isIntentionallyClosed) {
			this.attemptReconnect();
		}
	}

	/**
	 * Attempt to reconnect
	 */
	private attemptReconnect(): void {
		if (this.config.maxRetries > 0 && this.reconnectAttempts >= this.config.maxRetries) {
			this.log('Max reconnection attempts reached');
			this.setState('error');
			return;
		}

		this.reconnectAttempts++;
		const delay = Math.min(
			this.config.reconnectDelay * Math.pow(this.config.backoffMultiplier, this.reconnectAttempts - 1),
			this.config.maxReconnectDelay
		);

		this.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

		this.reconnectTimeout = setTimeout(() => {
			this.cleanup();
			this.setState('connecting');
			this.createConnection();
		}, delay);
	}

	/**
	 * Start heartbeat monitor
	 */
	private startHeartbeatMonitor(): void {
		this.stopHeartbeatMonitor();
		
		this.heartbeatTimer = setInterval(() => {
			this.missedHeartbeats++;
			
			if (this.missedHeartbeats >= this.config.missedHeartbeatsLimit) {
				this.log('Connection appears dead (missed heartbeats)');
				this.setState('error');
				this.handleError(new Error('Connection timeout - missed heartbeats'));
			}
		}, this.config.heartbeatInterval);
	}

	/**
	 * Stop heartbeat monitor
	 */
	private stopHeartbeatMonitor(): void {
		if (this.heartbeatTimer) {
			clearInterval(this.heartbeatTimer);
			this.heartbeatTimer = null;
		}
	}

	/**
	 * Set connection state
	 */
	private setState(state: SSEConnectionState): void {
		if (this.connectionState === state) return;
		
		this.connectionState = state;
		this.log(`State changed to: ${state}`);

		this.stateHandlers.forEach(handler => {
			try {
				handler(state);
			} catch (error) {
				this.log('State handler error:', error);
			}
		});
	}

	/**
	 * Cleanup resources
	 */
	private cleanup(): void {
		this.stopHeartbeatMonitor();
		
		if (this.reconnectTimeout) {
			clearTimeout(this.reconnectTimeout);
			this.reconnectTimeout = null;
		}

		if (this.eventSource) {
			this.eventSource.close();
			this.eventSource = null;
		}
	}

	/**
	 * Debug logging
	 */
	private log(...args: unknown[]): void {
		if (this.config.debug) {
			console.log('[SSE]', ...args);
		}
	}
}

/**
 * SSE Manager for handling multiple SSE connections
 */
export class SSEManager {
	private clients: Map<string, SSEClient> = new Map();
	private defaultConfig: Partial<SSEConfig> = {};

	/**
	 * Set default configuration for new connections
	 */
	setDefaultConfig(config: Partial<SSEConfig>): void {
		this.defaultConfig = { ...this.defaultConfig, ...config };
	}

	/**
	 * Create or get an SSE client
	 */
	client(name: string, config?: SSEConfig): SSEClient {
		if (!this.clients.has(name)) {
			if (!config) {
				throw new Error(`SSE client "${name}" not found and no config provided`);
			}
			
			const fullConfig = { ...this.defaultConfig, ...config };
			this.clients.set(name, new SSEClient(fullConfig));
		}
		
		return this.clients.get(name)!;
	}

	/**
	 * Connect a client by name
	 */
	connect(name: string, config?: SSEConfig): SSEClient {
		const client = this.client(name, config);
		client.connect();
		return client;
	}

	/**
	 * Disconnect a client by name
	 */
	disconnect(name: string): void {
		const client = this.clients.get(name);
		if (client) {
			client.disconnect();
		}
	}

	/**
	 * Disconnect all clients
	 */
	disconnectAll(): void {
		this.clients.forEach(client => client.disconnect());
	}

	/**
	 * Remove a client
	 */
	remove(name: string): void {
		const client = this.clients.get(name);
		if (client) {
			client.disconnect();
			this.clients.delete(name);
		}
	}

	/**
	 * Get all client names
	 */
	getClientNames(): string[] {
		return Array.from(this.clients.keys());
	}

	/**
	 * Check if a client exists
	 */
	has(name: string): boolean {
		return this.clients.has(name);
	}
}

// Singleton instance
let sseManager: SSEManager | null = null;

/**
 * Get the SSE manager singleton
 */
export function getSSEManager(): SSEManager {
	if (!sseManager) {
		sseManager = new SSEManager();
	}
	return sseManager;
}

/**
 * Create a new SSE client
 */
export function createSSEClient(config: SSEConfig): SSEClient {
	return new SSEClient(config);
}

/**
 * Connect to an SSE endpoint
 */
export function connectSSE(name: string, config: SSEConfig): SSEClient {
	return getSSEManager().connect(name, config);
}

// Export types
export type { SSEEvent as SSEEventType, SSEConnectionState as SSEConnectionStateType };
