// generated/routesdocsgospasfcpage.ts
function __gospa_setup_routesdocsgospasfcpage(element, { props, state }) {
  console.log("Gospa SFC Documentation loaded");
  console.log("Hello from TypeScript!");
  const __GOSPA_HANDLERS__ = {};
  window["__GOSPA_ISLAND_" + "routesdocsgospasfcpage" + "__"] = { handlers: __GOSPA_HANDLERS__ };
  const scope = (selector) => element.querySelector(selector + "." + "gospa-9d0f");
}
function mount(element, props, state) {
  __gospa_setup_routesdocsgospasfcpage(element, { props, state });
}

// generated/islands.ts
function registerSetup(name, setup) {
  window.__GOSPA_SETUPS__ = window.__GOSPA_SETUPS__ || {};
  window.__GOSPA_SETUPS__[name] = setup;
}
registerSetup("routesdocsgospasfcpage", (el, props, state) => {
  mount(el, props, state);
});
