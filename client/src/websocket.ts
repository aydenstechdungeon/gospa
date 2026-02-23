// WebSocket client for real-time state synchronization

import { Rune, batch } from './state.ts';

// Connection states
export type ConnectionState = 'connecting' | 'connected' | 'disconnecting' | 'disconnected';

// Message types matching server
export type MessageType = 'init' | 'update' | 'sync' | 'error' | 'ping' | 'pong' | 'action';

export interface StateMessage {
	type: MessageType;
	componentId?: string;
	action?: string;
	data?: Record<string, unknown>;
	payload?: Record<string, unknown>;
	state?: Record<string, unknown>; // Server global state from SendState()
	diff?: Record<string, unknown>;
	error?: string;
	timestamp?: number;
	sessionToken?: string;
	clientId?: string;
}

// Valid message types for validation
const VALID_MESSAGE_TYPES: Set<string> = new Set(['init', 'update', 'sync', 'error', 'ping', 'pong', 'action']);

// Validate WebSocket message structure
function validateMessage(raw: unknown): StateMessage | null {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
		return null;
	}
	
	const msg = raw as Record<string, unknown>;
	
	// Required: type field must be a valid message type
	if (typeof msg.type !== 'string' || !VALID_MESSAGE_TYPES.has(msg.type)) {
		return null;
	}
	
	// Validate optional fields have correct types
	const validated: StateMessage = { type: msg.type as MessageType };
	
	if (typeof msg.componentId === 'string') validated.componentId = msg.componentId;
	if (typeof msg.action === 'string') validated.action = msg.action;
	if (msg.data && typeof msg.data === 'object' && !Array.isArray(msg.data)) {
		validated.data = msg.data as Record<string, unknown>;
	}
	if (msg.payload && typeof msg.payload === 'object' && !Array.isArray(msg.payload)) {
		validated.payload = msg.payload as Record<string, unknown>;
	}
	if (msg.state && typeof msg.state === 'object' && !Array.isArray(msg.state)) {
		validated.state = msg.state as Record<string, unknown>;
	}
	if (msg.diff && typeof msg.diff === 'object' && !Array.isArray(msg.diff)) {
		validated.diff = msg.diff as Record<string, unknown>;
	}
	if (typeof msg.error === 'string') validated.error = msg.error;
	if (typeof msg.timestamp === 'number') validated.timestamp = msg.timestamp;
	if (typeof msg.sessionToken === 'string') validated.sessionToken = msg.sessionToken;
	if (typeof msg.clientId === 'string') validated.clientId = msg.clientId;
	
	return validated;
}

// Session storage key
const SESSION_STORAGE_KEY = 'gospa_session';

// Session data stored in localStorage
interface SessionData {
	token: string;
	clientId: string;
}

// WebSocket configuration
export interface WebSocketConfig {
	url: string;
	reconnect?: boolean;
	reconnectInterval?: number;
	maxReconnectAttempts?: number;
	heartbeatInterval?: number;
	onOpen?: () => void;
	onClose?: (event: CloseEvent) => void;
	onError?: (error: Event) => void;
	onMessage?: (message: StateMessage) => void;
}

// Helper functions for session persistence
function loadSession(): SessionData | null {
	try {
		const stored = localStorage.getItem(SESSION_STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored) as SessionData;
		}
	} catch (e) {
		console.warn('[GoSPA] Failed to load session:', e);
	}
	return null;
}

function saveSession(data: SessionData): void {
	try {
		localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(data));
	} catch (e) {
		console.warn('[GoSPA] Failed to save session:', e);
	}
}

function clearSession(): void {
	try {
		localStorage.removeItem(SESSION_STORAGE_KEY);
	} catch (e) {
		console.warn('[GoSPA] Failed to clear session:', e);
	}
}

// WebSocket client
export class WSClient {
	private ws: WebSocket | null = null;
	private config: Required<WebSocketConfig>;
	private reconnectAttempts = 0;
	private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
	private messageQueue: StateMessage[] = [];
	private connectionState: Rune<ConnectionState>;
	private pendingRequests = new Map<string, { resolve: (value: unknown) => void; reject: (error: Error) => void }>();
	private requestId = 0;
	private sessionData: SessionData | null = null;

	constructor(config: WebSocketConfig) {
		this.config = {
			reconnect: true,
			reconnectInterval: 1000,
			maxReconnectAttempts: 10,
			heartbeatInterval: 30000,
			onOpen: () => { },
			onClose: () => { },
			onError: () => { },
			onMessage: () => { },
			...config
		};
		this.connectionState = new Rune<ConnectionState>('disconnected');
		// Load existing session on construction
		this.sessionData = loadSession();
	}

	get state(): ConnectionState {
		return this.connectionState.get();
	}

	get isConnected(): boolean {
		return this.connectionState.get() === 'connected';
	}

	connect(): Promise<void> {
		return new Promise((resolve, reject) => {
			if (this.ws?.readyState === WebSocket.OPEN) {
				resolve();
				return;
			}

			this.connectionState.set('connecting');

			// SECURITY: Do NOT pass session token in URL - it leaks in logs/referrers
			// Instead, send it as the first message after connection opens
			try {
				this.ws = new WebSocket(this.config.url);
			} catch (error) {
				this.connectionState.set('disconnected');
				reject(error);
				return;
			}

			this.ws.onopen = () => {
				this.connectionState.set('connected');
				this.reconnectAttempts = 0;
				this.startHeartbeat();

				// SECURITY: Send session token as first message (not in URL)
				// Server will validate and associate this connection with the session
				if (this.sessionData?.token) {
					this.send({
						type: 'init',
						sessionToken: this.sessionData.token,
						clientId: this.sessionData.clientId
					});
				}

				this.flushMessageQueue();

				// State HMR: Request fresh state from server on reconnect
				// This softly patches the runes locally without refreshing the page!
				this.send({ type: 'sync' });

				this.config.onOpen();
				resolve();
			};

			this.ws.onclose = (event) => {
				this.connectionState.set('disconnected');
				this.stopHeartbeat();
				this.config.onClose(event);

				if (this.config.reconnect && this.reconnectAttempts < this.config.maxReconnectAttempts) {
					this.scheduleReconnect();
				}
			};

			this.ws.onerror = (error) => {
				this.config.onError(error);
				if (this.connectionState.get() === 'connecting') {
					reject(new Error('WebSocket connection failed'));
				}
			};

			this.ws.onmessage = (event) => {
				this.handleMessage(event.data);
			};
		});
	}

	disconnect(): void {
		if (this.ws) {
			this.connectionState.set('disconnecting');
			this.stopHeartbeat();
			this.ws.close(1000, 'Client disconnect');
			this.ws = null;
			this.connectionState.set('disconnected');
		}
	}

	private scheduleReconnect(): void {
		this.reconnectAttempts++;
		const delay = this.config.reconnectInterval * Math.min(this.reconnectAttempts, 5);

		setTimeout(() => {
			if (this.connectionState.get() === 'disconnected') {
				this.connect().catch(() => { });
			}
		}, delay);
	}

	private startHeartbeat(): void {
		this.heartbeatTimer = setInterval(() => {
			this.send({ type: 'ping' });
		}, this.config.heartbeatInterval);
	}

	private stopHeartbeat(): void {
		if (this.heartbeatTimer) {
			clearInterval(this.heartbeatTimer);
			this.heartbeatTimer = null;
		}
	}

	private flushMessageQueue(): void {
		while (this.messageQueue.length > 0 && this.isConnected) {
			const message = this.messageQueue.shift();
			if (message) {
				this.send(message);
			}
		}
	}

	send(message: StateMessage): void {
		if (this.ws?.readyState === WebSocket.OPEN) {
			this.ws.send(JSON.stringify(message));
		} else {
			this.messageQueue.push(message);
		}
	}

	sendWithResponse<T>(message: StateMessage): Promise<T> {
		return new Promise((resolve, reject) => {
			const id = `req_${++this.requestId}`;
			message.data = { ...message.data, _requestId: id };

			this.pendingRequests.set(id, {
				resolve: resolve as (value: unknown) => void,
				reject
			});

			this.send(message);

			// Timeout after 30 seconds
			setTimeout(() => {
				if (this.pendingRequests.has(id)) {
					this.pendingRequests.delete(id);
					reject(new Error('Request timeout'));
				}
			}, 30000);
		});
	}

	private handleMessage(data: string): void {
		try {
			const raw = JSON.parse(data);
			
			// SECURITY: Validate message structure before processing
			const message = validateMessage(raw);
			if (!message) {
				console.warn('[GoSPA] Received invalid WebSocket message, ignoring');
				return;
			}

			// Handle pong
			if (message.type === 'pong') {
				return;
			}

			// Save session data when server sends it (init message with session token)
			if (message.type === 'init' && message.sessionToken && message.clientId) {
				this.sessionData = { token: message.sessionToken, clientId: message.clientId };
				saveSession(this.sessionData);
			}

			// Handle response to pending request
			if (message.data?._responseId) {
				const id = message.data._responseId as string;
				const pending = this.pendingRequests.get(id);
				if (pending) {
					this.pendingRequests.delete(id);
					if (message.type === 'error') {
						pending.reject(new Error(message.error || 'Unknown error'));
					} else {
						pending.resolve(message.data);
					}
					// Do not return here, allow the payload to be processed as normal state update by onMessage hook below
				}
			}

			this.config.onMessage(message);
		} catch (error) {
			console.error('Failed to parse WebSocket message:', error);
		}
	}

	// Sync global state request
	requestSync(): void {
		this.send({ type: 'sync' });
	}

	// Send custom action to server
	sendAction(action: string, payload: any = {}): void {
		this.send({
			type: 'action',
			action,
			payload
		});
	}

	// Request state from server
	requestState(componentId: string): Promise<Record<string, unknown>> {
		return this.sendWithResponse({
			type: 'init',
			componentId
		});
	}
}

// Global action helper
export function sendAction(action: string, payload: any = {}): void {
	if (clientInstance) {
		clientInstance.sendAction(action, payload);
	} else {
		console.warn('[GoSPA] Cannot send action: WebSocket not initialized');
	}
}

// Singleton instance
let clientInstance: WSClient | null = null;

export function getWebSocketClient(): WSClient | null {
	return clientInstance;
}

export function initWebSocket(config: WebSocketConfig): WSClient {
	if (clientInstance) {
		clientInstance.disconnect();
	}
	clientInstance = new WSClient(config);
	return clientInstance;
}

// State synchronization helper
export interface SyncedStateOptions {
	componentId: string;
	key: string;
	ws?: WSClient;
	debounce?: number;
}

export function syncedRune<T>(
	initial: T,
	options: SyncedStateOptions
): Rune<T> {
	const rune = new Rune<T>(initial);
	const ws = options.ws || clientInstance;

	let isReverting = false;
	const originalSet = rune.set.bind(rune);

	rune.set = (newValue: T) => {
		if (isReverting) {
			originalSet(newValue);
			return;
		}

		// Optimistic UI Rollback: capture the previous verified state
		const backupValue = rune.get();
		originalSet(newValue);

		if (ws?.isConnected) {
			try {
				// We wrap it in a setTimeout for the debounce if needed
				const executeSync = () => {
					ws.send({
						type: 'update',
						payload: { key: options.key, value: newValue }
					});
				};

				if (options.debounce) {
					// NOTE: with debounce, rollback might get complicated if multiple sets occur,
					// but for this implementation we assume the standard Optimistic fire-and-forget.
					setTimeout(executeSync, options.debounce);
				} else {
					executeSync();
				}
			} catch (e) {
				console.warn('[GoSPA] Optimistic update failed, rolling back.', e);
				isReverting = true;
				originalSet(backupValue);
				isReverting = false;
			}
		} else {
			// Not connected, revert immediately
			console.warn('[GoSPA] WS disconnected, optimistic update rolled back.');
			isReverting = true;
			originalSet(backupValue);
			isReverting = false;
		}
	};

	return rune;
}

// Batch sync multiple state values
export function syncBatch(
	componentId: string,
	states: Record<string, Rune<unknown>>,
	ws?: WSClient
): void {
	const client = ws || clientInstance;
	if (!client?.isConnected) return;

	for (const [key, rune] of Object.entries(states)) {
		client.send({
			type: 'update',
			payload: { key, value: rune.get() }
		});
	}
}

// Apply server state updates
export function applyStateUpdate(
	states: Record<string, Rune<unknown>>,
	data: Record<string, unknown>
): void {
	batch(() => {
		for (const [key, value] of Object.entries(data)) {
			const rune = states[key];
			if (rune) {
				rune.set(value);
			}
		}
	});
}
