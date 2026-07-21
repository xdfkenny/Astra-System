// Andino API Proxy — sits between kiosk and Andino cloud API
// Caches product/user data in Redis. Deploy alongside Astra-System.

import express from "express";
import fetch from "node-fetch";
import Redis from "ioredis";

const app = express();
app.use(express.json());

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

// Health check
app.get("/health", (_req, res) => {
  res.json({ ok: true, service: "andino-proxy", uptime: process.uptime() });
});

app.listen(PORT, "0.0.0.0", () => {
  console.log(`Andino proxy listening on :${PORT}`);
  console.log(`Redis: ${process.env.REDIS_HOST || "redis"}:${process.env.REDIS_PORT || "6379"}`);
  console.log(`Cache TTL: ${CACHE_TTL}s`);
});
