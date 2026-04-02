const url = new URL(self.location.href);
const requestedCacheName = url.searchParams.get('cacheName');
const CACHE_NAME = requestedCacheName ? `${requestedCacheName}-v1` : 'gospa-docs-navigation-cache-v1';

self.addEventListener('install', (event) => {
	event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches.keys().then((names) => Promise.all(names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name)))).then(() => self.clients.claim())
	);
});

self.addEventListener('fetch', (event) => {
	const { request } = event;
	if (request.method !== 'GET') return;
	const accept = request.headers.get('accept') || '';
	const isNavigation = request.mode === 'navigate' || accept.includes('text/html');
	if (!isNavigation) return;

	event.respondWith((async () => {
		const cache = await caches.open(CACHE_NAME);
		const cachedResponse = await cache.match(request);

		const networkPromise = fetch(request).then((response) => {
			if (response.ok) {
				cache.put(request, response.clone());
			}
			return response;
		}).catch(() => null);

		return cachedResponse || networkPromise;
	})());
});
