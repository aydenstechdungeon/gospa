/**
 * HMR (Hot Module Replacement) Client Runtime
 * Handles module updates with state preservation for GoSPA.
 */

// HMR Message Types
interface HMRMessage {
	type: 'update' | 'reload' | 'error' | 'state-preserve' | 'connected';
	path?: string;
	moduleId?: string;
	event?: string;
	state?: Record<string, unknown>;
	error?: string;
	timestamp: number;
}

// Module state registry
interface ModuleState {
	moduleId: string;
	state: Record<string, unknown>;
	timestamp: number;
}

// HMR update handler type
type HMRUpdateHandler = (msg: HMRMessage) => void | Promise<void>;

// HMR error handler type
type HMRErrorHandler = (error: string) => void;

// State preservation function type
type StatePreservationFn = () => Record<string, unknown>;

// HMR Client Configuration
interface HMRClientConfig {
	wsUrl?: string;
	reconnectInterval?: number;
	maxReconnectAttempts?: number;
	onUpdate?: HMRUpdateHandler;
	onError?: HMRErrorHandler;
	onConnect?: () => void;
	onDisconnect?: () => void;
}

// Module registry for HMR
interface ModuleRegistry {
	[moduleId: string]: {
		version: number;
		exports: Record<string, unknown>;
		accept?: boolean;
		deps?: string[];
	};
}

/**
 * HMRClient manages WebSocket connection and module updates
 */
export class HMRClient {
	private ws: WebSocket | null = null;
	private config: Required<HMRClientConfig>;
	private reconnectAttempts = 0;
	private moduleRegistry: ModuleRegistry = {};
	private stateRegistry: Map<string, ModuleState> = new Map();
	private isConnecting = false;
	private updateQueue: HMRMessage[] = [];
	private isProcessing = false;

	constructor(config: HMRClientConfig = {}) {
		this.config = {
			wsUrl: config.wsUrl || `ws://${window.location.host}/__hmr`,
			reconnectInterval: config.reconnectInterval || 1000,
			maxReconnectAttempts: config.maxReconnectAttempts || 10,
			onUpdate: config.onUpdate || (() => {}),
			onError: config.onError || ((err) => console.error('[HMR]', err)),
			onConnect: config.onConnect || (() => {}),
			onDisconnect: config.onDisconnect || (() => {}),
		};

		// Set up global handlers
		this.setupGlobalHandlers();
	}

	/**
	 * Connect to HMR server
	 */
	connect(): void {
		if (this.ws?.readyState === WebSocket.OPEN || this.isConnecting) {
			return;
		}

		this.isConnecting = true;

		try {
			this.ws = new WebSocket(this.config.wsUrl);
			this.setupWebSocketHandlers();
		} catch (error) {
			this.isConnecting = false;
			this.config.onError(`Failed to connect: ${error}`);
			this.scheduleReconnect();
		}
	}

	/**
	 * Disconnect from HMR server
	 */
	disconnect(): void {
		if (this.ws) {
			this.ws.close();
			this.ws = null;
		}
	}

	/**
	 * Set up WebSocket event handlers
	 */
	private setupWebSocketHandlers(): void {
		if (!this.ws) return;

		this.ws.onopen = () => {
			this.isConnecting = false;
			this.reconnectAttempts = 0;
			console.log('[HMR] Connected');
			this.config.onConnect();

			// Process queued updates
			this.processUpdateQueue();
		};

		this.ws.onmessage = (event) => {
			try {
				const msg: HMRMessage = JSON.parse(event.data);
				this.handleMessage(msg);
			} catch (error) {
				this.config.onError(`Invalid message: ${error}`);
			}
		};

		this.ws.onclose = () => {
			this.isConnecting = false;
			console.log('[HMR] Disconnected');
			this.config.onDisconnect();
			this.scheduleReconnect();
		};

		this.ws.onerror = (error) => {
			this.isConnecting = false;
			this.config.onError('WebSocket error');
		};
	}

	/**
	 * Schedule reconnection attempt
	 */
	private scheduleReconnect(): void {
		if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
			this.config.onError('Max reconnection attempts reached');
			return;
		}

		this.reconnectAttempts++;
		console.log(`[HMR] Reconnecting in ${this.config.reconnectInterval}ms (attempt ${this.reconnectAttempts})`);

		setTimeout(() => {
			this.connect();
		}, this.config.reconnectInterval);
	}

	/**
	 * Handle incoming HMR message
	 */
	private handleMessage(msg: HMRMessage): void {
		switch (msg.type) {
			case 'connected':
				console.log('[HMR] Server connected');
				break;

			case 'update':
				this.queueUpdate(msg);
				break;

			case 'reload':
				console.log('[HMR] Full reload required');
				this.preserveAllStates();
				window.location.reload();
				break;

			case 'error':
				this.config.onError(msg.error || 'Unknown error');
				break;

			case 'state-preserve':
				if (msg.moduleId && msg.state) {
					this.restoreState(msg.moduleId, msg.state);
				}
				break;
		}
	}

	/**
	 * Queue update for processing
	 */
	private queueUpdate(msg: HMRMessage): void {
		this.updateQueue.push(msg);
		this.processUpdateQueue();
	}

	/**
	 * Process queued updates
	 */
	private async processUpdateQueue(): Promise<void> {
		if (this.isProcessing || this.updateQueue.length === 0) {
			return;
		}

		this.isProcessing = true;

		while (this.updateQueue.length > 0) {
			const msg = this.updateQueue.shift();
			if (msg) {
				await this.applyUpdate(msg);
			}
		}

		this.isProcessing = false;
	}

	/**
	 * Apply an HMR update
	 */
	private async applyUpdate(msg: HMRMessage): Promise<void> {
		console.log(`[HMR] Applying update for: ${msg.moduleId}`);

		// Preserve current state before update
		if (msg.moduleId) {
			this.preserveModuleState(msg.moduleId);
		}

		// Call custom update handler
		try {
			await this.config.onUpdate(msg);
		} catch (error) {
			this.config.onError(`Update failed: ${error}`);
			return;
		}

		// Restore state after update
		if (msg.moduleId && msg.state) {
			this.restoreState(msg.moduleId, msg.state);
		}
	}

	/**
	 * Set up global handlers for state preservation
	 */
	private setupGlobalHandlers(): void {
		// Expose HMR API globally
		(window as unknown as { __gospaHMR: HMRClient }).__gospaHMR = this;

		// State preservation before unload
		window.addEventListener('beforeunload', () => {
			this.preserveAllStates();
		});

		// Handle visibility change for mobile
		document.addEventListener('visibilitychange', () => {
			if (document.visibilityState === 'hidden') {
				this.preserveAllStates();
			}
		});
	}

	/**
	 * Register a module for HMR
	 */
	registerModule(moduleId: string, exports: Record<string, unknown>, deps?: string[]): void {
		this.moduleRegistry[moduleId] = {
			version: 0,
			exports,
			accept: true,
			deps,
		};
	}

	/**
	 * Accept updates for a module
	 */
	accept(moduleId: string): void {
		if (this.moduleRegistry[moduleId]) {
			this.moduleRegistry[moduleId].accept = true;
		}
	}

	/**
	 * Preserve state for a specific module
	 */
	preserveModuleState(moduleId: string): void {
		const state = this.extractModuleState(moduleId);
		if (state && Object.keys(state).length > 0) {
			this.stateRegistry.set(moduleId, {
				moduleId,
				state,
				timestamp: Date.now(),
			});

			// Send to server
			this.sendState(moduleId, state);
		}
	}

	/**
	 * Extract state from a module
	 */
	private extractModuleState(moduleId: string): Record<string, unknown> | null {
		// Try to get state from registered state getter
		const stateGetter = (window as unknown as { __gospaGetState?: (id: string) => Record<string, unknown> | null }).__gospaGetState;
		if (stateGetter) {
			return stateGetter(moduleId);
		}

		// Fallback: try to extract from module exports
		const module = this.moduleRegistry[moduleId];
		if (module?.exports) {
			const state: Record<string, unknown> = {};
			for (const [key, value] of Object.entries(module.exports)) {
				// Only preserve serializable state
				if (this.isSerializable(value)) {
					state[key] = value;
				}
			}
			return state;
		}

		return null;
	}

	/**
	 * Check if a value is serializable
	 */
	private isSerializable(value: unknown): boolean {
		if (value === null || value === undefined) return true;
		if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return true;
		if (value instanceof Date) return true;
		if (Array.isArray(value)) return value.every((v) => this.isSerializable(v));
		if (typeof value === 'object') {
			return Object.values(value as Record<string, unknown>).every((v) => this.isSerializable(v));
		}
		return false;
	}

	/**
	 * Preserve all module states
	 */
	preserveAllStates(): void {
		for (const moduleId of Object.keys(this.moduleRegistry)) {
			this.preserveModuleState(moduleId);
		}
	}

	/**
	 * Restore state for a module
	 */
	restoreState(moduleId: string, state: Record<string, unknown>): void {
		const module = this.moduleRegistry[moduleId];
		if (module?.exports) {
			for (const [key, value] of Object.entries(state)) {
				if (key in module.exports && typeof module.exports[key] === 'object') {
					Object.assign(module.exports[key] as Record<string, unknown>, value);
				} else {
					module.exports[key] = value;
				}
			}
		}

		// Notify state restoration
		const stateSetter = (window as unknown as { __gospaSetState?: (id: string, state: Record<string, unknown>) => void }).__gospaSetState;
		if (stateSetter) {
			stateSetter(moduleId, state);
		}
	}

	/**
	 * Send state to server
	 */
	private sendState(moduleId: string, state: Record<string, unknown>): void {
		if (this.ws?.readyState === WebSocket.OPEN) {
			this.ws.send(JSON.stringify({
				type: 'state-preserve',
				moduleId,
				state,
			}));
		}
	}

	/**
	 * Request state from server
	 */
	requestState(moduleId: string): void {
		if (this.ws?.readyState === WebSocket.OPEN) {
			this.ws.send(JSON.stringify({
				type: 'state-request',
				moduleId,
			}));
		}
	}

	/**
	 * Report error to server
	 */
	reportError(error: string): void {
		if (this.ws?.readyState === WebSocket.OPEN) {
			this.ws.send(JSON.stringify({
				type: 'error',
				error,
			}));
		}
	}

	/**
	 * Get current state for a module
	 */
	getState(moduleId: string): Record<string, unknown> | undefined {
		return this.stateRegistry.get(moduleId)?.state;
	}

	/**
	 * Check if connected
	 */
	isConnected(): boolean {
		return this.ws?.readyState === WebSocket.OPEN;
	}
}

// CSS HMR handling
export class CSSHMR {
	private static styleSheets: Map<string, HTMLLinkElement> = new Map();

	/**
	 * Register a stylesheet for HMR
	 */
	static registerStyle(href: string, element: HTMLLinkElement): void {
		this.styleSheets.set(href, element);
	}

	/**
	 * Update a stylesheet
	 */
	static updateStyle(href: string): void {
		const link = this.styleSheets.get(href);
		if (link) {
			// Add timestamp to force reload
			const url = new URL(link.href);
			url.searchParams.set('t', Date.now().toString());
			link.href = url.toString();
		}
	}

	/**
	 * Remove a stylesheet
	 */
	static removeStyle(href: string): void {
		const link = this.styleSheets.get(href);
		if (link) {
			link.remove();
			this.styleSheets.delete(href);
		}
	}
}

// Template HMR handling
export class TemplateHMR {
	private static templates: Map<string, string> = new Map();

	/**
	 * Register a template
	 */
	static registerTemplate(id: string, content: string): void {
		this.templates.set(id, content);
	}

	/**
	 * Update a template
	 */
	static updateTemplate(id: string, content: string): void {
		this.templates.set(id, content);

		// Dispatch custom event for template update
		window.dispatchEvent(new CustomEvent('gospa:template-update', {
			detail: { id, content },
		}));
	}

	/**
	 * Get a template
	 */
	static getTemplate(id: string): string | undefined {
		return this.templates.get(id);
	}
}

// Create global HMR client instance
let globalHMRClient: HMRClient | null = null;

/**
 * Initialize HMR client
 */
export function initHMR(config?: HMRClientConfig): HMRClient {
	if (!globalHMRClient) {
		globalHMRClient = new HMRClient(config);
		globalHMRClient.connect();
	}
	return globalHMRClient;
}

/**
 * Get global HMR client
 */
export function getHMR(): HMRClient | null {
	return globalHMRClient;
}

/**
 * Register module for HMR
 */
export function registerHMRModule(moduleId: string, exports: Record<string, unknown>, deps?: string[]): void {
	globalHMRClient?.registerModule(moduleId, exports, deps);
}

/**
 * Accept HMR updates
 */
export function acceptHMR(moduleId: string): void {
	globalHMRClient?.accept(moduleId);
}

// Auto-initialize if in browser
if (typeof window !== 'undefined') {
	// Wait for DOM ready
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', () => initHMR());
	} else {
		initHMR();
	}
}

export default HMRClient;
