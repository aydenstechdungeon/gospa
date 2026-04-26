import"./islands-3hqyeswk.js";

// generated/routesdocsgospasfctestpage.ts
function __gospa_setup_routesdocsgospasfctestpage(element, { props, state }) {
  const __gospa_runtime = window.__GOSPA_RUNTIME_ESM__ || {};
  const __gospa_state = typeof __gospa_runtime.$state === "function" ? __gospa_runtime.$state.bind(__gospa_runtime) : (initial) => ({ value: initial });
  const __gospa_derived = typeof __gospa_runtime.$derived === "function" ? __gospa_runtime.$derived.bind(__gospa_runtime) : (compute) => compute;
  const __gospa_effect = typeof __gospa_runtime.$effect === "function" ? __gospa_runtime.$effect.bind(__gospa_runtime) : (fn) => {
    fn();
    return () => {};
  };
  let notice = __gospa_state("Ready!");
  const __GOSPA_HANDLERS__ = {};
  element.__gospaHandlers = __GOSPA_HANDLERS__;
  const __gospaIslandKey = element.id || element.getAttribute("data-gospa-island") || "routesdocsgospasfctestpage";
  window["__GOSPA_ISLAND_" + __gospaIslandKey + "__"] = { handlers: __GOSPA_HANDLERS__ };
  const scope = (selector) => element.querySelector(selector + "." + "");
}
function mount(element, props, state) {
  __gospa_setup_routesdocsgospasfctestpage(element, { props, state });
}
function hydrate(element, props, state) {
  __gospa_setup_routesdocsgospasfctestpage(element, { props, state });
}
var routesdocsgospasfctestpage_default = { mount, hydrate };
export {
  mount,
  hydrate,
  routesdocsgospasfctestpage_default as default
};
