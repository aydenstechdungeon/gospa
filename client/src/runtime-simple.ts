// GoSPA Client Runtime - Simple version (DEPRECATED)
// ⚠️ DEPRECATED: This runtime variant is deprecated.
//
// Migration paths:
// - For default usage: Use 'gospa' (trust-the-server, no sanitization needed)
// - For user-generated content: Use 'gospa/runtime-secure' (includes DOMPurify)
//
// This variant used a basic HTML sanitizer which is no longer recommended.

import { createSimpleSanitizer, simpleSanitizer } from './sanitize-simple.ts';
import { setSanitizer } from './dom.ts';
import {
	init as coreInit,
	type RuntimeConfig,
	createComponent,
	destroyComponent,
	getComponent,
	getState,
	setState,
	callAction,
	bind,
	autoInit,
	getWebSocket,
	getNavigation,
	getTransitions,
} from './runtime-core.ts';

// Set up the default simple HTML sanitizer (will be reconfigured if SVGs are enabled)
setSanitizer(simpleSanitizer);

// Track if we've configured the sanitizer
let sanitizerConfigured = false;

// Track if we've shown the deprecation warning
let deprecationWarned = false;

// Extended init that configures sanitizer based on options
function init(options: RuntimeConfig = {}): void {
	// Show deprecation warning once
	if (!deprecationWarned && typeof console !== 'undefined') {
		deprecationWarned = true;
		console.warn(
			'[GoSPA DEPRECATION] gospa/simple is deprecated and will be removed in v2.0.\n' +
			'Migration: Use "gospa" for server-trusted content or "gospa/runtime-secure" for UGC.\n' +
			'See: https://gospa.dev/docs/migration-v2'
		);
	}

	// Configure sanitizer before core init if SVGs are enabled
	if (options.simpleRuntimeSVGs && !sanitizerConfigured) {
		const svgAwareSanitizer = createSimpleSanitizer({ allowSVGs: true, allowMath: true });
		setSanitizer(svgAwareSanitizer);
		sanitizerConfigured = true;
		if (options.debug) {
			console.warn('GoSPA: SVG/math elements enabled in simple runtime sanitizer. WARNING: This is a security risk for untrusted content.');
		}
	}
	
	// Call core init
	coreInit(options);
}

// Core exports (re-exported from runtime-core for convenience)
export {
	init,
	createComponent,
	destroyComponent,
	getComponent,
	getState,
	setState,
	callAction,
	bind,
	autoInit,
	getWebSocket,
	getNavigation,
	getTransitions,
};

export {
    Rune, Derived, Effect, StateMap, batch, effect, watch,
    bindElement, bindTwoWay, renderIf, renderList
} from './runtime-core.ts';

// Export types
export type { ComponentDefinition, ComponentInstance, RuntimeConfig } from './runtime-core.ts';
export type { Unsubscribe } from './state.ts';

// Direct imports for backward compatibility
import { registerBinding, unregisterBinding } from './dom.ts';
import { on, offAll, debounce, throttle, delegate, onKey, keys, transformers } from './events.ts';
import { WSClient, initWebSocket, getWebSocketClient, sendAction, syncedRune, applyStateUpdate, type StateMessage } from './websocket.ts';
import {
    navigate, back, forward, go, prefetch, getCurrentPath, isNavigating,
    onBeforeNavigate, onAfterNavigate, initNavigation, destroyNavigation,
    createNavigationState, type NavigationOptions
} from './navigation.ts';
import { setupTransitions, fade, fly, slide, scale, blur, crossfade } from './transition.ts';

// Re-export DOM bindings
export { registerBinding, unregisterBinding };

// Re-export events
export { on, offAll, debounce, throttle, delegate, onKey, keys, transformers };

// Re-export WebSocket, Navigation, and Transition APIs
export {
    // WebSocket
    WSClient, initWebSocket, getWebSocketClient, sendAction, syncedRune, applyStateUpdate,
    // Transitions
    fade, fly, slide, scale, blur, crossfade, setupTransitions,
    // Navigation
    navigate, back, forward, go, prefetch, getCurrentPath, isNavigating,
    onBeforeNavigate, onAfterNavigate, initNavigation, destroyNavigation, createNavigationState
};

export type { NavigationOptions, StateMessage };
