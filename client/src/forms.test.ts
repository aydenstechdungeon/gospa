import { describe, it, expect, beforeAll, beforeEach, mock } from "bun:test";
import { GlobalWindow } from "happy-dom";

const window = new GlobalWindow();
window.happyDOM.setURL("https://app.test/");
(globalThis as any).window = window;
(globalThis as any).document = window.document;
(globalThis as any).Element = window.Element;
(globalThis as any).HTMLElement = window.HTMLElement;
(globalThis as any).HTMLFormElement = window.HTMLFormElement;
(globalThis as any).HTMLInputElement = window.HTMLInputElement;
(globalThis as any).Event = window.Event;
(globalThis as any).SubmitEvent = window.SubmitEvent;
(globalThis as any).FormData = window.FormData;
(globalThis as any).URL = window.URL;
(globalThis as any).Response = window.Response;
(globalThis as any).IntersectionObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

let enhanceForm: typeof import("./forms").enhanceForm;

describe("forms", () => {
  beforeAll(async () => {
    ({ enhanceForm } = await import("./forms"));
  });

  beforeEach(() => {
    document.body.innerHTML = "";
  });

  it("sends GET form fields as query params without request body", async () => {
    const form = document.createElement("form");
    form.method = "GET";
    form.action = "/search";
    form.dataset.gospaAction = "lookup";

    const q = document.createElement("input");
    q.name = "q";
    q.value = "gospa";
    form.appendChild(q);

    const submit = document.createElement("button");
    submit.type = "submit";
    submit.value = "go";
    submit.setAttribute("data-gospa-action", "lookup");
    form.appendChild(submit);
    document.body.appendChild(form);

    const fetchMock = mock(
      async (_url: string, _init?: RequestInit) =>
        new Response(JSON.stringify({ data: { ok: true } }), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
    );
    (globalThis as any).fetch = fetchMock;

    const cleanup = enhanceForm(form);
    submit.click();
    await Promise.resolve();
    await Promise.resolve();
    cleanup();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const parsed = new URL(url);
    expect(parsed.searchParams.get("q")).toBe("gospa");
    expect(parsed.searchParams.get("_action")).toBe("lookup");
    expect(init.method).toBe("GET");
    expect(init.body).toBeUndefined();
  });

  it("maps non-2xx validation payloads to onValidation", async () => {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = "/save";
    form.dataset.gospaAction = "save";

    const email = document.createElement("input");
    email.name = "email";
    email.value = "bad";
    form.appendChild(email);
    document.body.appendChild(form);

    const fetchMock = mock(
      async () =>
        new Response(
          JSON.stringify({
            code: "FAIL",
            data: {
              fieldErrors: { email: "invalid email" },
              formError: "fix errors",
            },
          }),
          {
            status: 422,
            headers: { "content-type": "application/json" },
          },
        ),
    );
    (globalThis as any).fetch = fetchMock;

    const onValidation = mock(() => {});
    const onError = mock(() => {});
    const cleanup = enhanceForm(form, { onValidation, onError });
    form.requestSubmit();
    await Promise.resolve();
    await Promise.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
    cleanup();

    expect(onValidation).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledTimes(0);
    expect(email.getAttribute("aria-invalid")).toBe("true");
    expect(email.getAttribute("data-gospa-error")).toBe("invalid email");
  });

  it("aborts stale requests and only applies latest response", async () => {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = "/save";
    form.dataset.gospaAction = "save";

    const input = document.createElement("input");
    input.name = "email";
    input.value = "first";
    form.appendChild(input);
    document.body.appendChild(form);

    const resolvers: Array<(value: Response) => void> = [];
    const fetchMock = mock(
      (_url: string, init?: RequestInit) =>
        new Promise<Response>((resolve, reject) => {
          const signal = init?.signal;
          if (signal) {
            signal.addEventListener("abort", () => {
              const abortErr = new Error("Aborted");
              (abortErr as Error & { name: string }).name = "AbortError";
              reject(abortErr);
            });
          }
          resolvers.push(resolve);
        }),
    );
    (globalThis as any).fetch = fetchMock;

    const onSuccess = mock(() => {});
    const onError = mock(() => {});
    const cleanup = enhanceForm(form, { onSuccess, onError });

    form.requestSubmit();
    await Promise.resolve();
    input.value = "second";
    form.requestSubmit();
    await Promise.resolve();

    if (resolvers.length !== 2) {
      throw new Error(`expected two pending requests, got ${resolvers.length}`);
    }

    resolvers[0](
      new Response(JSON.stringify({ data: { value: "stale" } }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    await Promise.resolve();
    await Promise.resolve();

    resolvers[1](
      new Response(JSON.stringify({ data: { value: "latest" } }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    await Promise.resolve();
    await Promise.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
    cleanup();

    expect(onError).toHaveBeenCalledTimes(0);
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });
});
