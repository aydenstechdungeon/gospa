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
  const statusEl = document.getElementById("action-status");
  const forms = Array.from(document.querySelectorAll("form[data-action-form]"));
  function setStatus(message, isError = false) {
    if (!statusEl)
      return;
    statusEl.textContent = message;
    statusEl.setAttribute("data-status", isError ? "error" : "success");
  }
  async function submitEnhanced(form) {
    const submitButton = form.querySelector("button[type='submit']");
    if (submitButton)
      submitButton.disabled = true;
    try {
      const action = form.getAttribute("action");
      const url = action && action.length > 0 ? action : window.location.pathname + window.location.search;
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "X-Gospa-Enhance": "1",
          Accept: "application/json"
        },
        body: new FormData(form)
      });
      const payload = await response.json();
      if (!response.ok || payload.code === "FAIL") {
        const errorMsg = payload.error || "Action failed";
        setStatus(errorMsg, true);
        return;
      }
      const data = payload.data || {};
      const fallbackMsg = form.getAttribute("data-success-message") || "Success";
      setStatus(data.message || fallbackMsg);
    } catch (_err) {
      setStatus("Network error", true);
    } finally {
      if (submitButton)
        submitButton.disabled = false;
    }
  }
  for (const form of forms) {
    form.addEventListener("submit", (event) => {
      event.preventDefault();
      submitEnhanced(form);
    });
  }
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
