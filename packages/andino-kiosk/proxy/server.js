// Andino API Proxy — sits between kiosk and Andino cloud API
// Caches product/user data in Redis. Deploy alongside Astra-System.

import express from "express";
import fetch from "node-fetch";
import Redis from "ioredis";
import { createHash } from "node:crypto";

const app = express();
app.use(express.json());

app.use((_req, res, next) => {
  res.header("Access-Control-Allow-Origin", "*");
  res.header("Access-Control-Allow-Headers", "Authorization, Content-Type");
  res.header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
  if (_req.method === "OPTIONS") return res.sendStatus(200);
  next();
});

function uuidV5(name) {
  const ns = "6ba7b810-9dad-11d1-80b4-00c04fd430c8";
  const hash = createHash("sha1").update(ns + name).digest();
  hash[6] = (hash[6] & 0x0f) | 0x50;
  hash[8] = (hash[8] & 0x3f) | 0x80;
  const hex = hash.toString("hex");
  return `${hex.slice(0,8)}-${hex.slice(8,12)}-${hex.slice(12,16)}-${hex.slice(16,20)}-${hex.slice(20,32)}`;
}

const ANDINO_BASE = process.env.ANDINO_BASE_URL || "https://andinoapp.com";
const ANDINO_TOKEN = process.env.ANDINO_ACCESS_TOKEN || "";
const SCHOOL_ID = process.env.ANDINO_SCHOOL_ID || "9";
const PORT = process.env.PORT || 3000;
const CACHE_TTL = parseInt(process.env.CACHE_TTL || "300", 10);

const redis = new Redis({
  host: process.env.REDIS_HOST || "redis",
  port: parseInt(process.env.REDIS_PORT || "6379"),
  maxRetriesPerRequest: 3,
  retryStrategy: (times) => Math.min(times * 100, 3000),
});

redis.on("error", (err) => {
  console.warn("Redis unavailable, running without cache:", err.message);
});

function authHeaders() {
  return {
    Authorization: `Bearer ${ANDINO_TOKEN}`,
    Accept: "application/json",
  };
}

async function cacheGet(key) {
  try {
    const val = await redis.get(key);
    return val ? JSON.parse(val) : null;
  } catch {
    return null;
  }
}

async function cacheSet(key, data, ttl = CACHE_TTL) {
  try {
    await redis.setex(key, ttl, JSON.stringify(data));
  } catch { /* ignore */ }
}

// GET /api/products — fetch from Andino, cache in Redis
app.get("/api/products", async (_req, res) => {
  const cacheKey = `andino:products:${SCHOOL_ID}`;
  const cached = await cacheGet(cacheKey);
  if (cached) {
    res.set("X-Cache", "HIT");
    return res.json(cached);
  }

  try {
    // Andino paginates products — fetch all pages concurrently
    const first = await fetch(`${ANDINO_BASE}/api/pos/products?page=1`, { headers: authHeaders() });
    if (!first.ok) throw new Error(`Andino API: ${first.status}`);

    const firstJson = await first.json();
    const totalPages = firstJson.last_page || 1;
    let all = firstJson.data || [];

    if (totalPages > 1) {
      const pages = [];
      for (let p = 2; p <= totalPages; p++) {
        pages.push(
          fetch(`${ANDINO_BASE}/api/pos/products?page=${p}`, { headers: authHeaders() })
            .then((r) => r.json())
            .then((j) => j.data || [])
        );
      }
      const results = await Promise.all(pages);
      all = all.concat(...results);
    }

    res.set("X-Cache", "MISS");
    await cacheSet(cacheKey, all);
    res.json(all);
  } catch (err) {
    console.error("Products fetch failed:", err.message);
    if (cached) {
      res.set("X-Cache", "STALE");
      return res.json(cached);
    }
    res.status(502).json({ error: "Andino API unavailable", message: err.message });
  }
});

// GET /api/user/profile — proxy to Andino
app.get("/api/user/profile", async (req, res) => {
  const userId = req.query.userId;
  if (!userId) return res.status(400).json({ error: "userId required" });

  const cacheKey = `andino:user:${userId}`;
  const cached = await cacheGet(cacheKey);
  if (cached) {
    res.set("X-Cache", "HIT");
    return res.json(cached);
  }

  try {
    const r = await fetch(`${ANDINO_BASE}/api/auth/user`, { headers: authHeaders() });
    if (!r.ok) throw new Error(`Andino API: ${r.status}`);
    const data = await r.json();

    res.set("X-Cache", "MISS");
    await cacheSet(cacheKey, data);
    res.json(data);
  } catch (err) {
    if (cached) {
      res.set("X-Cache", "STALE");
      return res.json(cached);
    }
    res.status(502).json({ error: "Andino API unavailable" });
  }
});

// POST /api/auth/verify-pin — proxy to Andino
app.post("/api/auth/verify-pin", async (req, res) => {
  try {
    const r = await fetch(`${ANDINO_BASE}/api/auth/verify-pin`, {
      method: "POST",
      headers: { ...authHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify(req.body),
    });
    res.status(r.status).json(await r.json());
  } catch (err) {
    res.status(502).json({ error: "Andino API unavailable" });
  }
});

// GET /api/user/balance — proxy
app.get("/api/user/balance", async (req, res) => {
  try {
    const r = await fetch(`${ANDINO_BASE}/api/user/balance?userId=${req.query.userId}`, { headers: authHeaders() });
    res.status(r.status).json(await r.json());
  } catch (err) {
    res.status(502).json({ error: "Andino API unavailable" });
  }
});

// GET /v1/menu — fetch Andino products, transform to Astra MenuResponse format
app.get("/v1/menu", async (req, res) => {
  try {
    const storeId = req.query.store_id || "andino-school-9";
    const products = await doFetchAllProducts();

    const categoryMap = new Map();
    const items = [];

    for (const p of products) {
      if (!p || !p.id) continue;
      const cat = p.category || {};
      const catId = cat.id ? uuidV5(`andino-cat-${cat.name || "Uncategorized"}`) : uuidV5("andino-cat-Uncategorized");
      if (!categoryMap.has(catId)) {
        categoryMap.set(catId, {
          categoryId: catId,
          storeId,
          parentId: null,
          name: cat.name || "Uncategorized",
          description: cat.description || null,
          displayOrder: categoryMap.size + 1,
          imageUrl: cat.image || null,
          blurhash: null,
          isActive: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          deletedAt: null,
        });
      }

      items.push({
        itemId: uuidV5(`andino-item-${p.id}`),
        storeId,
        categoryId: catId,
        name: p.name,
        description: p.description || null,
        priceCents: Math.round(parseFloat(p.price) * 100),
        costCents: null,
        plu: null,
        barcode: null,
        sku: null,
        imageUrl: p.image || null,
        blurhash: null,
        taxCategory: "standard",
        isWeightBased: false,
        weightUnit: null,
        isActive: p.status === 1,
        metadata: null,
        modifierGroups: [],
        createdAt: p.created_at || new Date().toISOString(),
        updatedAt: p.updated_at || new Date().toISOString(),
        deletedAt: null,
      });
    }

    res.json({
      storeId,
      currency: "USD",
      taxRate: 0.0,
      categories: Array.from(categoryMap.values()),
      items,
    });
  } catch (err) {
    res.status(502).json({ error: "Andino API unavailable", message: err.message });
  }
});

async function doFetchAllProducts() {
  const cacheKey = `andino:products:${SCHOOL_ID}`;
  const cached = await cacheGet(cacheKey);
  if (cached) return cached;

  const first = await fetch(`${ANDINO_BASE}/api/pos/products?page=1`, { headers: authHeaders() });
  if (!first.ok) throw new Error(`Andino API: ${first.status}`);

  const firstJson = await first.json();
  const totalPages = firstJson.last_page || 1;
  let all = firstJson.data || [];

  if (totalPages > 1) {
    const pageFetches = [];
    for (let p = 2; p <= totalPages; p++) {
      pageFetches.push(
        fetch(`${ANDINO_BASE}/api/pos/products?page=${p}`, { headers: authHeaders() })
          .then((r) => r.json())
          .then((j) => j.data || [])
      );
    }
    const results = await Promise.all(pageFetches);
    all = all.concat(...results);
  }

  await cacheSet(cacheKey, all);
  return all;
}

// GET /api/products (legacy, keeps X-Cache headers)
app.get("/api/products", async (_req, res) => {
  const cacheKey = `andino:products:${SCHOOL_ID}`;
  const cached = await cacheGet(cacheKey);
  if (cached) {
    res.set("X-Cache", "HIT");
    return res.json(cached);
  }

  try {
    const all = await doFetchAllProducts();
    res.set("X-Cache", "MISS");
    res.json(all);
  } catch (err) {
    if (cached) {
      res.set("X-Cache", "STALE");
      return res.json(cached);
    }
    res.status(502).json({ error: "Andino API unavailable", message: err.message });
  }
});

// Health check
app.get("/health", (_req, res) => {
  res.json({ ok: true, service: "andino-proxy", uptime: process.uptime() });
});

app.listen(PORT, "0.0.0.0", () => {
  console.log(`Andino proxy listening on :${PORT}`);
  console.log(`Redis: ${process.env.REDIS_HOST || "redis"}:${process.env.REDIS_PORT || "6379"}`);
  console.log(`Cache TTL: ${CACHE_TTL}s`);
});
