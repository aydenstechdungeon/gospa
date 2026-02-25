
// Documentation functionality: Search and Dynamic ToC
(function () {
    let searchIndex = null;
    let fuse = null;

    async function initSearch() {
        if (searchIndex) return;
        try {
            const response = await fetch('/static/docs_search_index.json');
            searchIndex = await response.json();

            // Wait for Fuse to be available (it will be loaded via script tag)
            if (typeof Fuse === 'undefined') {
                await new Promise(resolve => {
                    const script = document.createElement('script');
                    script.src = 'https://cdn.jsdelivr.net/npm/fuse.js@7.0.0';
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
        }
    }

    function updateToC() {
        const tocList = document.querySelector('#toc ul');
        if (!tocList) return;

        tocList.innerHTML = '';
        const headings = document.querySelectorAll('.prose h2, .prose h3');

        if (headings.length === 0) {
            document.querySelector('#toc').parentElement.classList.add('hidden');
            return;
        }
        document.querySelector('#toc').parentElement.classList.remove('hidden');

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

        window.addEventListener('scroll', () => {
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
        });
    }

    function handleSearch(query) {
        if (!fuse) return [];
        return fuse.search(query).slice(0, 8);
    }

    // Listen for GoSPA navigation
    document.addEventListener('gospa:navigated', () => {
        updateToC();
        initSearch();
    });

    // Also run on initial load
    window.addEventListener('load', () => {
        updateToC();
        initSearch();
    });

    // Search UI interaction
    document.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault();
            document.getElementById('search-modal')?.classList.remove('hidden');
            document.getElementById('search-input')?.focus();
        }
    });

    // Global event delegation for search
    document.addEventListener('click', (e) => {
        if (e.target.closest('[data-action="open-search"]')) {
            document.getElementById('search-modal')?.classList.remove('hidden');
            document.getElementById('search-input')?.focus();
        }
        if (e.target.closest('[data-action="close-search"]') || (e.target.id === 'search-modal')) {
            document.getElementById('search-modal')?.classList.add('hidden');
        }
    });

    document.addEventListener('input', (e) => {
        if (e.target.id === 'search-input') {
            const results = handleSearch(e.target.value);
            const list = document.getElementById('search-results');
            if (!list) return;

            if (results.length === 0) {
                list.innerHTML = '<div class="p-4 text-center text-[var(--text-muted)]">No results found</div>';
                return;
            }

            list.innerHTML = results.map(res => `
                <a href="${res.item.url}" class="block p-4 hover:bg-[var(--bg-tertiary)] transition-all border-b border-[var(--border)] last:border-0">
                    <div class="font-bold text-[var(--accent-primary)]">${res.item.title}</div>
                    <div class="text-xs text-[var(--text-secondary)] mt-1 opacity-80">${res.item.description || res.item.content.substring(0, 100)}...</div>
                </a>
            `).join('');
        }
    });
})();
