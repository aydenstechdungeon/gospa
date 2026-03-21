# GoSPA Project Tightening & Polish Proposal

Generated: 2026-03-21

## Goal

Tighten the full GoSPA surface area so the project feels more internally consistent, easier to trust in production, and sharper for first-time adopters. This proposal covers the main Go framework, plugins, CLI, browser runtime, website, examples, testing, and release discipline.

## Executive Summary

GoSPA already has strong raw ingredients: a reactive Go core, a compact client runtime, a plugin model, a CLI, a docs website, and runnable examples. The biggest opportunity is no longer feature breadth; it is convergence. Several parts of the repo appear to have moved at different speeds, which creates drift between the public story, generated scaffolds, examples, runtime options, and operational guarantees.

The tightening pass should focus on four outcomes:

1. **Consistency** — the README, CLI scaffolds, examples, website docs, and public config fields should describe the same product.
2. **Operational safety** — production defaults, plugin acquisition, cache behavior, and remote/websocket behavior should be easy to reason about.
3. **Developer trust** — builds, examples, generated apps, and docs examples should all compile and run on the first attempt.
4. **Presentation polish** — the website and examples should showcase the framework’s strongest patterns, not just prove functionality.

## What Looks Most Urgent Right Now

### 1. Product-surface drift
The project currently exposes multiple parallel narratives:
- the root README presents a tight framework story,
- `gospa.Config` contains a much broader and more advanced surface area,
- the CLI scaffold creates a simpler starter,
- the website introduces the framework in yet another voice,
- and some examples appear to lag behind the current config shape.

This is the single largest quality drag because it affects users before they write any application code.

### 2. Example and scaffold credibility
Examples and scaffolds are the trust anchors for an early-stage framework. If they drift, users assume the framework itself is unstable.

A concrete example: `examples/form-remote/main.go` still sets `WebSocket: true`, while the current config uses `EnableWebSocket`. That implies at least one example is stale and likely does not compile cleanly against the current public API. This should be treated as a release-blocking polish issue, not a minor docs issue.

### 3. Too much “possible,” not enough “blessed”
GoSPA exposes many knobs for routing, rendering, runtime, hydration, storage, transport, and navigation. That power is valuable, but the project needs a more explicit distinction between:
- **stable/default paths**,
- **advanced but supported paths**,
- **experimental/planned paths**.

Without that, the framework feels larger than it is mature.

## Repo-Wide Tightening Plan

## A. Core Go Framework (`gospa`, `state`, `templ`, `fiber`, `component`, `store`)

### Tightening goals
- Make defaults opinionated and obviously production-safe.
- Reduce ambiguity around which config fields are stable, optional, planned, or environment-specific.
- Strengthen package boundaries so transport, rendering, and reactive state remain conceptually distinct.

### Recommendations

#### A1. Split `Config` into clearer groups in docs and code comments
`gospa.Config` is functionally rich, but it reads like a long accumulation of capabilities. Reframe it around domains:
- App / paths
- Rendering
- Runtime & hydration
- WebSocket / remote actions
- Security
- Storage / prefork
- Navigation optimizations

This can be done incrementally without breaking API compatibility:
- keep the flat struct for now,
- add comment headers and table-driven docs,
- later introduce helper constructors or nested config structs for new APIs only.

#### A2. Define stability labels for config fields
For each major option, annotate one of:
- `Stable`
- `Advanced`
- `Experimental`
- `Internal/subject to change`

This especially matters for navigation optimization fields and serialization/runtime variants. A framework earns trust faster when the public surface advertises certainty levels.

#### A3. Add a “golden defaults” profile
Provide named constructors such as:
- `gospa.DefaultConfig()` — current broad default
- `gospa.ProductionConfig()` — secure, documented production defaults
- `gospa.MinimalConfig()` — smallest supported baseline

This gives the project a blessed operating model rather than a bag of switches.

#### A4. Formalize cache and invalidation guarantees
The rendering/cache story should specify:
- what is cached,
- cache keys,
- invalidation triggers,
- TTL semantics,
- memory ceilings,
- concurrency behavior.

If GoSPA supports SSR, SSG, ISR, and PPR, the docs should expose one shared cache model instead of letting each mode feel bespoke.

#### A5. Tighten observability hooks
Add or standardize:
- structured log field names,
- lifecycle events around startup/build/render/cache/websocket/remote actions,
- optional metrics hooks or a simple adapter interface.

Frameworks become much easier to adopt in production when operators can answer “what is it doing?” without patching internals.

## B. Plugin System (`plugin/*`)

### Tightening goals
- Make plugin installation and execution safer.
- Clarify plugin lifecycle and compatibility guarantees.
- Reduce ambiguity around remote plugin loading.

### Recommendations

#### B1. Introduce plugin trust levels
Separate plugin workflows into:
- **local plugins**,
- **pinned remote plugins**,
- **floating remote plugins**.

Remote plugin loading from GitHub should strongly prefer pinned tags or commits and make the resolved ref prominent in user output.

#### B2. Add plugin manifest versioning
The `plugin.json` metadata should include a manifest schema version and a declared GoSPA compatibility range. This reduces breakage when the framework evolves faster than plugin authors update.

#### B3. Make installation/build/load phases explicit
The system should distinguish:
- fetch,
- verify,
- build,
- activate.

Right now remote loading can feel like one operation. Splitting those phases would improve both UX and security review.

#### B4. Add failure isolation guidance
Document whether plugin hook failures are:
- fatal,
- warning-only,
- retried,
- skipped.

Also add timeouts or context propagation around plugin hooks where possible. A plugin system feels polished when bad plugins degrade gracefully.

## C. CLI (`cli/*`, `cmd/*`)

### Tightening goals
- Make generated projects modern, minimal, and aligned with current framework APIs.
- Ensure CLI output is actionable and consistent.
- Turn the CLI into the canonical onboarding path.

### Recommendations

#### C1. Make `gospa create` the canonical source of truth
The scaffold should always reflect current best practice:
- current config field names,
- current docs URL/domain,
- current runtime assumptions,
- current security defaults.

Add an internal rule: any public config rename or starter-pattern change must update the scaffold in the same PR.

#### C2. Add scaffold variants
Provide a small set of first-class templates:
- `minimal`
- `docs-site`
- `realtime`
- `form-actions`

This is better than a one-size-fits-all starter because it lets examples and scaffolds reinforce each other.

#### C3. Improve build diagnostics
`gospa build` should report:
- which phases ran,
- which were skipped,
- where Bun was expected/found,
- produced artifacts and sizes,
- whether static compression succeeded.

A polished build command behaves like a release checklist, not just a shell wrapper.

#### C4. Add a `gospa doctor`
A dedicated doctor command should validate:
- Go version,
- Bun availability,
- Templ availability if required,
- route generation health,
- example/build prerequisites,
- conflicting config settings.

This would eliminate a lot of “why does this not run?” friction.

## D. Browser Runtime / Client Package (`client/*`, `embed/*`)

### Tightening goals
- Reduce runtime mode sprawl.
- Define a single mental model for hydration, navigation, and state sync.
- Keep Bun/TypeScript tooling disciplined and reproducible.

### Recommendations

#### D1. Declare one default runtime and a small variant matrix
The runtime family currently communicates power, but also complexity. Publish a compact matrix showing:
- intended use case,
- security model,
- sanitizer behavior,
- feature tradeoffs,
- bundle size target.

The default runtime should be the one used across docs, website, scaffolds, and examples unless a page explicitly teaches tradeoffs.

#### D2. Tighten package scripts and artifact ownership
Define which files are source of truth versus generated artifacts:
- `client/src/*` are authored,
- `embed/*.js` are generated,
- generation command is Bun-only,
- CI should fail on dirty generated output.

That removes ambiguity for contributors and avoids stale embed bundles.

#### D3. Add runtime contract tests
Create contract tests for:
- navigation lifecycle,
- hydration modes,
- websocket reconnect behavior,
- remote action error handling,
- sanitizer/no-sanitizer behavior,
- runtime variant parity where required.

This is the highest leverage place to prevent regressions in user-visible behavior.

#### D4. Publish browser support policy
Explicitly document support level for:
- modern evergreen browsers,
- `DecompressionStream`,
- `requestIdleCallback`,
- View Transitions API,
- service-worker-assisted features.

A polished runtime tells users what degrades and how.

## E. Website & Docs (`website/*`, `docs/*`, root `README.md`)

### Tightening goals
- Make the docs site feel like the same product as the codebase.
- Reduce copy drift and feature ambiguity.
- Upgrade the visual presentation from “good project site” to “authoritative framework home.”

### Recommendations

#### E1. Create one messaging spine
Unify the top-level language across README, website landing page, docs intro, and CLI scaffold copy:
- what GoSPA is,
- who it is for,
- why it is different,
- what is stable today.

Users should not encounter four different product definitions depending on entry point.

#### E2. Add a status model to docs
Every major capability page should include a small status badge:
- Stable
- Alpha
- Experimental
- Planned

That makes the project feel honest and well-managed, which is especially valuable for a framework still maturing.

#### E3. Tighten the landing page proof points
The website should prove three things quickly:
- **authoring model** (Go-first reactivity),
- **performance posture** (small runtime, efficient transport),
- **deployment story** (single binary, optional client runtime, practical production path).

The current site already has strong visual energy; the next polish step is sharper proof density and fewer generic claims.

#### E4. Introduce architecture diagrams that match implementation
Use one clean architecture diagram shared between README/docs/website. It should reflect actual packages and data flow, not aspirational structure from older plans.

#### E5. Add upgrade notes and compatibility pages
Framework users need:
- version-to-version upgrade notes,
- breaking change summaries,
- compatibility notes for Go, Bun, Fiber, and Templ.

That would dramatically improve perceived project maturity.

## F. Examples (`examples/*`)

### Tightening goals
- Ensure every example compiles and teaches one clear concept.
- Remove drift between examples and framework APIs.
- Make examples feel curated rather than accumulated.

### Recommendations

#### F1. Classify examples by purpose
Each example should declare:
- concept taught,
- required features,
- expected commands,
- whether it is starter-grade or advanced.

#### F2. Enforce example health in CI
Every example should be checked for:
- `go build`,
- route generation if needed,
- template generation if needed,
- formatting/lint cleanliness where practical.

If an example breaks, it should block release.

#### F3. Reduce noise in starter examples
The best examples are short and opinionated. Keep “counter” and “form actions” extremely clean. Move edge-case-heavy examples into an “advanced” group.

#### F4. Align examples with scaffold variants
If the CLI offers `minimal`, `realtime`, and `form-actions`, then examples should mirror those patterns exactly. This creates a strong learning ladder:
- README quickstart,
- generated scaffold,
- matching example,
- deeper docs.

## G. Testing, QA, and Release Discipline

### Tightening goals
- Convert repo breadth into reliable release quality.
- Make regressions obvious before users hit them.

### Recommendations

#### G1. Add a release matrix
Minimum release gates should include:
- root package tests,
- client typecheck/tests via Bun,
- website build checks,
- example compilation checks,
- generated artifact freshness,
- smoke test for `gospa create` + `gospa build`.

#### G2. Add snapshot/golden coverage for generated output
Useful candidates:
- generated scaffold files,
- embedded runtime file list,
- docs navigation/sidebar generation,
- selected SSR output for stable examples.

#### G3. Publish a “definition of done” for new features
Any new feature touching framework behavior should update, at minimum:
- code,
- docs,
- scaffold or example if relevant,
- tests,
- changelog.

This is how the repo stops drifting again after the tightening pass.

## H. Design & UX Polish Priorities

### Tightening goals
- Make the project feel deliberate, not merely feature-rich.
- Improve first-run and first-read clarity.

### Recommendations

#### H1. Sharpen the visual system on the website
Keep the current modern look, but enforce stronger consistency in:
- spacing scale,
- corner radius usage,
- panel density,
- typography hierarchy,
- CTA treatment.

The visual language should feel more like a cohesive design system and less like individually polished sections.

#### H2. Turn examples into teaching artifacts
Each example should include a short README or page banner explaining:
- what to inspect,
- what to edit first,
- what concept the user should leave understanding.

#### H3. Improve CLI copy tone
CLI output should be brief, confident, and uniform. Success, warning, and next-step messages should all follow the same voice.

## Recommended Order of Execution

### Phase 1 — Trust repair
1. Fix stale examples and scaffold drift.
2. Align README, website intro, and CLI-generated copy.
3. Add CI checks for examples, generated artifacts, and Bun client validation.

### Phase 2 — Surface-area clarification
4. Label config/runtime/plugin stability levels.
5. Document runtime variants and browser support.
6. Add plugin compatibility/versioning rules.

### Phase 3 — Production polish
7. Add observability guidance and production config profile.
8. Improve build diagnostics and add `gospa doctor`.
9. Tighten release gating and upgrade notes.

### Phase 4 — Presentation upgrade
10. Refine website proof hierarchy and architecture visuals.
11. Curate examples into a more intentional learning path.
12. Standardize docs status badges and page structure.

## Concrete Near-Term Backlog

### Must do next
- Fix stale example API usage.
- Audit all examples against the current public config.
- Audit `gospa create` output against the current docs and website domain.
- Add a repo-level validation script that runs Go tests, Bun typecheck/tests, and example builds.
- Define a single “recommended production setup” page.

### Should do soon
- Add plugin manifest schema version and compatibility metadata.
- Add runtime variant matrix and browser support page.
- Add scaffold variants.
- Add release checklist automation.

### Nice to have
- `gospa doctor`
- architecture diagram unification
- richer website proof sections and example callouts
- metrics hooks / tracing adapters

## Success Criteria

This tightening pass is successful when:
- the README, website, CLI, and examples all teach the same defaults,
- every bundled example builds and passes smoke checks,
- runtime and plugin variants have explicit status and compatibility guidance,
- contributors know which files are generated and how to refresh them,
- and a first-time user can go from install to confidence without hitting contradictory guidance.

## Notes From Current Repo Review

The recommendations above were shaped by a quick repo scan and are grounded in current project structure and visible drift points, including:
- rich but wide `gospa.Config` surface area,
- Bun-based client runtime packaging,
- CLI project generation as a key onboarding path,
- remote plugin loading with cached GitHub clones,
- a polished website already in place,
- and at least one stale example config usage that suggests example/API drift.

That makes the highest-value move a repo-wide convergence pass rather than another feature expansion sprint.
