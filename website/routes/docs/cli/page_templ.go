package cli

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

		if err := templruntime.WriteString(buf, 1, `<div class="space-y-12"><header><h1 class="text-4xl font-bold tracking-tight mb-4 text-transparent bg-clip-text bg-gradient-to-r from-[var(--accent-primary)] to-[var(--accent-secondary)]">CLI Reference</h1><p class="text-xl text-[var(--text-secondary)] leading-relaxed">The current GoSPA CLI centers on six supported workflows: scaffold, inspect, develop, build, generate, and clean.</p></header><section class="grid gap-6 lg:grid-cols-[1.3fr_0.7fr] items-start"><div class="rounded-[2rem] border border-[var(--border)] bg-[var(--bg-secondary)] p-6 md:p-8 shadow-[0_20px_60px_rgba(0,0,0,0.12)]"><div class="flex items-center gap-3 mb-5"><div class="px-3 py-1 rounded-full text-xs font-bold uppercase tracking-[0.28em] bg-[var(--accent-primary)]/10 text-[var(--accent-primary)]">Recommended Flow</div></div>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa create myapp\ncd myapp\ngo mod tidy\ngospa doctor\ngospa dev", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 2, `</div><div class="space-y-4"><div class="rounded-2xl border border-[var(--border)] bg-[var(--bg-secondary)] p-5"><h2 class="font-bold mb-2">What doctor gives you</h2><p class="text-sm text-[var(--text-secondary)] leading-relaxed">Before you start the server, validate Go/Bun availability, project layout, and runtime entrypoints from the same CLI users already installed.</p></div><div class="rounded-2xl border border-[var(--border)] bg-[var(--bg-secondary)] p-5"><h2 class="font-bold mb-2">What build now reports</h2><p class="text-sm text-[var(--text-secondary)] leading-relaxed">Production builds now surface the Bun executable, runtime artifact path, Go binary path and size, plus copied/compressed asset counts.</p></div></div></section><section class="space-y-8"><div><h2 class="text-2xl font-bold mb-4 border-b border-[var(--border)] pb-2 italic mono">gospa create</h2><p class="text-[var(--text-secondary)] mb-4">Create a new GoSPA project scaffold.</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa create myapp", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 3, `</div><div><h2 class="text-2xl font-bold mb-4 border-b border-[var(--border)] pb-2 italic mono">gospa doctor</h2><p class="text-[var(--text-secondary)] mb-4">Check local tooling and project layout before you start development or cut a release.</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa doctor\ngospa doctor --routes-dir ./routes", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 4, `</div><div><h2 class="text-2xl font-bold mb-4 border-b border-[var(--border)] pb-2 italic mono">gospa dev</h2><p class="text-[var(--text-secondary)] mb-4">Start the local development workflow with file watching and server restarts.</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa dev\ngospa dev --port 8080 --host 0.0.0.0\ngospa dev --routes-dir ./routes", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 5, `</div><div><h2 class="text-2xl font-bold mb-4 border-b border-[var(--border)] pb-2 italic mono">gospa build</h2><p class="text-[var(--text-secondary)] mb-4">Create a production build with Go plus Bun-based client runtime steps where applicable.</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa build\ngospa build -o ./dist --platform linux --arch amd64\ngospa build --minify=false --compress=false", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 6, `</div><div><h2 class="text-2xl font-bold mb-4 border-b border-[var(--border)] pb-2 italic mono">gospa generate</h2><p class="text-[var(--text-secondary)] mb-4">Generate Go routes plus TypeScript types and route metadata.</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa generate\ngospa generate -o ./generated --input-dir .", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 7, `</div><div><h2 class="text-2xl font-bold mb-4 border-b border-[var(--border)] pb-2 italic mono">gospa clean</h2><p class="text-[var(--text-secondary)] mb-4">Remove generated and build artifacts such as dist, node_modules, and templ-generated files.</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa clean", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		if err := templruntime.WriteString(buf, 8, `</div><div><h2 class="text-2xl font-bold mb-4 border-b border-[var(--border)] pb-2 italic mono">gospa version</h2><p class="text-[var(--text-secondary)] mb-4">Print the installed GoSPA version.</p>`); err != nil {
			return err
		}
		if err := components.CodeBlock("gospa version", "bash", "terminal").Render(ctx, buf); err != nil {
			return err
		}
		return templruntime.WriteString(buf, 9, `</div></section></div>`)
	})
}

var _ = templruntime.GeneratedTemplate
