/**
 * Auto-generated islands entry for GoSPA
 * Do not edit manually
 */

function registerLazySetup(name: string, loader: () => Promise<any>) {
  (window as any).__GOSPA_SETUPS__ = (window as any).__GOSPA_SETUPS__ || {};
  (window as any).__GOSPA_SETUPS__[name] = async (el: Element, props: Record<string, any>, state: any) => {
    const mod = await loader();
    const hydrateFn = mod.hydrate || mod.default?.hydrate || mod.mount || mod.default?.mount;
    if (hydrateFn) {
      return hydrateFn(el, props, state);
    }
  };
}

registerLazySetup('docsgospasfcpage', () => import('./docsgospasfcpage.ts'));
