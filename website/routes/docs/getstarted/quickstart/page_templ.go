package quickstart

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import "github.com/aydenstechdungeon/gospa/website/components"

func Page() templ.Component {
	return templruntime.GeneratedTemplate(func(input templruntime.GeneratedComponentInput) error {
		w, ctx := input.Writer, input.Context
		if err := ctx.Err(); err != nil {
			return err
		}
		buf, isBuf := templruntime.GetBuffer(w)
		if !isBuf {
			defer func() { _ = templruntime.ReleaseBuffer(buf) }()
		}
		ctx = templ.InitializeContext(ctx)
		ctx = templ.ClearChildren(ctx)

		if err := templruntime.WriteString(buf, 1, `<div class="space-y-8"><header><h1 class="text-4xl font-bold tracking-tight mb-4">Quick Start</h1><p class="text-xl text-[var(--text-secondary)]">Build your first reactive GoSPA application in minutes.</p></header><section><h2 id="create-project" class="text-2xl font-bold mb-4">1. Create a Project</h2><p class="text-[var(--text-secondary)] mb-4">Use the CLI to scaffold a new project, install dependencies, and verify the workspace before you start coding:</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa create my-app\ncd my-app\ngo mod tidy\ngospa doctor", "bash", "Terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 2, `</section><section class="rounded-[2rem] border border-[var(--border)] bg-[var(--bg-secondary)] p-6 md:p-8"><div class="flex flex-wrap items-center gap-3 mb-4"><div class="px-3 py-1 rounded-full text-xs font-bold uppercase tracking-[0.28em] bg-[var(--accent-primary)]/10 text-[var(--accent-primary)]">Recommended Baseline</div><div class="text-sm text-[var(--text-secondary)]">Use the presets to keep dev and production stories aligned.</div></div>`); err != nil {
			return err
		}
		if err := components.CodeBlock("config := gospa.DefaultConfig()\nconfig.AppName = \"my-app\"\n\n// for production:\n// config := gospa.ProductionConfig()", "go", "main.go").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 3, `</section><section><h2 id="dev-server" class="text-2xl font-bold mb-4">2. Start Dev Server</h2><p class="text-[var(--text-secondary)] mb-4">Launch the development environment with hot reloading:</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa dev", "bash", "Terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 4, `</section><section><h2 id="create-route" class="text-2xl font-bold mb-4">3. Create a Route</h2><p class="text-[var(--text-secondary)] mb-4">Add a new file <code class="bg-[var(--bg-tertiary)] px-1.5 py-0.5 rounded">routes/hello.templ</code>:</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock(`package routes

templ HelloPage() {
    <div class="p-8">
        <h1 class="text-3xl font-bold">Hello from GoSPA!</h1>
        <p>This page was created in seconds.</p>
    </div>
}`, "go", "routes/hello.templ").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 5, `</section><section><h2 id="add-reactivity" class="text-2xl font-bold mb-4">4. Add Reactivity</h2><p class="text-[var(--text-secondary)] mb-4">Make it interactive with reactive state:</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock(`package routes

import (
    "github.com/aydenstechdungeon/gospa/state"
)

templ HelloPage() {
    <div data-gospa-component="hello" class="p-8">
        <h1 class="text-3xl font-bold">Counter: <span data-bind="count">0</span></h1>
        <button 
            class="px-4 py-2 bg-blue-500 text-white rounded mt-4"
            data-on="click:increment"
        >
            Increment
        </button>
    </div>
}

func HelloState() *state.StateMap {
    sm := state.NewStateMap()
    sm.AddAny("count", 0)
    return sm
}`, "go", "routes/hello.templ").Render(ctx, buf); err != nil {
			return err
		}
		return templruntime.WriteString(buf, 6, `</section><div class="flex justify-between mt-16"><a href="/docs/getstarted/installation" class="group flex items-center gap-3 px-6 py-3 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] hover:border-[var(--accent-primary)]/50 transition-all"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-[var(--text-muted)] group-hover:text-[var(--accent-primary)] transition-colors"><path d="M19 12H5"></path><path d="m12 19-7-7 7-7"></path></svg><div><div class="text-[var(--text-muted)] text-xs uppercase tracking-widest font-bold mb-1">Previous</div><div class="font-bold group-hover:text-[var(--accent-primary)] transition-colors">Installation</div></div></a><a href="/docs/getstarted/structure" class="group flex items-center gap-3 px-6 py-3 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] hover:border-[var(--accent-primary)]/50 transition-all text-right"><div><div class="text-[var(--text-muted)] text-xs uppercase tracking-widest font-bold mb-1">Next</div><div class="font-bold group-hover:text-[var(--accent-primary)] transition-colors">Project Structure</div></div><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-[var(--text-muted)] group-hover:text-[var(--accent-primary)] transition-colors"><path d="M5 12h14"></path><path d="m12 5 7 7-7 7"></path></svg></a></div></div>`)
	})
}

var _ = templruntime.GeneratedTemplate
