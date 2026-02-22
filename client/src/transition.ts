// GoSPA Transition Engine
// Provides Svelte-like enter/leave animations

export interface TransitionConfig {
    delay?: number;
    duration?: number;
    easing?: string;
    css?: (t: number, u: number) => string;
    tick?: (t: number, u: number) => void;
}

export type TransitionFn = (node: Element, params: any) => TransitionConfig;

export const linear = (t: number) => t;
export const cubicOut = (t: number) => {
    const f = t - 1.0;
    return f * f * f + 1.0;
};
export const cubicInOut = (t: number) => {
    return t < 0.5 ? 4.0 * t * t * t : 0.5 * Math.pow(2.0 * t - 2.0, 3.0) + 1.0;
};

// Built-in transitions
export function fade(node: Element, { delay = 0, duration = 400, easing = linear } = {}): TransitionConfig {
    const o = +getComputedStyle(node).opacity;
    return {
        delay,
        duration,
        easing: 'linear', // we map the easing function mechanically in JS or via CSS
        css: (t) => `opacity: ${t * o}`
    };
}

export function fly(node: Element, { delay = 0, duration = 400, easing = cubicOut, x = 0, y = 0, opacity = 0 } = {}): TransitionConfig {
    const style = getComputedStyle(node);
    const targetOpacity = +style.opacity;
    const transform = style.transform === 'none' ? '' : style.transform;

    return {
        delay,
        duration,
        easing: 'ease-out',
        css: (t, u) => `
			transform: ${transform} translate(${(1 - t) * x}px, ${(1 - t) * y}px);
			opacity: ${targetOpacity - (targetOpacity - opacity) * u}
		`
    };
}

export function slide(node: Element, { delay = 0, duration = 400, easing = cubicOut } = {}): TransitionConfig {
    const style = getComputedStyle(node);
    const opacity = +style.opacity;
    const height = parseFloat(style.height);
    const paddingTop = parseFloat(style.paddingTop);
    const paddingBottom = parseFloat(style.paddingBottom);
    const marginTop = parseFloat(style.marginTop);
    const marginBottom = parseFloat(style.marginBottom);
    const borderTopWidth = parseFloat(style.borderTopWidth);
    const borderBottomWidth = parseFloat(style.borderBottomWidth);

    return {
        delay,
        duration,
        easing: 'ease-out',
        css: (t) => `
			overflow: hidden;
			opacity: ${Math.min(t * 20, 1) * opacity};
			height: ${t * height}px;
			padding-top: ${t * paddingTop}px;
			padding-bottom: ${t * paddingBottom}px;
			margin-top: ${t * marginTop}px;
			margin-bottom: ${t * marginBottom}px;
			border-top-width: ${t * borderTopWidth}px;
			border-bottom-width: ${t * borderBottomWidth}px;
		`
    };
}

const activeTransitions = new Set<Element>();

// Apply an enter transition
export function transitionIn(node: HTMLElement, fn: TransitionFn, params: any) {
    if (activeTransitions.has(node)) return;
    activeTransitions.add(node);

    const config = fn(node, params);
    const duration = config.duration || 400;
    const delay = config.delay || 0;
    const css = config.css || (() => '');

    const originalStyle = node.getAttribute('style') || '';

    // Pre-calculate frames (simple CSS keyframes approach)
    const name = `gospa-transition-${Math.random().toString(36).substring(2, 9)}`;
    const keyframes = `
		@keyframes ${name} {
			0% { ${css(0, 1)} }
			100% { ${css(1, 0)} }
		}
	`;

    const styleSheet = document.createElement('style');
    styleSheet.textContent = keyframes;
    document.head.appendChild(styleSheet);

    node.style.animation = `${name} ${duration}ms ${config.easing || 'linear'} ${delay}ms both`;

    setTimeout(() => {
        node.setAttribute('style', originalStyle);
        node.style.animation = '';
        styleSheet.remove();
        activeTransitions.delete(node);
    }, duration + delay);
}

// Apply a leave transition
export function transitionOut(node: HTMLElement, fn: TransitionFn, params: any, onComplete: () => void) {
    if (activeTransitions.has(node)) return;
    activeTransitions.add(node);

    const config = fn(node, params);
    const duration = config.duration || 400;
    const delay = config.delay || 0;
    const css = config.css || (() => '');

    const name = `gospa-transition-${Math.random().toString(36).substring(2, 9)}`;
    const keyframes = `
		@keyframes ${name} {
			0% { ${css(1, 0)} }
			100% { ${css(0, 1)} }
		}
	`;

    const styleSheet = document.createElement('style');
    styleSheet.textContent = keyframes;
    document.head.appendChild(styleSheet);

    node.style.animation = `${name} ${duration}ms ${config.easing || 'linear'} ${delay}ms both`;

    setTimeout(() => {
        styleSheet.remove();
        activeTransitions.delete(node);
        onComplete();
    }, duration + delay);
}

// Setup automated transitions using attributes like data-transition="fade"
export function setupTransitions(root: Element = document.body) {
    const observer = new MutationObserver((mutations) => {
        mutations.forEach(mutation => {
            if (mutation.type === 'childList') {
                // Handle added nodes
                mutation.addedNodes.forEach(node => {
                    if (node.nodeType === Node.ELEMENT_NODE) {
                        const el = node as HTMLElement;
                        if (el.closest('[data-gospa-static]')) return;
                        const transitionType = el.getAttribute('data-transition-in') || el.getAttribute('data-transition');
                        if (transitionType) {
                            const fn = getTransitionFn(transitionType);
                            if (fn) transitionIn(el, fn, getTransitionParams(el));
                        }
                    }
                });

                // Handle removed nodes (requires some DOM trickery to delay true removal)
                mutation.removedNodes.forEach(node => {
                    if (node.nodeType === Node.ELEMENT_NODE) {
                        const el = node as HTMLElement;
                        if (el.closest('[data-gospa-static]')) return;

                        const transitionType = el.getAttribute('data-transition-out') || el.getAttribute('data-transition');
                        if (transitionType) {
                            const fn = getTransitionFn(transitionType);
                            if (fn && !activeTransitions.has(el)) {
                                // We have to temporarily put it back in the DOM to animate it out
                                const clone = el.cloneNode(true) as HTMLElement;

                                // Clean up clone to prevent rogue bindings
                                clone.querySelectorAll('[data-bind]').forEach(n => n.removeAttribute('data-bind'));
                                clone.removeAttribute('data-bind');

                                if (mutation.previousSibling && mutation.previousSibling.parentNode) {
                                    mutation.previousSibling.parentNode.insertBefore(clone, mutation.previousSibling.nextSibling);
                                } else if (mutation.target) {
                                    mutation.target.appendChild(clone);
                                }

                                transitionOut(clone, fn, getTransitionParams(el), () => clone.remove());
                            }
                        }
                    }
                });
            }
        });
    });

    observer.observe(root, { childList: true, subtree: true });
}

function getTransitionFn(name: string): TransitionFn | null {
    if (name.startsWith('fade')) return fade;
    if (name.startsWith('fly')) return fly;
    if (name.startsWith('slide')) return slide;
    return null;
}

function getTransitionParams(node: Element): any {
    const paramStr = node.getAttribute('data-transition-params');
    if (!paramStr) return {};
    try {
        return JSON.parse(paramStr);
    } catch (e) {
        console.warn('Invalid transition parameters:', paramStr);
        return {};
    }
}
