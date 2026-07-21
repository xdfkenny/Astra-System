/// <reference lib="webworker" />
import { precacheAndRoute, cleanupOutdatedCaches } from "workbox-precaching";
import { registerRoute } from "workbox-routing";
import { StaleWhileRevalidate, NetworkFirst, CacheFirst } from "workbox-strategies";
import { ExpirationPlugin } from "workbox-expiration";
import { BackgroundSyncPlugin } from "workbox-background-sync";
import type { WorkboxPlugin } from "workbox-core";

declare const self: ServiceWorkerGlobalScope;

/**
 * Service worker: offline-first shell + Background Sync API for queued
 * cart/order mutations. Injected precache manifest covers the app shell so a
 * kiosk can cold-boot fully offline after the first successful install.
 */
precacheAndRoute(self.__WB_MANIFEST);
cleanupOutdatedCaches();

registerRoute(
  ({ request }) => request.destination === "image",
  new CacheFirst({
    cacheName: "astra-menu-images",
    plugins: [new ExpirationPlugin({ maxEntries: 500, maxAgeSeconds: 60 * 60 * 24 * 30 }) as WorkboxPlugin],
  }),
);

registerRoute(
  ({ url }) => url.pathname.startsWith("/v1/menu") && url.origin === self.location.origin,
  new StaleWhileRevalidate({ cacheName: "astra-menu-data" }),
);

const orderSyncPlugin = new BackgroundSyncPlugin("astra-order-mutations-queue", {
  maxRetentionTime: 48 * 60,
}) as WorkboxPlugin;

registerRoute(
  ({ url, request }) => url.pathname.startsWith("/v1/orders") && request.method === "POST",
  new NetworkFirst({
    cacheName: "astra-order-submissions",
    plugins: [orderSyncPlugin],
    networkTimeoutSeconds: 5,
  }),
  "POST",
);

self.addEventListener("install", () => {
  void self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

