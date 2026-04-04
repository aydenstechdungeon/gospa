import { describe, it, expect, beforeEach } from "bun:test";
import { GlobalWindow } from "happy-dom";

// Setup DOM environment
const window = new GlobalWindow();
globalThis.window = window as any;
globalThis.document = window.document as any;
globalThis.Element = window.Element as any;
globalThis.HTMLElement = window.HTMLElement as any;

describe("dom utils", () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <div id="test" class="foo bar" data-info="test-data" title="test-title">
        <span class="child">Child 1</span>
        <span class="child">Child 2</span>
      </div>
    `;
  });

  it("should find an element by selector", () => {
    const el = document.querySelector("#test");
    expect(el).not.toBeNull();
    expect((el as HTMLElement).id).toBe("test");
  });

  it("should find all elements by selector", () => {
    const elements = document.querySelectorAll(".child");
    expect(elements.length).toBe(2);
  });

  it("should manage classes correctly", () => {
    const el = document.querySelector("#test") as HTMLElement;

    expect(el.classList.contains("foo")).toBe(true);

    el.classList.add("baz");
    expect(el.classList.contains("baz")).toBe(true);

    el.classList.remove("foo");
    expect(el.classList.contains("foo")).toBe(false);
  });

  it("should manage attributes correctly", () => {
    const el = document.querySelector("#test") as HTMLElement;

    expect(el.getAttribute("title")).toBe("test-title");

    el.setAttribute("title", "new-title");
    expect(el.getAttribute("title")).toBe("new-title");
  });

  it("should manage data attributes correctly", () => {
    const el = document.querySelector("#test") as HTMLElement;

    expect(el.dataset.info).toBe("test-data");

    el.dataset.info = "new-data";
    expect(el.dataset.info).toBe("new-data");
  });
});
