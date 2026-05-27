const CACHE_NAME = 'limbo-cache-v1';
const ASSETS = [
  '/',
  '/index.html',
  '/manifest.json',
  '/css/styles.css',
  '/js/utils.js',
  '/js/api.js',
  '/js/components.js',
  '/js/app.js',
  '/assets/logo.svg'
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => {
      return cache.addAll(ASSETS).catch(err => console.warn('PWA: Cache pre-fill failed', err));
    })
  );
  self.skipWaiting();
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys => {
      return Promise.all(
        keys.map(key => {
          if (key !== CACHE_NAME) {
            return caches.delete(key);
          }
        })
      );
    })
  );
  self.clients.claim();
});

self.addEventListener('fetch', event => {
  // Bypass service worker interception for non-GET requests or external resources
  if (event.request.method !== 'GET' || !event.request.url.startsWith(self.location.origin)) {
    return;
  }

  // Bypass interception for API requests and page navigation.
  // This lets the browser handle auth redirects natively without CORS blocks.
  if (event.request.url.includes('/api/') || event.request.mode === 'navigate') {
    return;
  }
  
  event.respondWith(
    caches.match(event.request).then(cachedResponse => {
      if (cachedResponse) {
        // Fetch fresh copy in the background to update the cache
        fetch(event.request).then(networkResponse => {
          if (networkResponse.status === 200) {
            caches.open(CACHE_NAME).then(cache => cache.put(event.request, networkResponse));
          }
        }).catch(() => {});
        return cachedResponse;
      }
      
      return fetch(event.request).then(networkResponse => {
        if (networkResponse.status === 200) {
          const responseClone = networkResponse.clone();
          caches.open(CACHE_NAME).then(cache => cache.put(event.request, responseClone));
        }
        return networkResponse;
      }).catch(err => {
        // Fallback to cache if network fetch fails (e.g. CORS block, offline)
        return caches.match(event.request).then(cached => {
          if (cached) return cached;
          throw err;
        });
      });
    })
  );
});
