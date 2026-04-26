import { beforeEach, describe, expect, it } from "bun:test";
import { GlobalWindow } from "happy-dom";
import { IslandManager } from "./island.ts";

const window = new GlobalWindow();
(globalThis as any).window = window;
(globalThis as any).document = window.document;
(globalThis as any).Element = window.Element;
(globalThis as any).HTMLElement = window.HTMLElement;
(globalThis as any).IntersectionObserver = class {
  observe() {}
  disconnect() {}
};

describe("island manager teardown", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
  });

  it("cleans element and global handler references on destroyIsland", () => {
    document.body.innerHTML = `
      <div id="app">
        <div id="isl-1" data-gospa-island="Counter" data-gospa-mode="lazy"></div>
      </div>
    `;

    const el = document.getElementById("isl-1") as HTMLElement;
    (el as any).__gospaHandlers = { save: () => {} };
    (window as any)["__GOSPA_ISLAND_isl-1__"] = {
      handlers: { save: () => {} },
    };
    (window as any)["__GOSPA_ISLAND_Counter__"] = {
      handlers: { save: () => {} },
    };

    const manager = new IslandManager({ debug: false });
    manager.destroyIsland("isl-1");

    expect((el as any).__gospaHandlers).toBeUndefined();
    expect((window as any)["__GOSPA_ISLAND_isl-1__"]).toBeUndefined();
    expect((window as any)["__GOSPA_ISLAND_Counter__"]).toBeUndefined();

    manager.destroy();
  });

  it("prunes disconnected islands and removes stale global handlers", () => {
    document.body.innerHTML = `
      <div id="app">
        <div id="isl-1" data-gospa-island="One" data-gospa-mode="lazy"></div>
        <div id="isl-2" data-gospa-island="Two" data-gospa-mode="lazy"></div>
      </div>
    `;

    const app = document.getElementById("app") as HTMLElement;
    const stale = document.getElementById("isl-1") as HTMLElement;

    (window as any)["__GOSPA_ISLAND_isl-1__"] = {
      handlers: { click: () => {} },
    };
    (window as any)["__GOSPA_ISLAND_isl-2__"] = {
      handlers: { click: () => {} },
    };

    const manager = new IslandManager({ debug: false });
    app.removeChild(stale);
    manager.pruneDisconnectedIslands();

    expect(manager.getIsland("isl-1")).toBeUndefined();
    expect(manager.getIsland("isl-2")).toBeDefined();
    expect((window as any)["__GOSPA_ISLAND_isl-1__"]).toBeUndefined();
    expect((window as any)["__GOSPA_ISLAND_isl-2__"]).toBeDefined();

    manager.destroy();
  });

  it("keeps generated island ids stable across rediscovery", () => {
    document.body.innerHTML = `
      <div id="app">
        <div data-gospa-island="Counter" data-gospa-mode="lazy"></div>
      </div>
    `;

    const manager = new IslandManager({ debug: false });
    const first = manager.getIslands();
    if (first.length !== 1) {
      throw new Error("expected one discovered island");
    }
    const firstID = first[0]?.id;
    if (!firstID) {
      throw new Error("expected discovered island id");
    }

    manager.discoverIslands();
    const second = manager.getIslands();
    expect(second).toHaveLength(1);
    expect(second[0]?.id).toBe(firstID);

    manager.destroy();
  });
});
