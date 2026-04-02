import { describe, expect, it, mock, beforeEach, afterEach } from "bun:test";
import {
  SSEClient,
  SSEManager,
  getSSEManager,
  createSSEClient,
} from "./sse";

// Mock EventSource globally
class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: any) => void) | null = null;
  onerror: ((event: any) => void) | null = null;

  // Custom mock event listeners
  listeners: Record<string, ((event: any) => void)[]> = {};

  static lastInstance: MockEventSource | null = null;

  constructor(url: string) {
    this.url = url;
    MockEventSource.lastInstance = this;
  }

  close() {
    this.onopen = null;
    this.onmessage = null;
    this.onerror = null;
    this.listeners = {};
  }

  addEventListener(type: string, listener: (event: any) => void) {
    if (!this.listeners[type]) {
      this.listeners[type] = [];
    }
    this.listeners[type].push(listener);
  }

  triggerOpen() {
    if (this.onopen) this.onopen();
  }

  triggerMessage(data: any, lastEventId?: string) {
    if (this.onmessage) {
      this.onmessage({
        data: typeof data === "string" ? data : JSON.stringify(data),
        lastEventId,
      });
    }
  }

  triggerError(err: Error) {
    if (this.onerror) this.onerror(err);
  }

  triggerCustom(type: string, data: any, lastEventId?: string) {
    if (this.listeners[type]) {
      this.listeners[type].forEach((l) =>
        l({
          data: typeof data === "string" ? data : JSON.stringify(data),
          lastEventId,
        }),
      );
    }
  }
}

describe("SSEClient", () => {
  let originalEventSource: any;
  let originalWindow: any;

  beforeEach(() => {
    originalEventSource = (globalThis as any).EventSource;
    (globalThis as any).EventSource = MockEventSource;

    originalWindow = (globalThis as any).window;
    (globalThis as any).window = {
      location: { origin: "http://localhost" }
    };

    MockEventSource.lastInstance = null;
  });

  afterEach(() => {
    (globalThis as any).EventSource = originalEventSource;
    if (!originalWindow) {
      delete (globalThis as any).window;
    } else {
      (globalThis as any).window = originalWindow;
    }
  });

  it("rejects authentication headers that would leak into the URL", () => {
    const client = new SSEClient({
      url: "/events",
      headers: { Authorization: "Bearer demo-token" },
    });

    expect(() => client.connect()).toThrow(
      "SSE authentication headers are not supported",
    );
  });

  it("should connect and trigger events", () => {
    const client = new SSEClient({ url: "/events" });
    const messageHandler = mock((evt) => evt);
    const customHandler = mock((evt) => evt);
    const wildCardHandler = mock((evt) => evt);

    client.onMessage(messageHandler);
    client.on("update", customHandler);
    client.on("*", wildCardHandler);

    client.connect();

    expect(client.getState()).toBe("connecting");
    if (!MockEventSource.lastInstance)
      throw new Error("MockEventSource.lastInstance is null");

    const inst = MockEventSource.lastInstance;

    // Test open
    inst.triggerOpen();
    expect(client.getState()).toBe("connected");

    // Test base message
    // SSEClient parses JSON wrapper if possible. Let's send a valid JSON
    inst.triggerMessage('{"foo":"bar"}', "event123");

    expect(client.getLastEventId()).toBe("event123");

    // Check call
    expect(messageHandler).toHaveBeenCalledTimes(1);
    expect(wildCardHandler).toHaveBeenCalledTimes(1);

    // Test custom event
    inst.triggerCustom("update", "{}", "ev0");
    expect(customHandler).toHaveBeenCalledTimes(1);
    expect(wildCardHandler).toHaveBeenCalledTimes(2);

    expect(client.isConnected()).toBeTrue();

    client.disconnect();
    expect(client.getState()).toBe("disconnected");
  });

  it("handles reconnection logic on error", () => {
    const client = new SSEClient({
      url: "/events",
      autoReconnect: true,
      maxRetries: 2,
      reconnectDelay: 50,
    });
    const errHandler = mock((_e, _attempt) => null);
    client.onError(errHandler);

    client.connect();
    expect(client.getState()).toBe("connecting");
    MockEventSource.lastInstance!.triggerError(new Error("broken"));

    expect(client.getState()).toBe("error");
    expect(errHandler).toHaveBeenCalledTimes(1);
  });
});

describe("SSEManager", () => {
  it("manages clients", () => {
    const manager = new SSEManager();
    expect(manager.has("test")).toBeFalse();

    manager.setDefaultConfig({ reconnectDelay: 100 });
    manager.client("test", { url: "/events" });
    expect(manager.has("test")).toBeTrue();
    expect(manager.getClientNames()).toContain("test");

    manager.disconnect("test");
    manager.remove("test");
    expect(manager.has("test")).toBeFalse();
  });
});

describe("Exports", () => {
  it("exports singletons", () => {
    const mgr1 = getSSEManager();
    const mgr2 = getSSEManager();
    expect(mgr1).toBe(mgr2);

    expect(createSSEClient({ url: "/demo" })).toBeInstanceOf(SSEClient);
  });
});
