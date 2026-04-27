import {
  __require,
  __toESM
} from "./islands-3hqyeswk.js";

// generated/islands.ts
function registerLazySetup(name, loader) {
  window.__GOSPA_SETUPS__ = window.__GOSPA_SETUPS__ || {};
  window.__GOSPA_SETUPS__[name] = async (el, props, state) => {
    const mod = await loader();
    const hydrateFn = mod.hydrate || mod.default?.hydrate || mod.mount || mod.default?.mount;
    if (hydrateFn) {
      return hydrateFn(el, props, state);
    }
  };
}
registerLazySetup("routesdocsgospasfcpage", () => import("./routesdocsgospasfcpage-m8v73m06.js"));
