import {
  initWebSocket,
  type StateMessage,
  type WSClient,
} from "./websocket.ts";
import { createSSEClient, type SSEClient } from "./sse.ts";

export type TransportMode = "ws" | "sse" | "polling" | "none";

export interface TransportConfig {
  wsUrl?: string;
  sseUrl?: string;
  pollUrl?: string;
  pollInterval?: number;
  debug?: boolean;
  onMessage?: (message: StateMessage | Record<string, unknown>) => void;
  onModeChange?: (mode: TransportMode) => void;
  wsReconnectDelay?: number;
  wsMaxReconnect?: number;
  wsHeartbeat?: number;
  serializationFormat?: "json" | "msgpack";
}

export class TransportManager {
  private readonly config: Required<TransportConfig>;
  private ws: WSClient | null = null;
  private sse: SSEClient | null = null;
  private pollTimer: ReturnType<typeof setInterval> | null = null;
  private mode: TransportMode = "none";
  private stopped = false;

  constructor(config: TransportConfig) {
    this.config = {
      wsUrl: config.wsUrl ?? "",
      sseUrl: config.sseUrl ?? "/_sse/connect",
      pollUrl: config.pollUrl ?? "/_gospa/poll",
      pollInterval: config.pollInterval ?? 5000,
      debug: config.debug ?? false,
      onMessage: config.onMessage ?? (() => {}),
      onModeChange: config.onModeChange ?? (() => {}),
      wsReconnectDelay: config.wsReconnectDelay ?? 1000,
      wsMaxReconnect: config.wsMaxReconnect ?? 10,
      wsHeartbeat: config.wsHeartbeat ?? 30000,
      serializationFormat: config.serializationFormat ?? "json",
    };
  }

  getMode(): TransportMode {
    return this.mode;
  }

  async start(): Promise<TransportMode> {
    this.stopped = false;
    if (this.config.wsUrl) {
      const ok = await this.startWebSocket();
      if (ok) return this.mode;
    }

    const sseOk = this.startSSE();
    if (sseOk) return this.mode;

    this.startPolling();
    return this.mode;
  }

  stop(): void {
    this.stopped = true;
    if (this.ws) {
      this.ws.disconnect();
      this.ws = null;
    }
    if (this.sse) {
      this.sse.disconnect();
      this.sse = null;
    }
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
    this.setMode("none");
  }

  private async startWebSocket(): Promise<boolean> {
    try {
      const ws = initWebSocket({
        url: this.config.wsUrl,
        reconnect: true,
        reconnectInterval: this.config.wsReconnectDelay,
        maxReconnectAttempts: this.config.wsMaxReconnect,
        heartbeatInterval: this.config.wsHeartbeat,
        serializationFormat: this.config.serializationFormat,
        onConnectionFailed: () => {
          if (this.stopped) return;
          this.log(
            "WebSocket exhausted reconnects; switching transport fallback",
          );
          this.startSSE() || this.startPolling();
        },
        onMessage: (msg: StateMessage) => {
          this.config.onMessage(msg);
        },
      });

      await ws.connect();
      this.ws = ws;
      this.setMode("ws");
      return true;
    } catch (err) {
      this.log("WebSocket connection failed", err);
      return false;
    }
  }

  private startSSE(): boolean {
    try {
      const sse = createSSEClient({
        url: this.config.sseUrl,
        autoReconnect: true,
        debug: this.config.debug,
      });

      sse.onMessage((event) => {
        const payload =
          event && typeof event.data === "object" && event.data !== null
            ? (event.data as Record<string, unknown>)
            : { data: event.data };
        this.config.onMessage(payload);
      });

      sse.onError(() => {
        if (this.stopped) return;
        if (this.mode === "sse") {
          this.log("SSE degraded; switching to polling");
          this.startPolling();
        }
      });

      sse.connect();
      this.sse = sse;
      this.setMode("sse");
      return true;
    } catch (err) {
      this.log("SSE connection failed", err);
      return false;
    }
  }

  private startPolling(): void {
    if (this.pollTimer) return;
    this.setMode("polling");
    this.pollTimer = setInterval(async () => {
      try {
        const res = await fetch(this.config.pollUrl, {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
        });
        if (!res.ok) return;
        const payload = await res.json();
        if (Array.isArray((payload as any)?.messages)) {
          for (const msg of (payload as any).messages) {
            this.config.onMessage(msg);
          }
          return;
        }
        this.config.onMessage(payload as Record<string, unknown>);
      } catch (err) {
        this.log("Polling request failed", err);
      }
    }, this.config.pollInterval);
  }

  private setMode(mode: TransportMode): void {
    if (this.mode === mode) return;
    this.mode = mode;
    try {
      (window as any).__GOSPA_TRANSPORT_MODE__ = mode;
      window.dispatchEvent(
        new CustomEvent("gospa:transport-mode", {
          detail: { mode },
        }),
      );
    } catch {
      // Ignore browsers/environments without CustomEvent.
    }
    this.config.onModeChange(mode);
  }

  private log(message: string, ...rest: unknown[]): void {
    if (!this.config.debug) return;
    console.log("[GoSPA transport]", message, ...rest);
  }
}

let transportInstance: TransportManager | null = null;

export function initTransport(config: TransportConfig): TransportManager {
  if (transportInstance) {
    transportInstance.stop();
  }
  transportInstance = new TransportManager(config);
  void transportInstance.start();
  return transportInstance;
}

export function getTransportManager(): TransportManager | null {
  return transportInstance;
}
