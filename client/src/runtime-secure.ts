// GoSPA Client Runtime - Secure version with DOMPurify
// Use this runtime when handling user-generated content or when you need
// client-side HTML sanitization in addition to Templ's server-side escaping
//
// Bundle size: ~35KB (includes DOMPurify)
//
// Usage:
//   import { init } from 'gospa/runtime-secure';
//   init();

import { domPurifySanitizer, preloadSanitizer } from './sanitize.ts';
import { setSanitizer } from './dom.ts';

// Configure DOMPurify sanitizer for this runtime
setSanitizer(domPurifySanitizer);

// Preload DOMPurify immediately to ensure it's ready for first HTML binding
if (typeof window !== 'undefined') {
	const schedulePreload = window.requestIdleCallback || ((cb: () => void) => setTimeout(cb, 1));
	schedulePreload(() => preloadSanitizer());
}

// Re-export everything from the standard runtime
export * from './runtime.ts';

// Also export sanitization utilities for manual use
export { domPurifySanitizer, sanitize, sanitizeSync, isSanitizerReady, preloadSanitizer } from './sanitize.ts';
export { setSanitizer } from './dom.ts';
