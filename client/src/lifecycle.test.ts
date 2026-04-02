import { describe, it, expect, mock } from "bun:test";
import { onMounted, onDestroyed, onUpdated, runHooks } from "./lifecycle";

describe("lifecycle", () => {
  it("should register and run onMounted hooks", () => {
    const callback = mock(() => {});
    onMounted(callback);

    runHooks("mounted");
    expect(callback).toHaveBeenCalled();
  });

  it("should register and run onDestroyed hooks", () => {
    const callback = mock(() => {});
    onDestroyed(callback);

    runHooks("destroyed");
    expect(callback).toHaveBeenCalled();
  });

  it("should register and run onUpdated hooks", () => {
    const callback = mock(() => {});
    onUpdated(callback);

    runHooks("updated");
    expect(callback).toHaveBeenCalled();
  });

  it("should only run hooks for the specified type", () => {
    const mountedCallback = mock(() => {});
    const destroyedCallback = mock(() => {});

    onMounted(mountedCallback);
    onDestroyed(destroyedCallback);

    runHooks("mounted");
    expect(mountedCallback).toHaveBeenCalled();
    expect(destroyedCallback).not.toHaveBeenCalled();
  });
});
