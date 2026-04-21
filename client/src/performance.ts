// GoSPA Performance Monitoring
// Tracks runtime performance metrics for optimization

/**
 * Performance metric entry
 */
interface PerformanceMetric {
  name: string;
  duration: number;
  timestamp: number;
  metadata?: Record<string, unknown>;
}

/**
 * Performance monitoring configuration
 */
export interface PerformanceConfig {
  /** Enable performance monitoring (default: true in dev, false in prod) */
  enabled?: boolean;
  /** Maximum number of metrics to keep in memory (default: 1000) */
  maxMetrics?: number;
  /** Sample rate for metrics (0-1, default: 1) */
  sampleRate?: number;
  /** Enable console logging of metrics (default: false) */
  enableConsoleLog?: boolean;
}

/**
 * Performance monitor
 */
export class PerformanceMonitor {
  private metrics: PerformanceMetric[] = [];
  private marks: Map<string, { startTime: number; sampled: boolean }> =
    new Map();
  private config: Required<PerformanceConfig>;
  private observers: Set<(metric: PerformanceMetric) => void> = new Set();

  constructor(config: PerformanceConfig = {}) {
    this.config = {
      enabled:
        config.enabled ??
        (typeof process !== "undefined" &&
          process.env?.NODE_ENV !== "production"),
      maxMetrics: config.maxMetrics ?? 1000,
      sampleRate: config.sampleRate ?? 1,
      enableConsoleLog: config.enableConsoleLog ?? false,
    };
  }

  /**
   * Check if monitoring is enabled
   */
  private isEnabled(): boolean {
    return this.config.enabled;
  }

  /**
   * Start a performance measurement
   */
  start(name: string): void {
    if (!this.isEnabled()) return;

    const sampled =
      this.config.sampleRate >= 1 || Math.random() <= this.config.sampleRate;
    this.marks.set(name, { startTime: performance.now(), sampled });

    if (!sampled) {
      return;
    }

    const markName = `gospa:${name}:start`;

    if (typeof performance !== "undefined" && performance.mark) {
      performance.mark(markName);
    }
  }

  /**
   * End a performance measurement
   */
  end(name: string, metadata?: Record<string, unknown>): number | null {
    if (!this.isEnabled()) return null;

    const mark = this.marks.get(name);
    if (mark === undefined) {
      console.warn(`[GoSPA Performance] No start mark found for: ${name}`);
      return null;
    }

    this.marks.delete(name);

    if (!mark.sampled) {
      return null;
    }

    const endTime = performance.now();
    const duration = endTime - mark.startTime;

    // Record metric
    const metric: PerformanceMetric = {
      name,
      duration,
      timestamp: Date.now(),
      metadata,
    };

    this.addMetric(metric);

    // Use Performance API if available
    if (typeof performance !== "undefined" && performance.measure) {
      try {
        const startMark = `gospa:${name}:start`;
        const endMark = `gospa:${name}:end`;

        performance.mark(endMark);
        performance.measure(`gospa:${name}`, startMark, endMark);

        // Clean up marks
        performance.clearMarks(startMark);
        performance.clearMarks(endMark);
      } catch {
        // Ignore errors from Performance API
      }
    }

    return duration;
  }

  /**
   * Measure a function's execution time
   */
  measure<T>(name: string, fn: () => T, metadata?: Record<string, unknown>): T {
    if (!this.config.enabled) {
      return fn();
    }

    this.start(name);
    try {
      const result = fn();
      this.end(name, metadata);
      return result;
    } catch (error) {
      this.end(name, { ...metadata, error: true });
      throw error;
    }
  }

  /**
   * Measure an async function's execution time
   */
  async measureAsync<T>(
    name: string,
    fn: () => Promise<T>,
    metadata?: Record<string, unknown>,
  ): Promise<T> {
    if (!this.config.enabled) {
      return fn();
    }

    this.start(name);
    try {
      const result = await fn();
      this.end(name, metadata);
      return result;
    } catch (error) {
      this.end(name, { ...metadata, error: true });
      throw error;
    }
  }

  /**
   * Add a metric to the store
   */
  private addMetric(metric: PerformanceMetric): void {
    this.metrics.push(metric);

    // Trim if over max
    if (this.metrics.length > this.config.maxMetrics) {
      this.metrics = this.metrics.slice(-this.config.maxMetrics);
    }

    // Notify observers
    for (const observer of this.observers) {
      try {
        observer(metric);
      } catch (error) {
        console.error("[GoSPA Performance] Observer error:", error);
      }
    }

    // Console log if enabled
    if (this.config.enableConsoleLog) {
      console.log(
        `[GoSPA Performance] ${metric.name}: ${metric.duration.toFixed(2)}ms`,
        metric.metadata,
      );
    }
  }

  /**
   * Get all metrics
   */
  getMetrics(): PerformanceMetric[] {
    return [...this.metrics];
  }

  /**
   * Get metrics by name
   */
  getMetricsByName(name: string): PerformanceMetric[] {
    return this.metrics.filter((m) => m.name === name);
  }

  /**
   * Get average duration for a metric
   */
  getAverageDuration(name: string): number {
    const metrics = this.getMetricsByName(name);
    if (metrics.length === 0) return 0;

    const total = metrics.reduce((sum, m) => sum + m.duration, 0);
    return total / metrics.length;
  }

  /**
   * Get performance summary
   */
  getSummary(): Record<
    string,
    { count: number; avg: number; min: number; max: number }
  > {
    const summary: Record<
      string,
      { count: number; avg: number; min: number; max: number }
    > = {};

    for (const metric of this.metrics) {
      if (!summary[metric.name]) {
        summary[metric.name] = {
          count: 0,
          avg: 0,
          min: Infinity,
          max: -Infinity,
        };
      }

      const s = summary[metric.name];
      s.count++;
      s.min = Math.min(s.min, metric.duration);
      s.max = Math.max(s.max, metric.duration);
    }

    // Calculate averages
    for (const name of Object.keys(summary)) {
      const metrics = this.getMetricsByName(name);
      const total = metrics.reduce((sum, m) => sum + m.duration, 0);
      summary[name].avg = total / metrics.length;
    }

    return summary;
  }

  /**
   * Subscribe to metrics
   */
  subscribe(observer: (metric: PerformanceMetric) => void): () => void {
    this.observers.add(observer);
    return () => this.observers.delete(observer);
  }

  /**
   * Clear all metrics
   */
  clear(): void {
    this.metrics = [];
    this.marks.clear();
  }

  /**
   * Get memory usage (if available)
   */
  getMemoryUsage(): { used: number; total: number } | null {
    if (typeof performance !== "undefined" && "memory" in performance) {
      const memory = (performance as any).memory;
      return {
        used: memory.usedJSHeapSize,
        total: memory.totalJSHeapSize,
      };
    }
    return null;
  }

  /**
   * Get Web Vitals (if available)
   */
  async getWebVitals(): Promise<Record<string, number>> {
    const vitals: Record<string, number> = {};

    // First Contentful Paint
    if (typeof performance !== "undefined" && performance.getEntriesByType) {
      const paintEntries = performance.getEntriesByType("paint");
      for (const entry of paintEntries) {
        if (entry.name === "first-contentful-paint") {
          vitals["FCP"] = entry.startTime;
        }
      }

      // Largest Contentful Paint
      const lcpEntries = performance.getEntriesByType(
        "largest-contentful-paint",
      );
      if (lcpEntries.length > 0) {
        vitals["LCP"] = lcpEntries[lcpEntries.length - 1].startTime;
      }

      // First Input Delay
      const fidEntries = performance.getEntriesByType("first-input");
      if (fidEntries.length > 0) {
        const fid = fidEntries[0] as any;
        vitals["FID"] = fid.processingStart - fid.startTime;
      }

      // Cumulative Layout Shift
      const clsEntries = performance.getEntriesByType("layout-shift");
      let clsValue = 0;
      for (const entry of clsEntries) {
        if (!(entry as any).hadRecentInput) {
          clsValue += (entry as any).value;
        }
      }
      vitals["CLS"] = clsValue;
    }

    return vitals;
  }
}

/**
 * Create a performance monitor
 */
export function createPerformanceMonitor(
  config?: PerformanceConfig,
): PerformanceMonitor {
  return new PerformanceMonitor(config);
}

/**
 * Global performance monitor instance
 */
let globalMonitor: PerformanceMonitor | null = null;

/**
 * Get or create the global performance monitor
 */
export function getPerformanceMonitor(
  config?: PerformanceConfig,
): PerformanceMonitor {
  if (!globalMonitor) {
    globalMonitor = new PerformanceMonitor(config);
  }
  return globalMonitor;
}

/**
 * Destroy the global performance monitor
 */
export function destroyPerformanceMonitor(): void {
  if (globalMonitor) {
    globalMonitor.clear();
    globalMonitor = null;
  }
}

/**
 * Quick measure helper
 */
export function measure<T>(
  name: string,
  fn: () => T,
  metadata?: Record<string, unknown>,
): T {
  return getPerformanceMonitor().measure(name, fn, metadata);
}

/**
 * Quick async measure helper
 */
export function measureAsync<T>(
  name: string,
  fn: () => Promise<T>,
  metadata?: Record<string, unknown>,
): Promise<T> {
  return getPerformanceMonitor().measureAsync(name, fn, metadata);
}
