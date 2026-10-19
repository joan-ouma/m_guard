const CACHE_NAME = 'motorguard-cache-v1';
const OFFLINE_URLS = ['/', '/index.html', '/favicon.svg', '/manifest.webmanifest'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE_NAME)
      .then((cache) => cache.addAll(OFFLINE_URLS))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)))
      )
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  const url = event.request.url;

  // Do not intercept SSE, streaming, or API requests
  if (url.includes('/v1/') || url.includes('/sse') || event.request.method !== 'GET') {
    return;
  }

  event.respondWith(
    fetch(event.request).catch(() =>
      caches.match(event.request).then((res) => res || caches.match('/index.html'))
    )
  );
});

// Handle incoming AWS SNS / GCM / Web Push messages from the server
self.addEventListener('push', (event) => {
  let raw = {};
  if (event.data) {
    try {
      raw = event.data.json();
    } catch {
      raw = { title: 'MotorGuard Alert', body: event.data.text() };
    }
  }

  // AWS SNS wraps GCM/FCM payload in a stringified JSON key "GCM"
  let data = raw;
  if (raw.GCM) {
    try {
      data = typeof raw.GCM === 'string' ? JSON.parse(raw.GCM) : raw.GCM;
    } catch {
      data = raw;
    }
  }

  const notification = data.notification || {};
  const customData = data.data || {};

  const title = notification.title || data.title || raw.default || '🚨 MotorGuard Security Alert';
  const body =
    notification.body ||
    data.body ||
    data.activity ||
    customData.event_type ||
    raw.default ||
    'Critical vehicle alert received';
  const bikeId = customData.bike_id || data.bike_id;

  const options = {
    body: body,
    icon: notification.icon || data.icon || '/favicon.svg',
    badge: '/favicon.svg',
    tag: data.tag || `mg-sns-${Date.now()}`,
    vibrate: [300, 100, 300, 100, 300],
    data: {
      url: customData.url || (bikeId ? `/history?bike=${bikeId}` : '/'),
    },
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

// Handle clicking on OS browser notifications
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = event.notification.data?.url || '/';

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if (client.url && 'focus' in client) {
          if ('navigate' in client && targetUrl) {
            client.navigate(targetUrl);
          }
          return client.focus();
        }
      }
      if (clients.openWindow) {
        return clients.openWindow(targetUrl);
      }
    })
  );
});
