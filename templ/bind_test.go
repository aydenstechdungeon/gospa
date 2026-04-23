package templ

import (
	"context"
	"strings"
	"testing"

	"github.com/aydenstechdungeon/gospa/state"
)

func TestBindingHelpers(t *testing.T) {
	if Bind("count", TextBind)["data-bind"] != "text:count" {
		t.Fatal("Bind failed")
	}
	if BindWithAttr("v", "placeholder", AttrBind)["data-bind"] != "attr:v:placeholder" {
		t.Fatal("BindWithAttr failed")
	}
	if BindWithTransform("price", TextBind, "fmt")["data-bind"] != "text:price:fmt" {
		t.Fatal("BindWithTransform failed")
	}
	two := TwoWayBind("name")
	if two["data-bind"] != "value:name" || two["data-bind-two"] != "true" || two["data-sync"] != "input" {
		t.Fatal("TwoWayBind failed")
	}
	if ClassBinding("isActive", "active")["data-bind-class"] != "active:isActive" {
		t.Fatal("ClassBinding failed")
	}
	classes := ClassBindings(map[string]string{"active": "isActive", "disabled": "isDisabled"})["data-bind-class"].(string)
	if !(strings.Contains(classes, "active:isActive") && strings.Contains(classes, "disabled:isDisabled")) {
		t.Fatalf("ClassBindings failed: %q", classes)
	}
	if StyleBinding("color", "textColor")["data-bind-style"] != "color:textColor" {
		t.Fatal("StyleBinding failed")
	}
	if ShowBinding("visible")["data-bind-show"] != "visible" || IfBinding("enabled")["data-bind-if"] != "enabled" {
		t.Fatal("ShowBinding/IfBinding failed")
	}
	if ListBinding("items", "item")["data-bind-list"] != "items" || ListBinding("items", "item")["data-item-name"] != "item" {
		t.Fatal("ListBinding failed")
	}
	listWithKey := ListBindingWithKey("items", "item", "id")
	if listWithKey["data-bind-list"] != "items" || listWithKey["data-item-key"] != "id" {
		t.Fatal("ListBindingWithKey failed")
	}
	if AttrBinding("href", "url")["data-bind-attr"] != "href:url" {
		t.Fatal("AttrBinding failed")
	}
	attrs := AttrBindings(map[string]string{"href": "url", "title": "label"})["data-bind-attr"].(string)
	if !(strings.Contains(attrs, "href:url") && strings.Contains(attrs, "title:label")) {
		t.Fatalf("AttrBindings failed: %q", attrs)
	}
	if PropBinding("disabled", "isDisabled")["data-bind-prop"] != "disabled:isDisabled" {
		t.Fatal("PropBinding failed")
	}

	if Text("msg")["data-bind"] != "text:msg" || HTML("raw")["data-bind"] != "html:raw" ||
		Value("name")["data-bind"] != "value:name" || Checked("ok")["data-bind"] != "checked:ok" {
		t.Fatal("Text/HTML/Value/Checked failed")
	}
}

func TestComponentState(t *testing.T) {
	cs := NewComponentState("cmp-1")
	if cs.ID != "cmp-1" || cs.State == nil || cs.Bindings == nil {
		t.Fatal("NewComponentState failed")
	}

	r := state.NewRune[any]("x")
	if cs.AddRune("name", r) != cs {
		t.Fatal("AddRune should return receiver")
	}
	gotRune, ok := cs.GetRune("name")
	if !ok || gotRune != r {
		t.Fatal("GetRune did not return inserted rune")
	}
	if _, ok = cs.GetRune("missing"); ok {
		t.Fatal("GetRune should return false for missing key")
	}

	cs.AddBinding("name", Binding{Type: TextBind})
	cs.AddBinding("price", Binding{Type: AttrBind, Attr: "data-price"})
	cs.AddBinding("count", Binding{Type: ValueBind, Transform: "toInt"})
	cs.AddBinding("status", Binding{Type: ClassBind, Attr: "badge", Transform: "toClass"})
	rendered := cs.RenderBindings()
	if rendered["data-component"] != "cmp-1" {
		t.Fatal("RenderBindings missing data-component")
	}
	if rendered["data-bind-text"] != "name" {
		t.Fatal("RenderBindings default case failed")
	}
	if rendered["data-bind-attr"] != "data-price:price:" {
		t.Fatal("RenderBindings attr case failed")
	}
	if rendered["data-bind-value"] != "count:toInt" {
		t.Fatal("RenderBindings transform case failed")
	}
	if rendered["data-bind-class"] != "badge:status:toClass" {
		t.Fatal("RenderBindings attr+transform case failed")
	}

	jsonState, err := cs.ToJSON()
	if err != nil || !strings.Contains(jsonState, `"name":"x"`) {
		t.Fatalf("ToJSON failed: %v %q", err, jsonState)
	}
	stateAttrs := cs.StateAttrs()
	if stateAttrs["data-component"] != "cmp-1" {
		t.Fatal("StateAttrs failed")
	}

	out := renderComponent(t, cs.InitScript(), WithNonce(context.Background(), "nonce-bind"))
	if !strings.Contains(out, `nonce="nonce-bind"`) || !strings.Contains(out, `data-component-init="cmp-1"`) ||
		!strings.Contains(out, `window.__GOSPA_STATE__=`) {
		t.Fatalf("InitScript output invalid: %s", out)
	}
}

func TestUnsafeHelpers(t *testing.T) {
	if UnsafeHTML("<b>x</b>") != "<b>x</b>" {
		t.Fatal("UnsafeHTML failed")
	}
	if UnsafeAttr(`x"y`) != `x"y` {
		t.Fatal("UnsafeAttr failed")
	}
}

