// GoSPA Client Runtime - Main entry point
// A lightweight runtime for reactive SPAs with Go/Fiber/Templ
//
// For minimal bundle (~15KB), import from './runtime-core.ts' instead
// This file includes all features including WebSocket, Navigation, Transitions

// Core exports (re-exported from runtime-core for convenience)
import { domPurifySanitizer, preloadSanitizer } from './sanitize.ts';
import { setSanitizer } from './dom.ts';

// Set up the full DOMPurify sanitizer for the standard runtime
setSanitizer(domPurifySanitizer);

// Preload DOMPurify immediately to ensure it's ready for first HTML binding
// This runs in the background and caches the instance for sync use later
if (typeof window !== 'undefined') {
	// Use requestIdleCallback for non-blocking preload, or setTimeout as fallback
	const schedulePreload = window.requestIdleCallback || ((cb: () => void) => setTimeout(cb, 1));
	schedulePreload(() => preloadSanitizer());
}

import {
	init, createComponent, destroyComponent, getComponent, getState, setState, callAction, bind, autoInit,
	getWebSocket, getNavigation, getTransitions,
	remote, remoteAction, configureRemote, getRemotePrefix
} from './runtime-core.ts';

export {
	init, createComponent, destroyComponent, getComponent, getState, setState, callAction, bind, autoInit,
	getWebSocket, getNavigation, getTransitions,
	remote, remoteAction, configureRemote, getRemotePrefix
};

export {
	Rune, Derived, Effect, StateMap, batch, effect, watch,
	bindElement, bindTwoWay, renderIf, renderList
} from './runtime-core.ts';


// Export types
export type { ComponentDefinition, ComponentInstance, RuntimeConfig } from './runtime-core.ts';
export type { Unsubscribe } from './state.ts';
export type { RemoteOptions, RemoteResult } from './remote.ts';

// Direct imports for full-featured runtime (backward compatibility)
import { Rune, Derived, Effect, StateMap, batch, effect, watch, type Unsubscribe } from './state.ts';
import { bindElement, bindTwoWay, renderIf, renderList, registerBinding, unregisterBinding } from './dom.ts';
import { on, offAll, debounce, throttle, delegate, onKey, keys, transformers } from './events.ts';
import { WSClient, initWebSocket, getWebSocketClient, sendAction, syncedRune, applyStateUpdate, type StateMessage } from './websocket.ts';
import {
	navigate,
	back,
	forward,
	go,
	prefetch,
	getCurrentPath,
	isNavigating,
	onBeforeNavigate,
	onAfterNavigate,
	initNavigation,
	destroyNavigation,
	createNavigationState,
	type NavigationOptions
} from './navigation.ts';
import { setupTransitions, fade, fly, slide, scale, blur, crossfade } from './transition.ts';

// Re-export DOM bindings
export { registerBinding, unregisterBinding };

// Re-export events
export { on, offAll, debounce, throttle, delegate, onKey, keys, transformers };

// Re-export WebSocket, Navigation, and Transition APIs for backward compatibility
export {
	// WebSocket
	WSClient,
	initWebSocket,
	getWebSocketClient,
	sendAction,
	syncedRune,
	applyStateUpdate,

	// Transitions
	fade,
	fly,
	slide,
	scale,
	blur,
	crossfade,
	setupTransitions,

	// Navigation
	navigate,
	back,
	forward,
	go,
	prefetch,
	getCurrentPath,
	isNavigating,
	onBeforeNavigate,
	onAfterNavigate,
	initNavigation,
	destroyNavigation,
	createNavigationState
};

// Export types
export type { NavigationOptions, StateMessage };

