import { describe, it, expect, beforeEach } from "bun:test";
import {
  find,
  findAll,
  addClass,
  removeClass,
  hasClass,
  attr,
  data,
} from "./dom";
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
    const el = find("#test");
    expect(el).not.toBeNull();
    expect(el?.id).toBe("test");
  });

  it("should find all elements by selector", () => {
    const elements = findAll(".child");
    expect(elements.length).toBe(2);
  });

  it("should manage classes correctly", () => {
    const el = find("#test") as HTMLElement;

    expect(hasClass(el, "foo")).toBe(true);

    addClass(el, "baz");
    expect(hasClass(el, "baz")).toBe(true);

    removeClass(el, "foo");
    expect(hasClass(el, "foo")).toBe(false);
  });

  it("should manage attributes correctly", () => {
    const el = find("#test") as HTMLElement;

    expect(attr(el, "title")).toBe("test-title");

    attr(el, "title", "new-title");
    expect(el.getAttribute("title")).toBe("new-title");
  });

  it("should manage data attributes correctly", () => {
    const el = find("#test") as HTMLElement;

    expect(data(el, "info")).toBe("test-data");

    data(el, "info", "new-data");
    expect(el.dataset.info).toBe("new-data");
  });
});
