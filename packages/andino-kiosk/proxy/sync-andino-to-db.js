import { createHash } from "node:crypto";

const ANDINO_BASE = process.env.ANDINO_BASE_URL || "https://andinoapp.com";
const ANDINO_TOKEN = process.env.ANDINO_ACCESS_TOKEN || "";
const SCHOOL_ID = process.env.ANDINO_SCHOOL_ID || "9";
const PG_CONN = process.env.PG_CONN || "postgres://astra:astra-system@postgres:5432/astra_service";

function uuidV5(name) {
  const ns = "6ba7b810-9dad-11d1-80b4-00c04fd430c8";
  const hash = createHash("sha1").update(ns + name).digest();
  hash[6] = (hash[6] & 0x0f) | 0x50;
  hash[8] = (hash[8] & 0x3f) | 0x80;
  const hex = hash.toString("hex");
  return `${hex.slice(0,8)}-${hex.slice(8,12)}-${hex.slice(12,16)}-${hex.slice(16,20)}-${hex.slice(20,32)}`;
}

async function fetchAll() {
  const first = await fetch(`${ANDINO_BASE}/api/pos/products?page=1`, {
    headers: { Authorization: `Bearer ${ANDINO_TOKEN}`, Accept: "application/json" },
  });
  const fj = await first.json();
  const total = fj.last_page || 1;
  let all = fj.data || [];
  for (let p = 2; p <= total; p++) {
    const r = await fetch(`${ANDINO_BASE}/api/pos/products?page=${p}`, {
      headers: { Authorization: `Bearer ${ANDINO_TOKEN}`, Accept: "application/json" },
    });
    const d = await r.json();
    all = all.concat(d.data || []);
  }
  return all;
}

function generateSQL(products) {
  const storeId = "550e8400-e29b-41d4-a716-446655440000";
  const lines = [];
  const catIds = {};

  for (const p of products) {
    if (!p || !p.id || !p.name) continue;
    const itemId = uuidV5(`andino-item-${p.id}`);
    const catName = (p.category && p.category.name) || "Uncategorized";
    const catId = uuidV5(`andino-cat-${catName}`);
    catIds[catName] = catId;

    const priceCents = Math.round(parseFloat(p.price || "0") * 100);
    const imageUrl = (p.image || "").replace(/\\\//g, "/");

    lines.push(
      `('${itemId}','${storeId}','${catId}','${p.name.replace(/'/g, "''")}','${(p.description || "").replace(/'/g, "''")}',${priceCents},'${imageUrl}')`
    );
  }

  const cats = [];
  for (const [name, id] of Object.entries(catIds)) {
    cats.push(`('${id}','${storeId}',NULL,'${name.replace(/'/g, "''")}','',${Object.keys(catIds).indexOf(name) + 1})`);
  }

  const sql = `
DELETE FROM inventory WHERE store_id = '${storeId}';
DELETE FROM item_modifier_groups WHERE item_id IN (SELECT item_id FROM items WHERE store_id = '${storeId}' AND name IN (${lines.map((l) => `'${l.split("'")[3]}'`).join(",")}));
DELETE FROM items WHERE store_id = '${storeId}' AND item_id LIKE '%-%';

INSERT INTO categories (category_id, store_id, parent_id, name, description, display_order) VALUES
${cats.join(",\n")}
ON CONFLICT (category_id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description;

INSERT INTO items (item_id, store_id, category_id, name, description, price_cents, image_url) VALUES
${lines.join(",\n")}
ON CONFLICT (item_id) DO UPDATE SET name = EXCLUDED.name, price_cents = EXCLUDED.price_cents, image_url = EXCLUDED.image_url;

INSERT INTO inventory (store_id, item_id, quantity_available, reorder_point, reorder_quantity)
SELECT '${storeId}', item_id, 100, 10, 50 FROM items WHERE store_id = '${storeId}'
ON CONFLICT (store_id, item_id) DO UPDATE SET quantity_available = 100;
`;
  return sql;
}

async function main() {
  console.log("Fetching Andino products...");
  const products = await fetchAll();
  console.log(`Got ${products.length} products`);

  const sql = generateSQL(products);
  console.log(sql);

  const fs = await import("node:fs");
  fs.writeFileSync("/tmp/sync.sql", sql);
  console.log("SQL written to /tmp/sync.sql");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
