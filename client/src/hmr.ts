/**
 * HMR (Hot Module Replacement) Client Runtime
 * Handles module updates with state preservation for GoSPA.
 * Includes: error rollback, dependency graph, exponential backoff, cross-tab sync, visual indicators
 */

// HMR Message Types
interface HMRMessage {
  type: "update" | "reload" | "error" | "state-preserve" | "connected";
  path?: string;
  moduleId?: string;
  event?: string;
  reloadReason?:
    | "template-safe"
    | "style-safe"
    | "runtime-break"
    | "config-break";
  state?: Record<string, unknown>;
  stateDiff?: Record<string, unknown>; // Delta state updates
  error?: string;
  timestamp: number;
}

// Module state registry
interface ModuleState {
  moduleId: string;
  state: Record<string, unknown>;
  timestamp: number;
}

// Module version for rollback
interface ModuleVersion {
  exports: Record<string, unknown>;
  timestamp: number;
}

// HMR update handler type
type HMRUpdateHandler = (msg: HMRMessage) => void | Promise<void>;

// HMR error handler type
type HMRErrorHandler = (error: string) => void;

// State preservation function type

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

const BLOCKED_MERGE_KEYS = new Set(["__proto__", "prototype", "constructor"]);

function isSafeMergeKey(key: string): boolean {
  return !BLOCKED_MERGE_KEYS.has(key);
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
  private moduleVersions: Map<string, ModuleVersion> = new Map(); // For rollback - only 1 version per moduleId (overwrites on update)
  private moduleDepGraph: Map<string, Set<string>> = new Map(); // Dependency graph
  private dependentsMap: Map<string, Set<string>> = new Map(); // Reverse deps
  private isConnecting = false;
  private updateQueue: HMRMessage[] = [];
  private isProcessing = false;
  private broadcastChannel: BroadcastChannel | null = null; // Cross-tab sync

  constructor(config: HMRClientConfig = {}) {
    this.config = {
      wsUrl:
        config.wsUrl ||
        `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/__hmr`,
      reconnectInterval: config.reconnectInterval || 1000,
      maxReconnectAttempts: config.maxReconnectAttempts || 10,
      onUpdate: config.onUpdate || (() => {}),
      onError: config.onError || ((err) => console.error("[HMR]", err)),
      onConnect: config.onConnect || (() => {}),
      onDisconnect: config.onDisconnect || (() => {}),
    };

    // Set up global handlers
    this.setupGlobalHandlers();

    // Set up cross-tab communication
    this.setupCrossTabSync();
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
    // Clean up state registry to prevent memory leaks
    this.stateRegistry.clear();
    this.moduleRegistry = {};
  }

  /**
   * Set up WebSocket event handlers
   */
  private setupWebSocketHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      this.isConnecting = false;
      this.reconnectAttempts = 0;
      console.log("[HMR] Connected");
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
      console.log("[HMR] Disconnected");
      this.config.onDisconnect();
      this.scheduleReconnect();
    };

    this.ws.onerror = (_error) => {
      this.isConnecting = false;
      this.config.onError("WebSocket error");
    };
  }

  /**
   * Schedule reconnection attempt with exponential backoff and jitter
   */
  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.config.onError("Max reconnection attempts reached");
      return;
    }

    this.reconnectAttempts++;

    // Exponential backoff with jitter (70-100% of calculated delay)
    const baseDelay = this.config.reconnectInterval;
    const maxDelay = 30000; // 30 seconds max
    const jitter = 0.7 + Math.random() * 0.3; // 70-100%
    // Apply maxDelay BEFORE multiplying by jitter to prevent overflow
    const exponentialDelay = Math.min(
      baseDelay * Math.pow(2, this.reconnectAttempts - 1),
      maxDelay,
    );
    const delay = exponentialDelay * jitter;

    console.log(
      `[HMR] Reconnecting in ${Math.round(delay)}ms (attempt ${this.reconnectAttempts})`,
    );

    setTimeout(() => {
      this.connect();
    }, delay);
  }

  /**
   * Handle incoming HMR message
   */
  private handleMessage(msg: HMRMessage): void {
    switch (msg.type) {
      case "connected":
        console.log("[HMR] Server connected");
        break;

      case "update":
        this.queueUpdate(msg);
        break;

      case "reload":
        console.log(
          "[HMR] Full reload required.",
          msg.reloadReason || "unknown",
        );
        this.preserveAllStates();
        window.location.reload();
        break;

      case "error":
        this.config.onError(msg.error || "Unknown error");
        break;

      case "state-preserve":
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
   * Public bridge used by legacy server-injected HMR scripts.
   */
  handleUpdate(msg: HMRMessage): void {
    this.queueUpdate(msg);
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
   * Apply an HMR update with error rollback capability
   */
  private async applyUpdate(msg: HMRMessage): Promise<void> {
    console.log(`[HMR] Applying update for: ${msg.moduleId}`);

    const moduleId = msg.moduleId;
    if (!moduleId) {
      console.warn("[HMR] Update message missing moduleId");
      return;
    }

    if (msg.reloadReason === "style-safe") {
      const href = msg.path || moduleId;
      CSSHMR.updateStyle(href);
      this.showUpdateNotification(`${moduleId} (styles)`);
      return;
    }

    if (msg.reloadReason === "template-safe") {
      const id = moduleId;
      TemplateHMR.updateTemplate(id, "");
      this.showUpdateNotification(`${moduleId} (template)`);
      return;
    }

    const currentModule = this.moduleRegistry[moduleId];

    // Save current state before update for potential rollback
    if (moduleId && currentModule) {
      try {
        const serialized = JSON.stringify(currentModule.exports);
        this.moduleVersions.set(moduleId, {
          exports: JSON.parse(serialized),
          timestamp: Date.now(),
        });
      } catch (e) {
        // Ignore serialization errors
      }
    }

    // Apply delta state if provided (for efficiency)
    if (msg.stateDiff && moduleId) {
      this.applyStateDiff(moduleId, msg.stateDiff);
    }

    // Call custom update handler
    try {
      if (this.config.onUpdate) {
        await this.config.onUpdate(msg);
      }

      // Default island HMR handling
      if (
        moduleId &&
        (moduleId.startsWith("islands/") || moduleId.includes(".gospa"))
      ) {
        await this.handleIslandUpdate(moduleId);
      }

      // Update version counter on success
      if (currentModule) {
        currentModule.version++;
      }

      // Show visual indicator
      this.showUpdateNotification(moduleId);

      // Broadcast update to other tabs (only the changed module)
      this.broadcastState(moduleId);

      // Get and update affected modules (dependency graph)
      const affectedModules = this.getAffectedModules(moduleId);
      for (const affectedId of affectedModules) {
        console.log(`[HMR] Also updating dependent: ${affectedId}`);
      }
    } catch (error) {
      // Rollback on failure
      console.error(`[HMR] Update failed, rolling back: ${error}`);
      this.rollbackModule(moduleId);
      this.config.onError(`Update failed and rolled back: ${error}`);
    }

    // Restore state after update
    if (moduleId && msg.state) {
      this.restoreState(moduleId, msg.state);
    }
  }

  /**
   * Apply delta state changes efficiently
   */
  private applyStateDiff(
    moduleId: string,
    stateDiff: Record<string, unknown>,
  ): void {
    const module = this.moduleRegistry[moduleId];
    if (!module?.exports) return;

    for (const [key, value] of Object.entries(stateDiff)) {
      if (!isSafeMergeKey(key)) continue;
      if (key in module.exports && typeof module.exports[key] === "object") {
        const currentValue = module.exports[key] as Record<string, unknown>;
        if (value && typeof value === "object" && !Array.isArray(value)) {
          for (const [nestedKey, nestedValue] of Object.entries(value)) {
            if (!isSafeMergeKey(nestedKey)) continue;
            currentValue[nestedKey] = nestedValue;
          }
        }
      } else {
        module.exports[key] = value;
      }
    }
  }

  /**
   * Handle hot update for an island component
   */
  private async handleIslandUpdate(moduleId: string): Promise<void> {
    const islandName = moduleId.split("/").pop()?.replace(".gospa", "") || "";
    if (!islandName) return;

    console.log(`[HMR] Re-hydrating island instances: ${islandName}`);

    // Access global island manager
    const manager = (window as any).__GOSPA_ISLAND_MANAGER__;
    if (manager && manager.get()) {
      const gManager = manager.get();
      const islands = gManager
        .getIslands()
        .filter((i: any) => i.name === islandName);

      for (const island of islands) {
        // Force re-hydration by clearing hydrated status
        gManager.hydrated.delete(island.id);
        gManager.pending.delete(island.id);
        await gManager.hydrateIsland(island);
      }
    }
  }

  /**
   * Rollback a module to its previous version
   */
  private rollbackModule(moduleId: string): void {
    const savedVersion = this.moduleVersions.get(moduleId);
    const currentModule = this.moduleRegistry[moduleId];

    if (savedVersion && currentModule) {
      currentModule.exports = savedVersion.exports;
      console.log(`[HMR] Rolled back module: ${moduleId}`);
    }
  }

  /**
   * Get all modules that depend on a changed module (using dependency graph)
   */
  getAffectedModules(moduleId: string): string[] {
    const affected = new Set<string>();
    const queue = [moduleId];

    while (queue.length > 0) {
      const current = queue.shift()!;
      const dependents = this.dependentsMap.get(current);
      if (dependents) {
        for (const dep of dependents) {
          if (!affected.has(dep)) {
            affected.add(dep);
            queue.push(dep);
          }
        }
      }
    }

    return Array.from(affected);
  }

  /**
   * Set up global handlers for state preservation
   */
  private setupGlobalHandlers(): void {
    // Expose HMR API globally
    (window as unknown as { __gospaHMR: HMRClient }).__gospaHMR = this;

    // State preservation before unload
    window.addEventListener("beforeunload", () => {
      this.preserveAllStates();
    });

    // Handle visibility change for mobile
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "hidden") {
        this.preserveAllStates();
      }
    });
  }

  /**
   * Set up cross-tab communication using BroadcastChannel
   */
  private setupCrossTabSync(): void {
    if (typeof BroadcastChannel === "undefined") return;

    try {
      this.broadcastChannel = new BroadcastChannel("gospa-hmr");
      this.broadcastChannel.onmessage = (event) => {
        if (event.data.type === "state-sync") {
          this.syncStateFromTab(event.data.state);
        }
      };
    } catch (e) {
      console.warn("[HMR] Cross-tab sync not available");
    }
  }

  /**
   * Sync state from another tab
   */
  private syncStateFromTab(state: Record<string, unknown>): void {
    for (const [moduleId, moduleState] of Object.entries(state)) {
      this.restoreState(moduleId, moduleState as Record<string, unknown>);
    }
    console.log("[HMR] State synced from another tab");
  }

  /**
   * Broadcast state changes to other tabs (only the changed module)
   */
  private broadcastState(moduleId?: string): void {
    if (!this.broadcastChannel) return;

    // If moduleId provided, only broadcast that module's state (delta)
    // Otherwise broadcast all (fallback for initial sync)
    const states: Record<string, unknown> = {};
    if (moduleId) {
      const state = this.extractModuleState(moduleId);
      if (state) states[moduleId] = state;
    } else {
      for (const id of Object.keys(this.moduleRegistry)) {
        const state = this.extractModuleState(id);
        if (state) states[id] = state;
      }
    }

    this.broadcastChannel.postMessage({
      type: "state-sync",
      state: states,
    });
  }

  /**
   * Show visual notification when module is updated
   */
  private showUpdateNotification(moduleId: string): void {
    if (typeof document === "undefined") return;

    // Remove existing toast if any
    const existing = document.querySelector(".hmr-toast");
    if (existing) existing.remove();

    const toast = document.createElement("div");
    toast.className = "hmr-toast";
    toast.textContent = `Updated: ${moduleId}`;
    toast.setAttribute(
      "style",
      `
			position: fixed;
			bottom: 20px;
			right: 20px;
			background: #10b981;
			color: white;
			padding: 8px 16px;
			border-radius: 4px;
			font-family: system-ui, -apple-system, sans-serif;
			font-size: 12px;
			z-index: 99999;
			opacity: 0;
			transition: opacity 0.3s ease;
			box-shadow: 0 2px 8px rgba(0,0,0,0.15);
		`,
    );

    document.body.appendChild(toast);

    // Fade in
    requestAnimationFrame(() => (toast.style.opacity = "1"));

    // Fade out after 2 seconds
    setTimeout(() => {
      toast.style.opacity = "0";
      setTimeout(() => toast.remove(), 300);
    }, 2000);
  }

  /**
   * Register a module for HMR with dependency tracking
   */
  registerModule(
    moduleId: string,
    exports: Record<string, unknown>,
    deps?: string[],
  ): void {
    this.moduleRegistry[moduleId] = {
      version: 0,
      exports,
      accept: true,
      deps,
    };

    // Build dependency graph
    if (deps) {
      this.moduleDepGraph.set(moduleId, new Set(deps));
      for (const dep of deps) {
        if (!this.dependentsMap.has(dep)) {
          this.dependentsMap.set(dep, new Set());
        }
        this.dependentsMap.get(dep)!.add(moduleId);
      }
    }
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
    const stateGetter = (
      window as unknown as {
        __gospaGetState?: (id: string) => Record<string, unknown> | null;
      }
    ).__gospaGetState;
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
    if (
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "boolean"
    )
      return true;
    if (value instanceof Date) return true;
    if (Array.isArray(value)) return value.every((v) => this.isSerializable(v));
    if (typeof value === "object") {
      return Object.values(value as Record<string, unknown>).every((v) =>
        this.isSerializable(v),
      );
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
        if (!isSafeMergeKey(key)) continue;
        if (key in module.exports && typeof module.exports[key] === "object") {
          const currentValue = module.exports[key] as Record<string, unknown>;
          if (value && typeof value === "object" && !Array.isArray(value)) {
            for (const [nestedKey, nestedValue] of Object.entries(value)) {
              if (!isSafeMergeKey(nestedKey)) continue;
              currentValue[nestedKey] = nestedValue;
            }
          }
        } else {
          module.exports[key] = value;
        }
      }
    }

    // Notify state restoration
    const stateSetter = (
      window as unknown as {
        __gospaSetState?: (id: string, state: Record<string, unknown>) => void;
      }
    ).__gospaSetState;
    if (stateSetter) {
      stateSetter(moduleId, state);
    }
  }

  /**
   * Send state to server
   */
  private sendState(moduleId: string, state: Record<string, unknown>): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(
        JSON.stringify({
          type: "state-preserve",
          moduleId,
          state,
        }),
      );
    }
  }

  /**
   * Request state from server
   */
  requestState(moduleId: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(
        JSON.stringify({
          type: "state-request",
          moduleId,
        }),
      );
    }
  }

  /**
   * Report error to server
   */
  reportError(error: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(
        JSON.stringify({
          type: "error",
          error,
        }),
      );
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
      url.searchParams.set("t", Date.now().toString());
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
    window.dispatchEvent(
      new CustomEvent("gospa:template-update", {
        detail: { id, content },
      }),
    );
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
export function registerHMRModule(
  moduleId: string,
  exports: Record<string, unknown>,
  deps?: string[],
): void {
  globalHMRClient?.registerModule(moduleId, exports, deps);
}

/**
 * Accept HMR updates
 */
export function acceptHMR(moduleId: string): void {
  globalHMRClient?.accept(moduleId);
}

// Auto-initialize if in browser
if (typeof window !== "undefined") {
  // Wait for DOM ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => initHMR());
  } else {
    initHMR();
  }
}

export default HMRClient;
