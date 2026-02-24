# GoSPA Documentation Improvement Plan

This plan outlines the steps to enhance the GoSPA documentation (located in the `website/` directory) by breaking up large pages, implementing site-wide search, and improving overall readability and user experience.

## 1. Objective
Transform the GoSPA documentation from a collection of long pages into a modern, granular, and easily searchable knowledge base that feels premium and state-of-the-art.

## 2. Content Restructuring (Breaking up pages)
The current documentation contains several "mega-pages" (e.g., Getting Started) that cover too many disparate topics.

### 2.1 Audit & Splitting
- **Getting Started**: Split into:
    - `Introduction`: High-level vision and philosophy.
    - `Installation`: Binary installation, environment setup.
    - `Quick Start`: A 5-minute "Hello World" tutorial.
    - `Project Structure`: Deep dive into folders and files.
- **Routing**: Split into:
    - `Basics`: Static routes and file naming.
    - `Dynamic Routing`: Parameters and path building.
    - `Layouts`: Nested layouts and root layout.
    - `Route Groups`: Organization and authentication grouping.
- **Client Runtime**: Split into:
    - `Overview`: Initialization and lifecycle.
    - `DOM Bindings`: `data-bind`, `data-model`, `data-on`.
    - `Sanitization`: Security and HTML safety.
- **Reactive Primitives**: Create individual pages for `Rune`, `Derived`, `Effect`, and `Batching`.

### 2.2 Reorganized Sidebar
The sidebar (`website/components/sidebar.templ`) will be updated to reflect this new hierarchy, using expanded sections and clearer groupings.

## 3. Search Implementation
Implement a fast, local search to help users find specific APIs and concepts instantly.

### 3.1 Search Indexing
- Create a Go-based indexer that extracts text and metadata (title, headings, content) from the `.templ` files in `website/routes/docs/`.
- Generate a `docs_search_index.json` during the build process.

### 3.2 Search UI
- **Search Header**: Add a search input to the top navigation/sidebar with a `Cmd+K` shortcut.
- **Search Results**: A modal or dropdown that displays matching pages and specific sections with title/snippet previews.
- **Technology**: Use **Fuse.js** for client-side fuzzy searching on the generated index.

## 4. Readability & UX Enhancements
Improve the "flow" and visual comfort of reading long technical documentation.

### 4.1 Dynamic Table of Contents (ToC)
- Replace the hardcoded `getHeadlines` function in `layout.templ` with a dynamic client-side script that extracts `<h2>` and `<h3>` tags from the active page content.
- Implement an "active section" tracker that highlights the current heading in the ToC as the user scrolls.

### 4.2 Visual Callouts & Typography
- Implement standard visual callouts (Admonitions) for:
    - 💡 **Tip**: Helpful hints or shortcuts.
    - ⚠️ **Warning**: Critical security or performance caveats.
    - ℹ️ **Note**: Supplemental information.
- Increase line-height and optimize font sizes for better long-form reading comfort using the `prose` class from Tailwind (already partially implemented).

### 4.3 Navigation Polish
- **Breadcrumbs**: Add breadcrumbs to the top of doc pages for easier hierarchy navigation.
- **Pagination**: Add "Previous" and "Next" buttons at the bottom of every page to guide users through the documentation flow.

## 5. Implementation Roadmap

### Phase 1: Infrastructure
- Define a `DocMetadata` struct to standardize how pages provide their title/description.
- Improve the `Sidebar` to handle nested routes more elegantly.
- Create the search indexer utility.

### Phase 2: Content Migration
- Break up `getstarted`, `routing`, and `client-runtime` into sub-pages.
- Update `Sidebar` links.
- Implement Breadcrumbs and Pagination components.

### Phase 3: Interactive Features
- Implement the Search UI and Fuse.js integration.
- Implement Dynamic ToC and scroll-spy.
- Add "Copy Code" buttons to all `CodeBlock` components.

## 6. Maintenance
To ensure the documentation stays healthy:
- Add a periodic check to ensure all links in the sidebar are valid.
- Automate the search index generation as part of the `gospa build` pipeline.
