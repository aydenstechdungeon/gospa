
// Documentation functionality: Search, Dynamic ToC, and Sidebar State
(function () {
    let searchIndex = null;
    let fuse = null;
    let sidebarScrollPos = 0;
    let searchInitializationPromise = null;

    async function initSearch() {
        if (searchIndex) return;
        if (searchInitializationPromise) return searchInitializationPromise;

        searchInitializationPromise = (async () => {
            try {
                const response = await fetch('/static/docs_search_index.json');
                searchIndex = await response.json();

                // Wait for Fuse to be available (it will be loaded via script tag)
                if (typeof Fuse === 'undefined') {
                    await new Promise(resolve => {
                        const script = document.createElement('script');
                        script.src = 'https://cdn.jsdelivr.net/npm/fuse.js@7.0.0/dist/fuse.basic.min.js';
                        script.onload = resolve;
                        document.head.appendChild(script);
                    });
                }

                fuse = new Fuse(searchIndex, {
                    keys: ['title', 'description', 'sections.title', 'content'],
                    threshold: 0.3,
                    includeMatches: true
                });
            } catch (err) {
                console.error('Failed to initialize search:', err);
                searchInitializationPromise = null; // Reset on failure so we can try again
            }
        })();

        return searchInitializationPromise;
    }

    function updateToC() {
        const tocList = document.querySelector('#toc ul');
        if (!tocList) return;

        tocList.innerHTML = '';
        const headings = document.querySelectorAll('.prose h2, .prose h3');

        if (headings.length === 0) {
            const tocContainer = document.querySelector('#toc')?.parentElement;
            if (tocContainer) tocContainer.classList.add('hidden');
            return;
        }
        const tocContainer = document.querySelector('#toc')?.parentElement;
        if (tocContainer) tocContainer.classList.remove('hidden');

        headings.forEach(heading => {
            const id = heading.id || heading.innerText.toLowerCase().replace(/\s+/g, '-');
            heading.id = id;

            const li = document.createElement('li');
            const a = document.createElement('a');
            a.href = `#${id}`;
            a.innerText = heading.innerText;
            a.className = 'hover:text-[var(--accent-primary)] transition-colors block py-1';

            if (heading.tagName === 'H3') {
                a.classList.add('pl-4', 'opacity-80');
            }

            li.appendChild(a);
            tocList.appendChild(li);
        });

        // Initialize scroll spy
        initScrollSpy();
    }

    function initScrollSpy() {
        const headings = Array.from(document.querySelectorAll('.prose h2, .prose h3'));
        const tocLinks = Array.from(document.querySelectorAll('#toc a'));

        // Remove existing scroll listener if any
        window.removeEventListener('scroll', handleScroll);
        window.addEventListener('scroll', handleScroll);

        function handleScroll() {
            let activeId = null;
            const scrollPos = window.scrollY + 100;

            headings.forEach(heading => {
                if (scrollPos >= heading.offsetTop) {
                    activeId = heading.id;
                }
            });

            tocLinks.forEach(link => {
                link.classList.remove('text-[var(--accent-primary)]', 'font-bold');
                if (link.getAttribute('href') === `#${activeId}`) {
                    link.classList.add('text-[var(--accent-primary)]', 'font-bold');
                }
            });
        }
    }

    // Save sidebar scroll position
    function saveSidebarScroll() {
        const sidebar = document.querySelector('#docs-sidebar aside');
        if (sidebar) {
            sidebarScrollPos = sidebar.scrollTop;
        }
    }

    // Restore sidebar scroll position
    function restoreSidebarScroll() {
        const sidebar = document.querySelector('#docs-sidebar aside');
        if (sidebar && sidebarScrollPos > 0) {
            sidebar.scrollTop = sidebarScrollPos;
        }
    }

    // Update sidebar active state based on current path
    function updateSidebarActiveState() {
        const currentPath = window.location.pathname;
        const sidebarLinks = document.querySelectorAll('#docs-sidebar a');

        sidebarLinks.forEach(link => {
            const href = link.getAttribute('href');
            if (!href) return;

            // Remove active classes
            link.classList.remove('bg-[var(--accent-primary)]/10', 'text-[var(--accent-primary)]', 'font-semibold', 'border-l-2', 'border-[var(--accent-primary)]', 'rounded-l-none');
            link.classList.add('text-[var(--text-secondary)]');

            // Add active classes if this is the current page
            const normalizedHref = href.replace(/\/$/, '');
            const normalizedPath = currentPath.replace(/\/$/, '');

            if (normalizedPath === normalizedHref ||
                (normalizedHref !== '/docs' && normalizedPath.startsWith(normalizedHref))) {
                link.classList.remove('text-[var(--text-secondary)]');
                link.classList.add('bg-[var(--accent-primary)]/10', 'text-[var(--accent-primary)]', 'font-semibold', 'border-l-2', 'border-[var(--accent-primary)]', 'rounded-l-none');
            }
        });
    }

    function handleSearch(query) {
        if (!fuse) return [];
        return fuse.search(query).slice(0, 8);
    }

    // Extract context snippet around matched text
    function getContextSnippet(result, query) {
        if (!result.matches || result.matches.length === 0) {
            return result.item.description || result.item.content.substring(0, 120) + '...';
        }

        // Find the best match (prioritize content matches)
        const contentMatch = result.matches.find(m => m.key === 'content');
        const match = contentMatch || result.matches[0];

        if (!match.value) {
            return result.item.description || result.item.content.substring(0, 120) + '...';
        }

        // Get the first matched indices
        const [start, end] = match.indices[0];
        const contextRadius = 60;
        const text = match.value;

        // Calculate snippet boundaries
        let snippetStart = Math.max(0, start - contextRadius);
        let snippetEnd = Math.min(text.length, end + contextRadius);

        // Expand to word boundaries
        while (snippetStart > 0 && text[snippetStart - 1] !== ' ') snippetStart--;
        while (snippetEnd < text.length && text[snippetEnd] !== ' ') snippetEnd++;

        let snippet = text.substring(snippetStart, snippetEnd);

        // Add ellipsis if truncated
        if (snippetStart > 0) snippet = '...' + snippet;
        if (snippetEnd < text.length) snippet = snippet + '...';

        return snippet;
    }

    // Highlight matched text in snippet
    function highlightSnippet(snippet, query) {
        const terms = query.toLowerCase().split(/\s+/).filter(t => t.length > 0);
        let highlighted = snippet;

        terms.forEach(term => {
            const regex = new RegExp(`(${escapeRegex(term)})`, 'gi');
            highlighted = highlighted.replace(regex, '<mark class="bg-[var(--accent-primary)]/20 text-[var(--accent-primary)] px-0.5 rounded">$1</mark>');
        });

        return highlighted;
    }

    function escapeRegex(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }

    function initAsyncCSS() {
        document.querySelectorAll('link[data-gospa-async-css]').forEach((preloadLink) => {
            if (preloadLink.dataset.gospaAsyncBound === '1') return;
            preloadLink.dataset.gospaAsyncBound = '1';
            preloadLink.addEventListener('load', () => {
                preloadLink.rel = 'stylesheet';
                preloadLink.removeAttribute('as');
            });
        });
    }

    function initRuntime() {
        const body = document.body;
        const runtimePath = body?.dataset.gospaRuntimePath;
        if (!runtimePath || window.GoSPA) return;

        const wsUrl = body.dataset.gospaWsUrl || '';
        const debug = body.dataset.gospaDebug === 'true';
        const hydrationMode = body.dataset.gospaHydrationMode || 'lazy';
        const hydrationTimeout = Number.parseInt(body.dataset.gospaHydrationTimeout || '3000', 10);
        const serializationFormat = body.dataset.gospaSerializationFormat || 'json';

        window.__GOSPA_CONFIG__ = {
            navigationOptions: {
                speculativePrefetching: {
                    enabled: true,
                    ttl: 45000,
                    hoverDelay: 80,
                    viewportMargin: 220,
                },
                serviceWorkerNavigationCaching: {
                    enabled: true,
                    cacheName: 'gospa-docs-navigation-cache',
                    path: '/gospa-navigation-sw.js',
                },
            },
        };

        import(runtimePath).then((runtime) => {
            window.GoSPA = runtime;
            runtime.init({
                wsUrl,
                debug,
                hydration: {
                    mode: hydrationMode,
                    timeout: hydrationTimeout,
                },
                serializationFormat,
            });
        }).catch((error) => {
            console.error('Failed to initialize GoSPA runtime:', error);
        });
    }

    function initGlobalActions() {
        // eslint-disable-next-line no-unused-vars
        window.switchLang = function switchLang(_btn, lang) {
            const targetLang = lang || localStorage.getItem('gospa-pref-lang') || 'js';
            localStorage.setItem('gospa-pref-lang', targetLang);

            const activeClasses = ['bg-[var(--accent-primary)]', 'text-white'];
            const inactiveClasses = ['bg-[var(--bg-primary)]', 'text-[var(--text-secondary)]', 'hover:text-[var(--text-primary)]'];

            document.querySelectorAll('[data-dual-code-block]').forEach((container) => {
                const jsBtn = container.querySelector('[data-lang="js"]');
                const tsBtn = container.querySelector('[data-lang="ts"]');
                const jsCode = container.querySelector('[data-code="js"]');
                const tsCode = container.querySelector('[data-code="ts"]');

                if (!jsBtn || !tsBtn || !jsCode || !tsCode) return;

                if (targetLang === 'js') {
                    activeClasses.forEach((c) => jsBtn.classList.add(c));
                    inactiveClasses.forEach((c) => jsBtn.classList.remove(c));
                    inactiveClasses.forEach((c) => tsBtn.classList.add(c));
                    activeClasses.forEach((c) => tsBtn.classList.remove(c));
                    jsCode.classList.remove('hidden');
                    tsCode.classList.add('hidden');
                } else {
                    activeClasses.forEach((c) => tsBtn.classList.add(c));
                    inactiveClasses.forEach((c) => tsBtn.classList.remove(c));
                    inactiveClasses.forEach((c) => jsBtn.classList.add(c));
                    activeClasses.forEach((c) => jsBtn.classList.remove(c));
                    tsCode.classList.remove('hidden');
                    jsCode.classList.add('hidden');
                }
            });
        };

        const initLang = () => window.switchLang(null);
        initLang();
        document.addEventListener('gospa:navigated', initLang);
    }

    // Listen for GoSPA navigation
    document.addEventListener('gospa:navigated', () => {
        // Save sidebar scroll before DOM update
        saveSidebarScroll();

        // Update docs-specific content
        updateToC();
        updateSidebarActiveState();

        // Restore sidebar scroll after a brief delay to allow DOM to settle
        setTimeout(() => {
            restoreSidebarScroll();
        }, 0);
    });

    // Also run on initial load
    window.addEventListener('load', () => {
        initRuntime();
        initGlobalActions();
        initAsyncCSS();
        updateToC();
        updateSidebarActiveState();
    });

    // Strategy: Lazy load search on interaction or intent (hover)
    function openSearch() {
        initSearch();
        document.getElementById('search-modal')?.classList.remove('hidden');
        document.getElementById('search-input')?.focus();
    }

    // Search UI interaction
    document.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault();
            openSearch();
        }
    });

    // Global event delegation for search
    document.addEventListener('click', (e) => {
        const actionTarget = e.target.closest('[data-action]');
        if (actionTarget) {
            const action = actionTarget.getAttribute('data-action');
            switch (action) {
                case 'copy-code': {
                    const codeEl = actionTarget.parentElement?.querySelector('pre:not(.hidden) code') ||
                        actionTarget.parentElement?.querySelector('code');
                    if (codeEl) {
                        navigator.clipboard.writeText(codeEl.innerText).then(() => {
                            const span = actionTarget.querySelector('span');
                            if (span) {
                                const old = span.innerText;
                                span.innerText = 'Copied!';
                                setTimeout(() => {
                                    span.innerText = old;
                                }, 2000);
                            }
                        });
                    }
                    break;
                }
                case 'toggle-mobile-menu':
                    document.getElementById('mobile-menu')?.classList.toggle('hidden');
                    document.body.classList.toggle('overflow-hidden');
                    break;
                case 'close-mobile-menu':
                    document.getElementById('mobile-menu')?.classList.add('hidden');
                    document.body.classList.remove('overflow-hidden');
                    break;
                default:
                    break;
            }
        }

        if (e.target.closest('[data-action="open-search"]')) {
            openSearch();
        }
        if (e.target.closest('[data-action="close-search"]') || (e.target.id === 'search-modal')) {
            document.getElementById('search-modal')?.classList.add('hidden');
        }
    });

    // Pre-load logic: Pre-fetch search index when the user hovers over the search button
    document.addEventListener('mouseover', (e) => {
        if (e.target.closest('[data-action="open-search"]')) {
            initSearch();
        }
    }, { once: false });

    document.addEventListener('input', (e) => {
        if (e.target.id === 'search-input') {
            const query = e.target.value;

            // Handle initialization state if typing starts before Fuse is ready
            if (!fuse && query.length > 0) {
                const list = document.getElementById('search-results');
                if (list) {
                    list.innerHTML = `
                        <div class="p-12 text-center flex flex-col items-center gap-4">
                            <div class="w-8 h-8 border-2 border-[var(--accent-primary)] border-t-transparent rounded-full animate-spin"></div>
                            <div class="text-[var(--text-muted)] text-sm font-medium">Initializing search engine...</div>
                        </div>
                    `;
                }
                initSearch().then(() => {
                    // Re-trigger search after initialization if query is still the same
                    if (e.target.value === query) {
                        e.target.dispatchEvent(new Event('input', { bubbles: true }));
                    }
                });
                return;
            }

            const results = handleSearch(query);
            const list = document.getElementById('search-results');
            if (!list) return;

            if (query.length === 0) {
                list.innerHTML = '';
                return;
            }

            if (results.length === 0) {
                list.innerHTML = '<div class="p-4 text-center text-[var(--text-muted)]">No results found</div>';
                return;
            }

            list.innerHTML = results.map(res => {
                const snippet = getContextSnippet(res, query);
                const highlightedSnippet = highlightSnippet(snippet, query);
                return `
                <a href="${res.item.url}" class="block p-4 hover:bg-[var(--bg-tertiary)] transition-all border-b border-[var(--border)] last:border-0 group">
                    <div class="flex items-center gap-2">
                        <div class="font-bold text-[var(--accent-primary)] group-hover:underline">${res.item.title}</div>
                        ${res.item.section ? `<span class="text-xs text-[var(--text-muted)]">— ${res.item.section}</span>` : ''}
                    </div>
                    <div class="text-sm text-[var(--text-secondary)] mt-1.5 leading-relaxed">${highlightedSnippet}</div>
                </a>
            `}).join('');
        }
    });
})();
