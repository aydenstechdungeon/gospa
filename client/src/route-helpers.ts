import {
  navigate,
  prefetch,
  prefetchOnHover as bindPrefetchOnHover,
  onBeforeNavigate,
  onAfterNavigate,
  invalidateAll,
  type NavigateOptions,
  type HoverPrefetchOptions,
} from "./navigation.ts";
import type { ActionEnhanceSuccess } from "./forms.ts";

export type { NavigateOptions };

export interface CallRouteActionOptions extends RequestInit {
  throwOnError?: boolean;
}

export class RouteActionError<R = unknown> extends Error {
  readonly status: number;
  readonly payload: ActionEnhanceSuccess<R>;

  constructor(
    status: number,
    payload: ActionEnhanceSuccess<R>,
    message = `Route action failed (${status})`,
  ) {
    super(message);
    this.name = "RouteActionError";
    this.status = status;
    this.payload = payload;
  }
}

/**
 * Fetch route load data using the GoSPA data endpoint contract.
 */
export async function loadRouteData<T = Record<string, unknown>>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const dataURL = new URL(path, window.location.origin);
  dataURL.searchParams.set("__data", "1");
  const res = await fetch(dataURL.toString(), {
    ...init,
    credentials: init?.credentials ?? "same-origin",
    headers: {
      Accept: "application/json",
      ...(init?.headers || {}),
    },
  });
  if (!res.ok) {
    throw new Error(`Failed to load route data (${res.status})`);
  }
  const payload = (await res.json()) as { data?: T };
  return (payload.data ?? ({} as T)) as T;
}

/**
 * Route action helper with progressive-enhancement semantics.
 */
export async function callRouteAction<R = unknown>(
  path: string,
  action: string,
  body?: BodyInit | null,
  init?: CallRouteActionOptions,
): Promise<ActionEnhanceSuccess<R>> {
  const actionURL = new URL(path, window.location.origin);
  actionURL.searchParams.set("_action", action);
  const throwOnError = init?.throwOnError !== false;
  const requestInit = { ...(init || {}) };
  delete (requestInit as CallRouteActionOptions).throwOnError;
  const res = await fetch(actionURL.toString(), {
    method: requestInit.method ?? "POST",
    credentials: requestInit.credentials ?? "same-origin",
    ...requestInit,
    headers: {
      Accept: "application/json",
      "X-Gospa-Enhance": "1",
      ...(requestInit.headers || {}),
    },
    body: body ?? requestInit.body ?? null,
  });
  const payload = await res
    .json()
    .catch(() => ({ error: `Action failed with HTTP ${res.status}` }));
  if (!res.ok && throwOnError) {
    throw new RouteActionError<R>(
      res.status,
      payload as ActionEnhanceSuccess<R>,
      (payload as { error?: string })?.error ||
        `Route action failed (${res.status})`,
    );
  }
  return payload as ActionEnhanceSuccess<R>;
}

/**
 * Compatibility helper: preload route data.
 */
export async function preloadData<T = Record<string, unknown>>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  return loadRouteData<T>(path, init);
}

/**
 * Compatibility helper: preload route code/navigation payloads.
 */
export async function preloadCode(path: string): Promise<void> {
  await prefetch(path);
}

/**
 * Compatibility helper: programmatic navigation.
 */
export async function goto(
  to: string,
  options?: NavigateOptions,
): Promise<boolean> {
  return navigate(to, options);
}

/**
 * Refresh current route data using the GoSPA data endpoint contract.
 */
export async function refresh(init?: RequestInit): Promise<void> {
  const current =
    window.location.pathname + window.location.search + window.location.hash;
  await loadRouteData(current, init);
}

/**
 * Declaratively prefetch matching links on hover.
 */
export function prefetchOnHover(
  selector: string,
  options?: HoverPrefetchOptions,
): () => void {
  return bindPrefetchOnHover(selector, options);
}

export { onBeforeNavigate as beforeNavigate, onAfterNavigate as afterNavigate };
export { invalidateAll };
