import { describe, it, expect, beforeEach, mock } from "bun:test";
import { GlobalWindow } from "happy-dom";
import {
  parseEventString,
  on,
  offAll,
  debounce,
  throttle,
  bindEvent,
  transformers,
  delegate,
  onKey,
  setupEventDelegation,
} from "./events";
import { Rune } from "./state";

const window = new GlobalWindow();
(globalThis as any).window = window;
(globalThis as any).document = window.document;
(globalThis as any).Element = window.Element;
(globalThis as any).HTMLElement = window.HTMLElement;
(globalThis as any).HTMLInputElement = window.HTMLInputElement;
(globalThis as any).HTMLFormElement = window.HTMLFormElement;
(globalThis as any).KeyboardEvent = window.KeyboardEvent;
(globalThis as any).Event = window.Event;
(globalThis as any).FormData = window.FormData;

describe("events", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
  });

  it("parses event strings with modifiers", () => {
    expect(parseEventString("click:prevent:stop")).toEqual({
      event: "click",
      modifiers: ["prevent", "stop"],
    });
  });

  it("applies prevent/stop modifiers and unregisters handler", () => {
    const button = document.createElement("button");
    document.body.appendChild(button);

    const handler = mock((_e: Event) => {});
    const cleanup = on(button, "click:prevent:stop", handler);

    const event = new window.MouseEvent("click", {
      bubbles: true,
      cancelable: true,
    });

    const stopSpy = mock(() => {});
    event.stopPropagation = stopSpy as any;

    button.dispatchEvent(event);

    expect(event.defaultPrevented).toBe(true);
    expect(stopSpy).toHaveBeenCalledTimes(1);
    expect(handler).toHaveBeenCalledTimes(1);

    cleanup();
    button.dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("honors self modifier", () => {
    const parent = document.createElement("div");
    const child = document.createElement("button");
    parent.appendChild(child);
    document.body.appendChild(parent);

    const handler = mock((_e: Event) => {});
    on(parent, "click:self", handler);

    child.dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
    parent.dispatchEvent(new window.MouseEvent("click", { bubbles: true }));

    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("offAll removes listeners for a target", () => {
    const input = document.createElement("input");
    const clickHandler = mock(() => {});
    const changeHandler = mock(() => {});

    on(input, "click", clickHandler);
    on(input, "change", changeHandler);

    offAll(input);

    input.dispatchEvent(new window.MouseEvent("click"));
    input.dispatchEvent(new window.Event("change"));

    expect(clickHandler).toHaveBeenCalledTimes(0);
    expect(changeHandler).toHaveBeenCalledTimes(0);
  });

  it("debounce and throttle control event burst behavior", async () => {
    const debouncedFn = mock(() => {});
    const d = debounce(debouncedFn, 10);
    d.handler(new window.Event("input"));
    d.handler(new window.Event("input"));
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(debouncedFn).toHaveBeenCalledTimes(1);

    const throttledFn = mock(() => {});
    const t = throttle(throttledFn, 15);
    t.handler(new window.Event("click"));
    t.handler(new window.Event("click"));
    t.handler(new window.Event("click"));
    await new Promise((resolve) => setTimeout(resolve, 30));
    expect(throttledFn).toHaveBeenCalledTimes(2);

    t.cancel();
    d.cancel();
  });

  it("bindEvent updates rune through transformer", () => {
    const input = document.createElement("input");
    input.value = "initial";
    const r = new Rune<unknown>("");

    bindEvent(input, "input", r, transformers.value);

    input.value = "next";
    input.dispatchEvent(new window.Event("input"));

    expect(r.get()).toBe("next");
  });

  it("delegates matching selector only", () => {
    const root = document.createElement("div");
    const ok = document.createElement("button");
    ok.className = "ok";
    const nope = document.createElement("button");
    root.appendChild(ok);
    root.appendChild(nope);
    document.body.appendChild(root);

    const handler = mock((_e: Event) => {});
    const cleanup = delegate(root, ".ok", "click", handler);

    ok.dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
    nope.dispatchEvent(new window.MouseEvent("click", { bubbles: true }));

    expect(handler).toHaveBeenCalledTimes(1);

    cleanup();
  });

  it("onKey filters by key and can prevent default", () => {
    const handler = mock((_e: KeyboardEvent) => {});
    const wrapped = onKey(["Enter", "Escape"], handler, {
      preventDefault: true,
    });

    const enter = new window.KeyboardEvent("keydown", {
      key: "Enter",
      cancelable: true,
    });
    const tab = new window.KeyboardEvent("keydown", {
      key: "Tab",
      cancelable: true,
    });

    wrapped(enter);
    wrapped(tab);

    expect(enter.defaultPrevented).toBe(true);
    expect(tab.defaultPrevented).toBe(false);
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("setupEventDelegation calls island handlers for matching events", () => {
    document.body.innerHTML = `
      <div id="root">
        <div id="isl-1" data-gospa-island="Counter">
          <button id="btn" data-gospa-on="click:save"></button>
        </div>
      </div>
    `;

    const root = document.getElementById("root")!;
    const btn = document.getElementById("btn")!;

    const save = mock((_e: Event) => {});
    (window as any)["__GOSPA_ISLAND_isl-1__"] = {
      handlers: {
        save,
      },
    };

    setupEventDelegation(root);
    btn.dispatchEvent(new window.MouseEvent("click", { bubbles: true }));

    expect(save).toHaveBeenCalledTimes(1);
  });
});
