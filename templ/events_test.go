package templ

import (
	"strings"
	"testing"
)

func TestEventHelpers(t *testing.T) {
	if On("click", "inc")["data-on"] != "click:inc" {
		t.Fatal("On failed")
	}
	if OnWithModifiers("click", "save", "prevent", "stop")["data-on"] != "click:save:prevent|stop" {
		t.Fatal("OnWithModifiers failed")
	}
	if OnClick("h")["data-on"] != "click:h" || OnClickPrevent("h")["data-on"] != "click:h:prevent" {
		t.Fatal("click helpers failed")
	}
	if OnInput("h")["data-on"] != "input:h" || OnChange("h")["data-on"] != "change:h" || OnSubmit("h")["data-on"] != "submit:h:prevent" {
		t.Fatal("input/change/submit helpers failed")
	}
	if OnKeydown("h")["data-on"] != "keydown:h" || OnKeyup("h")["data-on"] != "keyup:h" {
		t.Fatal("key helpers failed")
	}
	if OnFocus("h")["data-on"] != "focus:h" || OnBlur("h")["data-on"] != "blur:h" {
		t.Fatal("focus helpers failed")
	}
	if OnMouseenter("h")["data-on"] != "mouseenter:h" || OnMouseleave("h")["data-on"] != "mouseleave:h" {
		t.Fatal("mouse helpers failed")
	}
	if Debounced("input", "search", 300)["data-debounce"] != "300" || Throttled("input", "search", 100)["data-throttle"] != "100" {
		t.Fatal("debounce/throttle helpers failed")
	}
	if OnKey("Enter", "submit")["data-on-key"] != "Enter:submit" ||
		OnCtrlKey("s", "save")["data-on-key"] != "ctrl+s:save" ||
		OnShiftKey("Tab", "prev")["data-on-key"] != "shift+Tab:prev" ||
		OnAltKey("n", "new")["data-on-key"] != "alt+n:new" ||
		OnKeyCombo("ctrl+shift+s", "saveAll")["data-on-key"] != "ctrl+shift+s:saveAll" {
		t.Fatal("key combo helpers failed")
	}

	keys := OnKeys(map[string]string{"Enter": "submit", "Escape": "cancel"})["data-on-key"].(string)
	if !strings.Contains(keys, "Enter:submit") || !strings.Contains(keys, "Escape:cancel") {
		t.Fatalf("OnKeys failed: %q", keys)
	}

	if ServerAction("delete", "id=1")["data-action"] != "delete" || ServerAction("delete", "id=1")["data-params"] != "id=1" {
		t.Fatal("ServerAction failed")
	}
	jsonAction := ServerActionJSON("update", map[string]any{"id": 1})
	if jsonAction["data-action"] != "update" || jsonAction["data-action-json"] != "true" {
		t.Fatal("ServerActionJSON failed")
	}

	if FormAction("create")["data-form-action"] != "create" || Navigate("/docs")["data-navigate"] != "/docs" || NavigateBack()["data-navigate"] != "back" {
		t.Fatal("navigation/form helpers failed")
	}
	if ScrollTo("section-1")["data-scroll-to"] != "section-1" || ScrollToTop()["data-scroll"] != "top" {
		t.Fatal("scroll helpers failed")
	}
	if Toggle("open")["data-toggle"] != "open" || Increment("count", 2)["data-amount"] != "2" || Decrement("count", 1)["data-amount"] != "1" {
		t.Fatal("state mutation helpers failed")
	}
	if SetState("filter", "active")["data-set"] != "filter" || SetState("filter", "active")["data-value"] != "active" {
		t.Fatal("SetState failed")
	}

	if PreventDefault()["data-prevent"] != "true" || StopPropagation()["data-stop"] != "true" || Once()["data-once"] != "true" {
		t.Fatal("modifier helpers failed")
	}
	if Passive()["data-passive"] != "true" || Capture()["data-capture"] != "true" || Self()["data-self"] != "true" {
		t.Fatal("passive/capture/self helpers failed")
	}
	if EventModifiers("prevent", "stop")["data-modifiers"] != "prevent|stop" {
		t.Fatal("EventModifiers failed")
	}
}
