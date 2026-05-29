const CACHE_NAME = 'limbo-cache-v35';
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
});

self.addEventListener('message', event => {
  if (event.data && event.data.action === 'skipWaiting') {
    self.skipWaiting();
  }
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
        }).catch(() => { });
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

self.addEventListener('push', event => {
  let data = { title: 'Limbo Update', body: 'A new notification is available.' };
  try {
    if (event.data) {
      data = event.data.json();
    }
  } catch (err) {
    console.error('Failed to parse push data:', err);
  }

  const actions = [];
  if (data.seerrUrl) {
    actions.push({
      action: 'open-seerr',
      title: 'Open Seerr'
    });
  }

  const options = {
    body: data.body,
    icon: '/assets/logo.svg',
    badge: '/assets/logo.svg',
    data: {
      url: data.url || '/',
      seerrUrl: data.seerrUrl
    },
    actions: actions
  };

  event.waitUntil(
    self.registration.showNotification(data.title, options)
  );
});

self.addEventListener('notificationclick', event => {
  event.notification.close();
  let urlToOpen = event.notification.data.url;

  if (event.action === 'open-seerr' && event.notification.data.seerrUrl) {
    urlToOpen = event.notification.data.seerrUrl;
  }

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(windowClients => {
      for (let client of windowClients) {
        if (client.url === urlToOpen && 'focus' in client) {
          return client.focus();
        }
      }
      if (clients.openWindow) {
        return clients.openWindow(urlToOpen);
      }
    })
  );
});
