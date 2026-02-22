// Package templ provides event handler helpers for GoSPA.
package templ

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"
)

// EventHandler represents a client-side event handler.
type EventHandler struct {
	// Event is the event name (click, input, change, etc.)
	Event string
	// Handler is the handler function name or expression.
	Handler string
	// Modifiers are event modifiers (prevent, stop, capture, etc.)
	Modifiers []string
	// Debounce is the debounce duration in milliseconds (0 = no debounce).
	Debounce int
	// Throttle is the throttle duration in milliseconds (0 = no throttle).
	Throttle int
}

// On creates an event handler attribute.
// Usage: <button { templ.On("click", "increment") }>Click me</button>
func On(event string, handler string) templ.Attributes {
	return templ.Attributes{
		"data-on": event + ":" + handler,
	}
}

// OnWithModifiers creates an event handler with modifiers.
// Usage: <button { templ.OnWithModifiers("click", "submit", "prevent", "stop") }>Submit</button>
func OnWithModifiers(event string, handler string, modifiers ...string) templ.Attributes {
	modStr := strings.Join(modifiers, "|")
	return templ.Attributes{
		"data-on": event + ":" + handler + ":" + modStr,
	}
}

// OnClick creates a click handler.
// Usage: <button { templ.OnClick("handleClick") }>Click</button>
func OnClick(handler string) templ.Attributes {
	return On("click", handler)
}

// OnClickPrevent creates a click handler with preventDefault.
// Usage: <a href="/link" { templ.OnClickPrevent("handleClick") }>Click</a>
func OnClickPrevent(handler string) templ.Attributes {
	return OnWithModifiers("click", handler, "prevent")
}

// OnInput creates an input handler.
// Usage: <input { templ.OnInput("handleInput") } />
func OnInput(handler string) templ.Attributes {
	return On("input", handler)
}

// OnChange creates a change handler.
// Usage: <select { templ.OnChange("handleChange") }></select>
func OnChange(handler string) templ.Attributes {
	return On("change", handler)
}

// OnSubmit creates a submit handler with preventDefault.
// Usage: <form { templ.OnSubmit("handleSubmit") }></form>
func OnSubmit(handler string) templ.Attributes {
	return OnWithModifiers("submit", handler, "prevent")
}

// OnKeydown creates a keydown handler.
// Usage: <input { templ.OnKeydown("handleKeydown") } />
func OnKeydown(handler string) templ.Attributes {
	return On("keydown", handler)
}

// OnKeyup creates a keyup handler.
// Usage: <input { templ.OnKeyup("handleKeyup") } />
func OnKeyup(handler string) templ.Attributes {
	return On("keyup", handler)
}

// OnFocus creates a focus handler.
// Usage: <input { templ.OnFocus("handleFocus") } />
func OnFocus(handler string) templ.Attributes {
	return On("focus", handler)
}

// OnBlur creates a blur handler.
// Usage: <input { templ.OnBlur("handleBlur") } />
func OnBlur(handler string) templ.Attributes {
	return On("blur", handler)
}

// OnMouseenter creates a mouseenter handler.
// Usage: <div { templ.OnMouseenter("handleEnter") }></div>
func OnMouseenter(handler string) templ.Attributes {
	return On("mouseenter", handler)
}

// OnMouseleave creates a mouseleave handler.
// Usage: <div { templ.OnMouseleave("handleLeave") }></div>
func OnMouseleave(handler string) templ.Attributes {
	return On("mouseleave", handler)
}

// Debounced creates a debounced event handler.
// Usage: <input { templ.Debounced("input", "search", 300) } />
func Debounced(event string, handler string, ms int) templ.Attributes {
	return templ.Attributes{
		"data-on":       event + ":" + handler,
		"data-debounce": fmt.Sprintf("%d", ms),
	}
}

// Throttled creates a throttled event handler.
// Usage: <input { templ.Throttled("input", "search", 100) } />
func Throttled(event string, handler string, ms int) templ.Attributes {
	return templ.Attributes{
		"data-on":       event + ":" + handler,
		"data-throttle": fmt.Sprintf("%d", ms),
	}
}

// OnKey creates a keyboard event handler for specific keys.
// Usage: <input { templ.OnKey("Enter", "submit") } />
func OnKey(key string, handler string) templ.Attributes {
	return templ.Attributes{
		"data-on-key": key + ":" + handler,
	}
}

// OnKeys creates a keyboard event handler for multiple keys.
// Usage: <input { templ.OnKeys(map[string]string{"Enter": "submit", "Escape": "cancel"}) } />
func OnKeys(handlers map[string]string) templ.Attributes {
	var parts []string
	for key, handler := range handlers {
		parts = append(parts, key+":"+handler)
	}
	return templ.Attributes{
		"data-on-key": strings.Join(parts, ","),
	}
}

// OnCtrlKey creates a handler for Ctrl/Cmd + key combination.
// Usage: <input { templ.OnCtrlKey("s", "save") } />
func OnCtrlKey(key string, handler string) templ.Attributes {
	return templ.Attributes{
		"data-on-key": "ctrl+" + key + ":" + handler,
	}
}

// OnShiftKey creates a handler for Shift + key combination.
// Usage: <input { templ.OnShiftKey("Tab", "focusPrev") } />
func OnShiftKey(key string, handler string) templ.Attributes {
	return templ.Attributes{
		"data-on-key": "shift+" + key + ":" + handler,
	}
}

// OnAltKey creates a handler for Alt + key combination.
// Usage: <input { templ.OnAltKey("n", "newItem") } />
func OnAltKey(key string, handler string) templ.Attributes {
	return templ.Attributes{
		"data-on-key": "alt+" + key + ":" + handler,
	}
}

// OnKeyCombo creates a handler for key combinations.
// Usage: <input { templ.OnKeyCombo("ctrl+shift+s", "saveAll") } />
func OnKeyCombo(combo string, handler string) templ.Attributes {
	return templ.Attributes{
		"data-on-key": combo + ":" + handler,
	}
}

// ServerAction creates a server action handler.
// Usage: <button { templ.ServerAction("deleteItem", "itemId=123") }>Delete</button>
func ServerAction(action string, params string) templ.Attributes {
	return templ.Attributes{
		"data-action": action,
		"data-params": params,
	}
}

// ServerActionJSON creates a server action handler with JSON params.
// Usage: <button { templ.ServerActionJSON("updateItem", map[string]any{"id": 123, "name": "test"}) }>Update</button>
func ServerActionJSON(action string, params map[string]any) templ.Attributes {
	// JSON encoding will be handled by the runtime
	return templ.Attributes{
		"data-action":      action,
		"data-action-json": "true",
		"data-params":      params,
	}
}

// FormAction creates a form action handler.
// Usage: <button type="submit" { templ.FormAction("createUser") }>Create</button>
func FormAction(action string) templ.Attributes {
	return templ.Attributes{
		"data-form-action": action,
	}
}

// Navigate creates a client-side navigation handler.
// Usage: <button { templ.Navigate("/about") }>Go to About</button>
func Navigate(path string) templ.Attributes {
	return templ.Attributes{
		"data-navigate": path,
	}
}

// NavigateBack creates a back navigation handler.
// Usage: <button { templ.NavigateBack() }>Go Back</button>
func NavigateBack() templ.Attributes {
	return templ.Attributes{
		"data-navigate": "back",
	}
}

// ScrollTo creates a scroll-to-element handler.
// Usage: <button { templ.ScrollTo("section-1") }>Scroll to Section 1</button>
func ScrollTo(elementID string) templ.Attributes {
	return templ.Attributes{
		"data-scroll-to": elementID,
	}
}

// ScrollToTop creates a scroll-to-top handler.
// Usage: <button { templ.ScrollToTop() }>Back to Top</button>
func ScrollToTop() templ.Attributes {
	return templ.Attributes{
		"data-scroll": "top",
	}
}

// Toggle creates a toggle handler for boolean state.
// Usage: <button { templ.Toggle("isExpanded") }>Toggle</button>
func Toggle(key string) templ.Attributes {
	return templ.Attributes{
		"data-toggle": key,
	}
}

// Increment creates an increment handler for numeric state.
// Usage: <button { templ.Increment("count", 1) }>+</button>
func Increment(key string, amount int) templ.Attributes {
	return templ.Attributes{
		"data-increment": key,
		"data-amount":    fmt.Sprintf("%d", amount),
	}
}

// Decrement creates a decrement handler for numeric state.
// Usage: <button { templ.Decrement("count", 1) }>-</button>
func Decrement(key string, amount int) templ.Attributes {
	return templ.Attributes{
		"data-decrement": key,
		"data-amount":    fmt.Sprintf("%d", amount),
	}
}

// SetState creates a handler to set state directly.
// Usage: <button { templ.SetState("filter", "active") }>Active</button>
func SetState(key string, value string) templ.Attributes {
	return templ.Attributes{
		"data-set":   key,
		"data-value": value,
	}
}

// PreventDefault creates a preventDefault modifier attribute.
// Usage: <a href="/link" { templ.PreventDefault() } onclick="handleClick">Link</a>
func PreventDefault() templ.Attributes {
	return templ.Attributes{
		"data-prevent": "true",
	}
}

// StopPropagation creates a stopPropagation modifier attribute.
// Usage: <div { templ.StopPropagation() } onclick="handleClick">Click</div>
func StopPropagation() templ.Attributes {
	return templ.Attributes{
		"data-stop": "true",
	}
}

// Once creates a once modifier (handler runs only once).
// Usage: <button { templ.Once() } onclick="showWelcome">Welcome</button>
func Once() templ.Attributes {
	return templ.Attributes{
		"data-once": "true",
	}
}

// Passive creates a passive event listener modifier.
// Usage: <div { templ.Passive() } onscroll="handleScroll">Scrollable</div>
func Passive() templ.Attributes {
	return templ.Attributes{
		"data-passive": "true",
	}
}

// Capture creates a capture event listener modifier.
// Usage: <div { templ.Capture() } onclick="handleCapture">Capture</div>
func Capture() templ.Attributes {
	return templ.Attributes{
		"data-capture": "true",
	}
}

// Self creates a self modifier (only triggers if event.target is the element itself).
// Usage: <div { templ.Self() } onclick="handleClick">Click me only</div>
func Self() templ.Attributes {
	return templ.Attributes{
		"data-self": "true",
	}
}

// EventModifiers combines multiple event modifiers.
// Usage: <button { templ.EventModifiers("prevent", "stop") }>Click</button>
func EventModifiers(modifiers ...string) templ.Attributes {
	return templ.Attributes{
		"data-modifiers": strings.Join(modifiers, "|"),
	}
}
