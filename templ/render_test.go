package templ

import (
	"context"
	"strconv"
	"strings"
	"testing"

	ahtempl "github.com/a-h/templ"
)

func renderComponent(ctx context.Context, t *testing.T, c ahtempl.Component) string {
	t.Helper()
	var b strings.Builder
	if err := c.Render(ctx, &b); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return b.String()
}

func assertContainsAll(t *testing.T, got string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(got, part) {
			t.Fatalf("expected output to contain %q, got: %s", part, got)
		}
	}
}

func TestRenderBasics(t *testing.T) {
	ctx := WithNonce(context.Background(), "nonce-123")

	tests := []struct {
		name      string
		component ahtempl.Component
		contains  []string
	}{
		{
			name:      "runtime script with nonce and escaped src",
			component: RuntimeScript(`/app.js?x=<tag>`),
			contains:  []string{`<script src="/app.js?x=&lt;tag&gt;" type="module" nonce="nonce-123"></script>`},
		},
		{
			name:      "inline runtime script with nonce",
			component: RuntimeScriptInline(`window.x=1;`),
			contains:  []string{`<script nonce="nonce-123">window.x=1;</script>`},
		},
		{
			name:      "css link escapes href",
			component: CSS(`/style.css?v=<bad>`),
			contains:  []string{`<link rel="stylesheet" href="/style.css?v=&lt;bad&gt;">`},
		},
		{
			name:      "inline css with nonce",
			component: CSSInline(`body{margin:0}`),
			contains:  []string{`<style nonce="nonce-123">body{margin:0}</style>`},
		},
		{
			name:      "meta escapes values",
			component: Meta(`desc`, `x<y`),
			contains:  []string{`<meta name="desc" content="x&lt;y">`},
		},
		{
			name:      "meta property escapes values",
			component: MetaProperty(`og:title`, `a<b`),
			contains:  []string{`<meta property="og:title" content="a&lt;b">`},
		},
		{
			name:      "title escapes value",
			component: Title(`hello <world>`),
			contains:  []string{`<title>hello &lt;world&gt;</title>`},
		},
		{
			name:      "favicon escapes href",
			component: Favicon(`/favicon.ico?x=<1>`),
			contains:  []string{`<link rel="icon" href="/favicon.ico?x=&lt;1&gt;">`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderComponent(ctx, t, tt.component)
			assertContainsAll(t, out, tt.contains...)
		})
	}
}

func TestHeadAndPages(t *testing.T) {
	ctx := WithNonce(context.Background(), "n-page")
	head := Head(Title("My App"), Meta("description", "app"))
	body := Fragment(TextContent("hello"))

	html := renderComponent(ctx, t, HTMLPage("en", head, body))
	assertContainsAll(t, html,
		`<!DOCTYPE html><html lang="en"><head>`,
		`<title>My App</title>`,
		`<meta name="description" content="app">`,
		`</head><body>hello</body></html>`,
	)

	empty := renderComponent(ctx, t, HTMLPage("fr", nil, nil))
	assertContainsAll(t, empty, `<!DOCTYPE html><html lang="fr"><head></head><body></body></html>`)
}

func TestSPAPage(t *testing.T) {
	ctx := WithNonce(context.Background(), "nonce-spa")
	out := renderComponent(ctx, t, SPAPage(SPAConfig{
		Lang:        "en",
		Title:       `GoSPA <Docs>`,
		Meta:        []MetaTag{{Name: "description", Content: "site"}},
		Stylesheets: []string{"/a.css", "/b.css"},
		Head:        Meta("robots", "index,follow"),
		Body:        TextContent("Body"),
		RootID:      "app",
		RuntimeSrc:  "/runtime.js",
		AutoInit:    true,
	}))

	assertContainsAll(t, out,
		`<!DOCTYPE html><html lang="en"><head>`,
		`<meta charset="UTF-8">`,
		`<meta name="viewport" content="width=device-width, initial-scale=1.0">`,
		`<title>GoSPA &lt;Docs&gt;</title>`,
		`<meta name="description" content="site">`,
		`<link rel="stylesheet" href="/a.css">`,
		`<link rel="stylesheet" href="/b.css">`,
		`<meta name="robots" content="index,follow">`,
		`</head><body><div id="app" data-gospa-root>Body</div>`,
		`<script src="/runtime.js" type="module" nonce="nonce-spa"></script>`,
		`<script nonce="nonce-spa" data-gospa-auto></script>`,
		`</body></html>`,
	)
}

func TestRawHTMLAndTextContent(t *testing.T) {
	ctx := context.Background()
	assertContainsAll(t, renderComponent(ctx, t, Raw(`<b>x</b>`)), `<b>x</b>`)
	assertContainsAll(t, renderComponent(ctx, t, HTMLContent(`<em>safe</em>`)), `<em>safe</em>`)
	assertContainsAll(t, renderComponent(ctx, t, TextContent(`<em>escape</em>`)), `&lt;em&gt;escape&lt;/em&gt;`)
}

func TestAttributeHelpers(t *testing.T) {
	merged := Attrs(ahtempl.Attributes{"id": "x"}, ahtempl.Attributes{"class": "y"}, ahtempl.Attributes{"id": "z"})
	if merged["id"] != "z" || merged["class"] != "y" {
		t.Fatalf("unexpected merged attrs: %#v", merged)
	}

	if Class("a", "b")["class"] != "a b" {
		t.Fatalf("unexpected class value")
	}
	classIf := ClassIf(map[string]bool{"active": true, "hidden": false})
	if classIf["class"] != "active" {
		t.Fatalf("unexpected class-if value: %q", classIf["class"])
	}

	style := Style(map[string]string{"color": "red", "display": "block"})["style"].(string)
	assertContainsAll(t, style, "color: red", "display: block")

	data := DataAttrs(map[string]any{"id": 1, "kind": "x"})
	if data["data-id"] != 1 || data["data-kind"] != "x" {
		t.Fatalf("unexpected data attrs: %#v", data)
	}

	if ID("i")["id"] != "i" || Name("n")["name"] != "n" || Type("text")["type"] != "text" {
		t.Fatalf("basic attr helpers failed")
	}
	if ValueAttr("v")["value"] != "v" || Placeholder("p")["placeholder"] != "p" {
		t.Fatalf("value/placeholder helpers failed")
	}
	if Href("/x")["href"] != "/x" || Src("/img")["src"] != "/img" || Alt("a")["alt"] != "a" {
		t.Fatalf("href/src/alt helpers failed")
	}
	if Target("_blank")["target"] != "_blank" || Rel("noopener")["rel"] != "noopener" {
		t.Fatalf("target/rel helpers failed")
	}
	if Aria("label", "ok")["aria-label"] != "ok" || Role("button")["role"] != "button" || TabIndex(3)["tabindex"] != 3 {
		t.Fatalf("aria/role/tabindex helpers failed")
	}

	if Disabled(true)["disabled"] != "" || Readonly(true)["readonly"] != "" || Required(true)["required"] != "" {
		t.Fatalf("expected boolean attrs for true")
	}
	if CheckedAttr(true)["checked"] != "" || Selected(true)["selected"] != "" || Hidden(true)["hidden"] != "" {
		t.Fatalf("expected checked/selected/hidden attrs for true")
	}

	if Disabled(false) != nil || Readonly(false) != nil || Required(false) != nil ||
		CheckedAttr(false) != nil || Selected(false) != nil || Hidden(false) != nil {
		t.Fatalf("expected nil attrs for false")
	}
}

func TestControlFlowHelpers(t *testing.T) {
	ctx := context.Background()

	frag := Fragment(TextContent("a"), TextContent("b"))
	assertContainsAll(t, renderComponent(ctx, t, frag), "ab")
	assertContainsAll(t, renderComponent(ctx, t, Empty()), "")

	assertContainsAll(t, renderComponent(ctx, t, When(true, TextContent("yes"))), "yes")
	assertContainsAll(t, renderComponent(ctx, t, When(false, TextContent("no"))), "")
	assertContainsAll(t, renderComponent(ctx, t, WhenElse(true, TextContent("t"), TextContent("f"))), "t")
	assertContainsAll(t, renderComponent(ctx, t, WhenElse(false, TextContent("t"), TextContent("f"))), "f")

	forOut := renderComponent(ctx, t, For([]int{1, 2, 3}, func(v, _ int) ahtempl.Component {
		return TextContent(strconv.Itoa(v))
	}))
	assertContainsAll(t, forOut, "123")

	keyed := renderComponent(ctx, t, ForKey([]string{"a", "b"}, func(s string) string { return "k-" + s }, func(s string, _ int) ahtempl.Component {
		return TextContent(s)
	}))
	assertContainsAll(t, keyed, `<template data-key="k-a">a</template>`, `<template data-key="k-b">b</template>`)

	sw := renderComponent(ctx, t, Switch(
		Case(false, TextContent("no")),
		Case(true, TextContent("yes")),
		Default(TextContent("default")),
	))
	assertContainsAll(t, sw, "yes")
}

func TestHeadManagerAndHeadHelpers(t *testing.T) {
	ctx := WithNonce(context.Background(), "nonce-head")
	h := NewHeadManager().
		SetHeadTitle("Page").
		AddHeadMeta("description", "desc").
		AddHeadMetaProperty("og:title", "OG").
		AddHeadLink("preload", "/font.woff2", map[string]string{"as": "font"}).
		AddHeadScript("/app.js", true, true).
		AddHeadInlineScript("window.x=1").
		AddHeadStyle("/app.css").
		AddHeadInlineStyle("body{margin:0}").
		AddHeadElement(HeadElement{Tag: "custom-tag", Content: "C", Key: "custom", Priority: 1})

	out := renderComponent(ctx, t, h.Render())
	assertContainsAll(t, out,
		`<title data-gospa-head="title">Page</title>`,
		`name="description"`,
		`property="og:title"`,
		`rel="preload"`,
		`as="font"`,
		`src="/app.js"`,
		`async`,
		`defer`,
		`nonce="nonce-head"`,
		`<style`,
		`body{margin:0}`,
		`<custom-tag data-gospa-head="custom">C</custom-tag>`,
	)

	// Priority check: title (100) must render before script (10).
	titleIdx := strings.Index(out, `<title data-gospa-head="title">Page</title>`)
	scriptIdx := strings.Index(out, `<script`)
	if titleIdx == -1 || scriptIdx == -1 || titleIdx > scriptIdx {
		t.Fatalf("expected title to appear before script, output: %s", out)
	}

	if min(1, 2) != 1 || min(5, 3) != 3 {
		t.Fatalf("min helper failed")
	}

	headOut := renderComponent(context.Background(), t, Fragment(
		HeadTitle("T"),
		HeadMeta("description", "d"),
		HeadMetaProp("og:site_name", "gospa"),
		HeadLink("canonical", "/home", map[string]string{"hreflang": "en"}),
		HeadScript("/bundle.js", true, true),
		HeadStyle("/style.css"),
	))
	assertContainsAll(t, headOut,
		`<title data-gospa-head="title">T</title>`,
		`data-gospa-head="meta-description"`,
		`data-gospa-head="meta-prop-og:site_name"`,
		`rel="canonical"`,
		`href="/home"`,
		`hreflang="en"`,
		`src="/bundle.js"`,
		`async`,
		`defer`,
		`data-gospa-head="style-/style.css"`,
	)
}
