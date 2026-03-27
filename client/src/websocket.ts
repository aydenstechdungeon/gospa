import { Rune, batch } from "./state.ts";
import { decode, encode } from "@msgpack/msgpack";

// Connection states
export type ConnectionState =
  | "connecting"
  | "connected"
  | "disconnecting"
  | "disconnected";

// Message types matching server
export type MessageType =
  | "init"
  | "update"
  | "sync"
  | "error"
  | "ping"
  | "pong"
  | "action";

export interface StateMessage {
  type:
    | string
    | "init"
    | "update"
    | "sync"
    | "error"
    | "ping"
    | "pong"
    | "action"
    | "patch"
    | "compressed";
  componentId?: string;
  action?: string;
  data?: any;
  payload?: Record<string, unknown>;
  state?: Record<string, unknown>; // Server global state from SendState()
  diff?: Record<string, unknown>;
  patch?: Record<string, unknown>;
  compressed?: boolean;
  error?: string;
  timestamp?: number;
  sessionToken?: string;
  clientId?: string;
  key?: string;
  value?: unknown;
  success?: boolean;
}

// Validate WebSocket message structure
function validateMessage(raw: unknown): StateMessage | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }

  const msg = raw as Record<string, unknown>;

  // Required: type field must be a string
  if (typeof msg.type !== "string") {
    return null;
  }

  // Validate optional fields have correct types
  const validated: StateMessage = { type: msg.type as any };

  if (typeof msg.componentId === "string")
    validated.componentId = msg.componentId;
  if (typeof msg.action === "string") validated.action = msg.action;
  if (typeof msg.key === "string") validated.key = msg.key;
  if (msg.value !== undefined) validated.value = msg.value;
  if (typeof msg.success === "boolean") validated.success = msg.success;

  if (msg.data !== undefined) {
    validated.data = msg.data;
  }
  if (
    msg.payload &&
    typeof msg.payload === "object" &&
    !Array.isArray(msg.payload)
  ) {
    validated.payload = msg.payload as Record<string, unknown>;
  }
  if (msg.state && typeof msg.state === "object" && !Array.isArray(msg.state)) {
    validated.state = msg.state as Record<string, unknown>;
  }
  if (msg.diff && typeof msg.diff === "object" && !Array.isArray(msg.diff)) {
    validated.diff = msg.diff as Record<string, unknown>;
  }
  if (msg.patch && typeof msg.patch === "object" && !Array.isArray(msg.patch)) {
    validated.patch = msg.patch as Record<string, unknown>;
  }
  if (typeof msg.compressed === "boolean")
    validated.compressed = msg.compressed;
  if (typeof msg.error === "string") validated.error = msg.error;
  if (typeof msg.timestamp === "number") validated.timestamp = msg.timestamp;
  if (typeof msg.sessionToken === "string")
    validated.sessionToken = msg.sessionToken;
  if (typeof msg.clientId === "string") validated.clientId = msg.clientId;

  return validated;
}

// Session storage key
const SESSION_COOKIE_KEY = "gospa_session";

// Session data stored in cookies
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
  onConnectionFailed?: (error: Error) => void;
  onMessage?: (message: StateMessage) => void;
  serializationFormat?: "json" | "msgpack";
  /**
   * Persist session token/clientId in sessionStorage.
   * Disabled by default to reduce token exposure in XSS scenarios.
   */
  persistSession?: boolean;
  /**
   * Persist unsent WS message queue across reloads.
   */
  persistQueueOnUnload?: boolean;
}

// Helper functions for session persistence
// NOTE: Token is now handled by HttpOnly cookies for security.
// We only persist the clientId if needed for local identification.
function loadSession(): SessionData | null {
  try {
    const saved = localStorage.getItem(SESSION_COOKIE_KEY);
    if (saved) {
      return JSON.parse(saved) as SessionData;
    }
  } catch (e) {
    console.warn("[GoSPA] Failed to load session:", e);
  }
  return null;
}

function saveSession(data: SessionData): void {
  try {
    // Only save non-sensitive identification if needed
    localStorage.setItem(
      SESSION_COOKIE_KEY,
      JSON.stringify({ clientId: data.clientId }),
    );
  } catch (e) {
    console.warn("[GoSPA] Failed to save session:", e);
  }
}

function clearSession(): void {
  try {
    localStorage.removeItem(SESSION_COOKIE_KEY);
  } catch (e) {
    console.warn("[GoSPA] Failed to clear session:", e);
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
  private pendingRequests = new Map<
    string,
    {
      resolve: (value: unknown) => void;
      reject: (error: Error) => void;
      timeout: ReturnType<typeof setTimeout>;
    }
  >();
  private requestId = 0;
  private sessionData: SessionData | null = null;
  private beforeUnloadHandler: (() => void) | null = null;

  constructor(config: WebSocketConfig) {
    this.config = {
      reconnect: true,
      reconnectInterval: 1000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
      onOpen: () => {},
      onClose: () => {},
      onError: () => {},
      onConnectionFailed: () => {},
      onMessage: () => {},
      serializationFormat: "json",
      persistSession: false,
      persistQueueOnUnload: true,
      ...config,
    };
    this.connectionState = new Rune<ConnectionState>("disconnected");
    this.sessionData = this.config.persistSession ? loadSession() : null;
    if (!this.config.persistSession) {
      clearSession();
    }

    if (this.config.persistQueueOnUnload) {
      try {
        const savedQueue = sessionStorage.getItem("gospa_ws_queue");
        if (savedQueue) {
          this.messageQueue = JSON.parse(savedQueue) || [];
          sessionStorage.removeItem("gospa_ws_queue");
        }
      } catch (e) {
        console.warn("[GoSPA] Failed to restore message queue:", e);
      }
    }

    this.beforeUnloadHandler = () => {
      if (!this.config.persistQueueOnUnload) return;
      if (this.messageQueue.length > 0) {
        try {
          sessionStorage.setItem(
            "gospa_ws_queue",
            JSON.stringify(this.messageQueue),
          );
        } catch (e) {
          console.warn("[GoSPA] Failed to persist message queue:", e);
        }
      }
    };
    window.addEventListener("beforeunload", this.beforeUnloadHandler);
  }

  get state(): ConnectionState {
    return this.connectionState.get();
  }

  get isConnected(): boolean {
    return this.connectionState.get() === "connected";
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.connectionState.set("connecting");

      // SECURITY: Do NOT pass session token in URL - it leaks in logs/referrers
      // Instead, send it as the first message after connection opens
      try {
        this.ws = new WebSocket(this.config.url);
        if (this.config.serializationFormat === "msgpack") {
          this.ws.binaryType = "arraybuffer";
        }
      } catch (error) {
        this.connectionState.set("disconnected");
        reject(error);
        return;
      }

      this.ws.onopen = () => {
        this.connectionState.set("connected");
        this.reconnectAttempts = 0;
        this.startHeartbeat();

        // SECURITY: Send session token as first message (not in URL)
        // Server will validate and associate this connection with the session
        if (this.sessionData?.clientId) {
          // Only send init if we have a clientId
          const initMsg: StateMessage = {
            type: "init",
            clientId: this.sessionData.clientId,
          };

          // Only send token if explicitly provided (legacy support)
          if (this.sessionData.token) {
            initMsg.sessionToken = this.sessionData.token;
          }

          this.send(initMsg);
        }

        this.flushMessageQueue();

        // State HMR: Request fresh state from server on reconnect
        // This softly patches the runes locally without refreshing the page!
        this.send({ type: "sync" });

        this.config.onOpen();
        resolve();
      };

      this.ws.onclose = (event) => {
        this.connectionState.set("disconnected");
        this.stopHeartbeat();
        this.config.onClose(event);

        if (
          this.config.reconnect &&
          this.reconnectAttempts < this.config.maxReconnectAttempts
        ) {
          this.scheduleReconnect();
        } else {
          this.config.onConnectionFailed(
            new Error("Max reconnect attempts reached"),
          );
        }
      };

      this.ws.onerror = (error) => {
        this.config.onError(error);
        if (this.connectionState.get() === "connecting") {
          reject(new Error("WebSocket connection failed"));
        }
      };

      this.ws.onmessage = (event) => {
        this.handleMessage(event.data);
      };
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.connectionState.set("disconnecting");
      this.stopHeartbeat();
      this.ws.close(1000, "Client disconnect");
      this.ws = null;
      this.connectionState.set("disconnected");
    }
    if (this.beforeUnloadHandler) {
      window.removeEventListener("beforeunload", this.beforeUnloadHandler);
      this.beforeUnloadHandler = null;
    }
  }

  private scheduleReconnect(): void {
    this.reconnectAttempts++;
    // Cap the delay at 5x the interval for fast initial retries,
    // even though total attempts may be much higher.
    const delay =
      this.config.reconnectInterval * Math.min(this.reconnectAttempts, 5);

    setTimeout(() => {
      if (this.connectionState.get() === "disconnected") {
        this.connect().catch(() => {});
      }
    }, delay);
  }

  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      this.send({ type: "ping" });
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
      if (this.config.serializationFormat === "msgpack") {
        this.ws.send(encode(message));
      } else {
        this.ws.send(JSON.stringify(message));
      }
    } else {
      this.messageQueue.push(message);
    }
  }

  sendWithResponse<T>(message: StateMessage): Promise<T> {
    return new Promise((resolve, reject) => {
      const id = `req_${++this.requestId}`;
      message.data = { ...message.data, _requestId: id };

      // Timeout after 30 seconds
      const timeout = setTimeout(() => {
        if (this.pendingRequests.has(id)) {
          this.pendingRequests.delete(id);
          reject(new Error("Request timeout"));
        }
      }, 30000);

      this.pendingRequests.set(id, {
        resolve: resolve as (value: unknown) => void,
        reject,
        timeout,
      });

      this.send(message);
    });
  }

  private async handleMessage(data: any): Promise<void> {
    try {
      let raw: any;
      if (
        this.config.serializationFormat === "msgpack" &&
        (data instanceof ArrayBuffer || data instanceof Uint8Array)
      ) {
        const buffer = data instanceof ArrayBuffer ? data : data.buffer;
        raw = decode(new Uint8Array(buffer));
      } else if (data instanceof Blob) {
        const buffer = await data.arrayBuffer();
        return this.handleMessage(buffer);
      } else {
        raw = typeof data === "string" ? JSON.parse(data) : data;
      }

      // SECURITY: Validate message structure before processing
      const message = validateMessage(raw);
      if (!message) {
        console.debug(
          "[GoSPA] Received invalid WebSocket message, ignoring:",
          raw,
        );
        return;
      }

      // Handle compressed messages
      if (message.type === "compressed" && typeof message.data === "string") {
        try {
          const compressedData = Uint8Array.from(atob(message.data), (c) =>
            c.charCodeAt(0),
          );
          const ds = new DecompressionStream("gzip");
          const writer = ds.writable.getWriter();
          writer.write(compressedData);
          writer.close();
          const response = new Response(ds.readable);
          const decompressed = await response.arrayBuffer();
          return this.handleMessage(decompressed);
        } catch (err) {
          console.error("[GoSPA] Failed to decompress message:", err);
          return;
        }
      }

      // Handle pong
      if (message.type === "pong") {
        return;
      }

      // Save session data when server sends it (init message with session token)
      if (message.type === "init" && message.sessionToken && message.clientId) {
        this.sessionData = {
          token: message.sessionToken,
          clientId: message.clientId,
        };
        if (this.config.persistSession) {
          saveSession(this.sessionData);
        }
      }

      // Handle response to pending request
      if (message.data?._responseId) {
        const id = message.data._responseId as string;
        const pending = this.pendingRequests.get(id);
        if (pending) {
          clearTimeout(pending.timeout);
          this.pendingRequests.delete(id);
          if (message.type === "error") {
            const rawError = message.error || "Unknown error";
            // Use native Error object which stores message as plain text.
            // The danger only exists if the UI developer does el.innerHTML = err.message.
            pending.reject(new Error(rawError));
          } else {
            pending.resolve(message.data);
          }
        }
      }

      this.config.onMessage(message);
    } catch (error) {
      console.error("[GoSPA] Failed to handle WebSocket message:", error);
    }
  }

  // Sync global state request
  requestSync(): void {
    this.send({ type: "sync" });
  }

  // Send custom action to server
  sendAction(action: string, payload: any = {}): void {
    this.send({
      type: "action",
      action,
      payload,
    });
  }

  // Request state from server
  requestState(componentId: string): Promise<Record<string, unknown>> {
    return this.sendWithResponse({
      type: "init",
      componentId,
    });
  }
}

// Global action helper
export function sendAction(action: string, payload: any = {}): void {
  if (clientInstance) {
    clientInstance.sendAction(action, payload);
  } else {
    console.warn("[GoSPA] Cannot send action: WebSocket not initialized");
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
  options: SyncedStateOptions,
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
            type: "update",
            payload: { key: options.key, value: newValue },
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
        console.warn("[GoSPA] Optimistic update failed, rolling back.", e);
        isReverting = true;
        originalSet(backupValue);
        isReverting = false;
      }
    } else {
      // Not connected, revert immediately
      console.warn("[GoSPA] WS disconnected, optimistic update rolled back.");
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
  ws?: WSClient,
): void {
  const client = ws || clientInstance;
  if (!client?.isConnected) return;

  for (const [key, rune] of Object.entries(states)) {
    client.send({
      type: "update",
      payload: { key, value: rune.get() },
    });
  }
}

// Apply server state updates
export function applyStateUpdate(
  states: Record<string, Rune<unknown>>,
  data: Record<string, unknown>,
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
