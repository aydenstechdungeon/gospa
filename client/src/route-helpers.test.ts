import { describe, it, expect, beforeAll, beforeEach, mock } from "bun:test";
import { GlobalWindow } from "happy-dom";

const window = new GlobalWindow();
window.happyDOM.setURL("https://app.test/");
(window as any).__GOSPA_CONFIG__ = {
  navigationOptions: { speculativePrefetching: { enabled: false } },
};
(globalThis as any).window = window;
(globalThis as any).document = window.document;
(globalThis as any).Element = window.Element;
(globalThis as any).HTMLElement = window.HTMLElement;
(globalThis as any).HTMLAnchorElement = window.HTMLAnchorElement;
(globalThis as any).Event = window.Event;
(globalThis as any).URL = window.URL;
(globalThis as any).Response = window.Response;
(globalThis as any).IntersectionObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

let loadRouteData: typeof import("./route-helpers").loadRouteData;
let callRouteAction: typeof import("./route-helpers").callRouteAction;
let invalidateAll: typeof import("./route-helpers").invalidateAll;
let refresh: typeof import("./route-helpers").refresh;
let prefetchOnHover: typeof import("./route-helpers").prefetchOnHover;

describe("route-helpers", () => {
  beforeAll(async () => {
    ({
      loadRouteData,
      callRouteAction,
      invalidateAll,
      refresh,
      prefetchOnHover,
    } = await import("./route-helpers"));
  });

  beforeEach(() => {
    document.body.innerHTML = "";
  });

  it("loadRouteData fetches __data endpoint with same-origin defaults", async () => {
    const fetchMock = mock(
      async (_url: string, _init?: RequestInit) =>
        new Response(JSON.stringify({ data: { title: "Dashboard" } }), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
    );
    (globalThis as any).fetch = fetchMock;

    const data = await loadRouteData<{ title: string }>("/dashboard");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const parsed = new URL(url);
    expect(parsed.searchParams.get("__data")).toBe("1");
    expect(init.credentials).toBe("same-origin");
    expect((init.headers as Record<string, string>).Accept).toBe(
      "application/json",
    );
    expect(data.title).toBe("Dashboard");
  });

  it("callRouteAction appends _action and sends enhancement headers", async () => {
    const fetchMock = mock(
      async (_url: string, _init?: RequestInit) =>
        new Response(JSON.stringify({ code: "SUCCESS", data: { ok: true } }), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
    );
    (globalThis as any).fetch = fetchMock;

    const body = new FormData();
    body.set("email", "a@example.com");
    const payload = await callRouteAction<{ ok: boolean }>(
      "/users",
      "save",
      body,
    );

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const parsed = new URL(url);
    expect(parsed.searchParams.get("_action")).toBe("save");
    expect(init.method).toBe("POST");
    expect(init.credentials).toBe("same-origin");
    expect((init.headers as Record<string, string>).Accept).toBe(
      "application/json",
    );
    expect((init.headers as Record<string, string>)["X-Gospa-Enhance"]).toBe(
      "1",
    );
    expect(payload.data?.ok).toBe(true);
  });

  it("callRouteAction throws RouteActionError by default on non-2xx", async () => {
    const fetchMock = mock(
      async () =>
        new Response(JSON.stringify({ code: "FAIL", error: "Invalid input" }), {
          status: 422,
          headers: { "content-type": "application/json" },
        }),
    );
    (globalThis as any).fetch = fetchMock;

    await expect(
      callRouteAction("/users", "save", new FormData()),
    ).rejects.toThrow("Invalid input");
  });

  it("callRouteAction can return payload for non-2xx when throwOnError=false", async () => {
    const fetchMock = mock(
      async () =>
        new Response(JSON.stringify({ code: "FAIL", error: "Invalid input" }), {
          status: 422,
          headers: { "content-type": "application/json" },
        }),
    );
    (globalThis as any).fetch = fetchMock;

    const payload = await callRouteAction("/users", "save", new FormData(), {
      throwOnError: false,
    });
    expect(payload.code).toBe("FAIL");
  });

  it("invalidateAll posts all=1 payload", async () => {
    const fetchMock = mock(
      async () =>
        new Response(JSON.stringify({ ok: true, invalidated: 0 }), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
    );
    (globalThis as any).fetch = fetchMock;

    await invalidateAll();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const calls = fetchMock.mock.calls as unknown[][];
    const call = calls[0];
    if (!call) throw new Error("Expected invalidateAll to call fetch once");
    const init = call[1] as RequestInit;
    expect(init.method).toBe("POST");
    expect(String(init.body)).toContain(`"all":true`);
  });

  it("refresh reloads current route data endpoint", async () => {
    const fetchMock = mock(
      async (_url: string, _init?: RequestInit) =>
        new Response(JSON.stringify({ data: { ok: true } }), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
    );
    (globalThis as any).fetch = fetchMock;
    window.happyDOM.setURL("https://app.test/dashboard?tab=1");

    await refresh();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url] = fetchMock.mock.calls[0] as [string];
    const parsed = new URL(url);
    expect(parsed.pathname).toBe("/dashboard");
    expect(parsed.searchParams.get("__data")).toBe("1");
  });

  it("prefetchOnHover binds hover prefetch and supports cleanup", async () => {
    const fetchMock = mock(
      async (_url: string, _init?: RequestInit) =>
        new Response(
          "<html><head><title>X</title></head><body></body></html>",
          {
            status: 200,
            headers: { "content-type": "text/html" },
          },
        ),
    );
    (globalThis as any).fetch = fetchMock;

    document.body.innerHTML = `<a id="hover-link" href="/prefetch-target">target</a>`;
    const dispose = prefetchOnHover("#hover-link", { delay: 0 });

    const link = document.getElementById("hover-link") as HTMLAnchorElement;
    link.dispatchEvent(
      new window.MouseEvent("mouseover", { bubbles: true }) as unknown as Event,
    );
    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(fetchMock).toHaveBeenCalledTimes(1);
    dispose();

    link.dispatchEvent(
      new window.MouseEvent("mouseover", { bubbles: true }) as unknown as Event,
    );
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
