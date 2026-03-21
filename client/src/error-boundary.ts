// GoSPA Error Boundary System
// Provides component-level error handling with fallback rendering

import { Rune } from "./state.ts";

/**
 * Error boundary configuration
 */
export interface ErrorBoundaryConfig {
  /** Fallback element or render function when an error occurs */
  fallback: ((error: Error, componentId: string) => Element) | Element;
  /** Optional error callback for logging/reporting */
  onError?: (error: Error, componentId: string) => void;
  /** Whether to retry the component after error recovery */
  retryable?: boolean;
  /** Maximum retry attempts */
  maxRetries?: number;
}

/**
 * Error boundary state
 */
interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  retryCount: number;
}

/**
 * Global error boundary registry
 */
const errorBoundaries = new Map<string, ErrorBoundaryState>();

/**
 * Global error handlers
 */
const globalErrorHandlers = new Set<
  (error: Error, componentId: string) => void
>();

/**
 * Register a global error handler
 */
export function onComponentError(
  handler: (error: Error, componentId: string) => void,
): () => void {
  globalErrorHandlers.add(handler);
  return () => globalErrorHandlers.delete(handler);
}

/**
 * Wrap a component with error boundary protection.
 * Catches errors during mount, update, and destroy phases.
 *
 * @example
 * ```typescript
 * const safeComponent = withErrorBoundary('my-component', {
 *   fallback: (error) => {
 *     const el = document.createElement('div');
 *     el.className = 'error-fallback';
 *     el.textContent = `Error: ${error.message}`;
 *     return el;
 *   },
 *   onError: (error, id) => console.error(`Component ${id} failed:`, error)
 * });
 * ```
 */
export function withErrorBoundary(
  componentId: string,
  config: ErrorBoundaryConfig,
): {
  wrapMount: (mountFn: () => void | (() => void)) => () => void | (() => void);
  wrapDestroy: (destroyFn: () => void) => () => void;
  wrapAction: <T>(
    actionFn: (...args: unknown[]) => T,
  ) => (...args: unknown[]) => T;
  clearError: () => void;
  getState: () => ErrorBoundaryState;
} {
  // Initialize boundary state
  if (!errorBoundaries.has(componentId)) {
    errorBoundaries.set(componentId, {
      hasError: false,
      error: null,
      retryCount: 0,
    });
  }

  const getState = () => errorBoundaries.get(componentId)!;

  const handleError = (error: Error): void => {
    const state = getState();
    state.hasError = true;
    state.error = error;

    // Call config error handler
    config.onError?.(error, componentId);

    // Notify global handlers
    for (const handler of globalErrorHandlers) {
      try {
        handler(error, componentId);
      } catch (handlerError) {
        console.error("[GoSPA] Error in error handler:", handlerError);
      }
    }

    // Render fallback
    const element = document.querySelector(
      `[data-gospa-component="${componentId}"]`,
    );
    if (element) {
      const fallbackEl =
        typeof config.fallback === "function"
          ? config.fallback(error, componentId)
          : (config.fallback.cloneNode(true) as Element);

      // Clear existing content
      element.innerHTML = "";
      element.appendChild(fallbackEl);

      // Add retry button if retryable
      if (config.retryable && state.retryCount < (config.maxRetries ?? 3)) {
        const retryBtn = document.createElement("button");
        retryBtn.textContent = "Retry";
        retryBtn.className = "gospa-retry-btn";
        retryBtn.onclick = () => {
          state.retryCount++;
          state.hasError = false;
          state.error = null;
          // Trigger re-mount by dispatching custom event
          element.dispatchEvent(
            new CustomEvent("gospa:retry", { detail: { componentId } }),
          );
        };
        element.appendChild(retryBtn);
      }
    }
  };

  const wrapMount = (
    mountFn: () => void | (() => void),
  ): (() => void | (() => void)) => {
    return () => {
      const state = getState();
      if (state.hasError) {
        // Skip mount if already in error state
        return () => {};
      }

      try {
        return mountFn();
      } catch (error) {
        handleError(error as Error);
        return () => {};
      }
    };
  };

  const wrapDestroy = (destroyFn: () => void): (() => void) => {
    return () => {
      try {
        destroyFn();
      } catch (error) {
        // Log but don't propagate destroy errors
        console.error(
          `[GoSPA] Error destroying component ${componentId}:`,
          error,
        );
      }
    };
  };

  const wrapAction = <T>(
    actionFn: (...args: unknown[]) => T,
  ): ((...args: unknown[]) => T) => {
    return (...args: unknown[]): T => {
      const state = getState();
      if (state.hasError) {
        throw new Error(
          `Component ${componentId} is in error state: ${state.error?.message}`,
        );
      }

      try {
        return actionFn(...args);
      } catch (error) {
        handleError(error as Error);
        throw error;
      }
    };
  };

  const clearError = (): void => {
    const state = getState();
    state.hasError = false;
    state.error = null;
    state.retryCount = 0;
  };

  return {
    wrapMount,
    wrapDestroy,
    wrapAction,
    clearError,
    getState,
  };
}

/**
 * Create a simple error fallback element
 */
export function createErrorFallback(message?: string): Element {
  const el = document.createElement("div");
  el.className = "gospa-error-fallback";
  el.setAttribute("role", "alert");
  el.innerHTML = `
		<div class="gospa-error-content">
			<svg class="gospa-error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="12" cy="12" r="10"/>
				<line x1="12" y1="8" x2="12" y2="12"/>
				<line x1="12" y1="16" x2="12.01" y2="16"/>
			</svg>
			<p class="gospa-error-message">${message || "Something went wrong"}</p>
		</div>
	`;
  return el;
}

/**
 * Get error boundary state for a component
 */
export function getErrorBoundaryState(
  componentId: string,
): ErrorBoundaryState | undefined {
  return errorBoundaries.get(componentId);
}

/**
 * Clear all error boundaries
 */
export function clearAllErrorBoundaries(): void {
  for (const state of errorBoundaries.values()) {
    state.hasError = false;
    state.error = null;
    state.retryCount = 0;
  }
}

/**
 * Check if a component is in error state
 */
export function isInErrorState(componentId: string): boolean {
  return errorBoundaries.get(componentId)?.hasError ?? false;
}
