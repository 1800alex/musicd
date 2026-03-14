// Service Worker for Music Player PWA
const CACHE_VERSION = "music-player-v1";
const STATIC_CACHE = `${CACHE_VERSION}-static`;
const DYNAMIC_CACHE = `${CACHE_VERSION}-dynamic`;
const API_CACHE = `${CACHE_VERSION}-api`;

// Assets to cache on install
const STATIC_ASSETS = ["/ui/", "/ui/favicon.ico"];

// Install event - cache essential assets
self.addEventListener("install", (event) => {
	console.log("[SW] Installing service worker");
	event.waitUntil(
		caches
			.open(STATIC_CACHE)
			.then((cache) => {
				console.log("[SW] Caching static assets");
				return cache.addAll(STATIC_ASSETS).catch((err) => {
					console.warn("[SW] Some static assets failed to cache:", err);
				});
			})
			.then(() => self.skipWaiting())
	);
});

// Activate event - clean up old caches
self.addEventListener("activate", (event) => {
	console.log("[SW] Activating service worker");
	event.waitUntil(
		caches
			.keys()
			.then((cacheNames) => {
				return Promise.all(
					cacheNames
						.filter(
							(name) =>
								name.startsWith("music-player-") &&
								name !== STATIC_CACHE &&
								name !== DYNAMIC_CACHE &&
								name !== API_CACHE
						)
						.map((name) => {
							console.log("[SW] Deleting old cache:", name);
							return caches.delete(name);
						})
				);
			})
			.then(() => self.clients.claim())
	);
});

// Fetch event - network first with fallback to cache
self.addEventListener("fetch", (event) => {
	const { request } = event;
	const url = new URL(request.url);

	// Skip non-GET requests
	if (request.method !== "GET") {
		return;
	}

	// API requests - network first, cache as fallback
	if (url.pathname.startsWith("/api/") || url.pathname.startsWith("/ui/api/")) {
		event.respondWith(
			fetch(request)
				.then((response) => {
					if (!response || response.status !== 200 || response.type === "error") {
						return response;
					}
					const responseToCache = response.clone();
					caches.open(API_CACHE).then((cache) => {
						cache.put(request, responseToCache);
					});
					return response;
				})
				.catch(() => {
					return caches.match(request).then((cachedResponse) => {
						return cachedResponse || new Response("Offline - data not available", { status: 503 });
					});
				})
		);
		return;
	}

	// Static assets - cache first
	if (url.pathname.includes("/favicon") || url.pathname.includes("/static/") || url.pathname.includes("/ui/")) {
		event.respondWith(
			caches.match(request).then((cachedResponse) => {
				if (cachedResponse) {
					return cachedResponse;
				}
				return fetch(request)
					.then((response) => {
						if (!response || response.status !== 200) {
							return response;
						}
						const responseToCache = response.clone();
						caches.open(STATIC_CACHE).then((cache) => {
							cache.put(request, responseToCache);
						});
						return response;
					})
					.catch(() => {
						return new Response("Offline", { status: 503 });
					});
			})
		);
		return;
	}

	// Default - network first
	event.respondWith(
		fetch(request)
			.then((response) => {
				if (!response || response.status !== 200) {
					return response;
				}
				const responseToCache = response.clone();
				caches.open(DYNAMIC_CACHE).then((cache) => {
					cache.put(request, responseToCache);
				});
				return response;
			})
			.catch(() => {
				return caches.match(request).then((cachedResponse) => {
					return cachedResponse || new Response("Offline", { status: 503 });
				});
			})
	);
});

// Handle messages from clients
self.addEventListener("message", (event) => {
	if (event.data && event.data.type === "SKIP_WAITING") {
		self.skipWaiting();
	}
});
