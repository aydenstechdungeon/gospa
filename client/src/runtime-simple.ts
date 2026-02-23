// GoSPA Client Runtime - Simple version
// A lightweight runtime using a basic HTML sanitizer for higher performance
//
// This file includes all features including WebSocket, Navigation, Transitions

import { simpleSanitizer } from './sanitize-simple.ts';
import { setSanitizer } from './dom.ts';

// Set up the simple HTML sanitizer for this runtime version (performance over security)
setSanitizer(simpleSanitizer);

// Core exports (re-exported from runtime-core for convenience)
export {
    init, createComponent, destroyComponent, getComponent, getState, setState, callAction, bind, autoInit,
    getWebSocket, getNavigation, getTransitions
} from './runtime-core.ts';

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
