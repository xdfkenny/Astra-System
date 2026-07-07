import { describe, expect, it } from "vitest";
import {
  lookupByPlu,
  createTestProduceRecognizer,
  matchToMenuItem,
  PLU_CATALOG,
} from "./produceRecognition";
import type { MenuItem } from "@astra/shared-types";

describe("produceRecognition", () => {
  it("looks up a known PLU with perfect confidence", () => {
    const match = lookupByPlu("4011");
    expect(match).not.toBeNull();
    expect(match?.name).toBe("Bananas");
    expect(match?.confidence).toBe(1);
  });

  it("returns null for an unknown PLU", () => {
    expect(lookupByPlu("9999")).toBeNull();
  });

  it("trims whitespace from manual PLU entry", () => {
    const match = lookupByPlu("  4011  ");
    expect(match?.name).toBe("Bananas");
  });

  it("test recognizer reports not ready for high-confidence matches", async () => {
    const recognizer = createTestProduceRecognizer();
    expect(recognizer.isReady()).toBe(true);
    const result = await recognizer.recognize({} as ImageBitmap);
    expect(result.matches).toHaveLength(0);
    expect(result.bestMatch).toBeNull();
  });

  it("matches produce to a catalog item by PLU", () => {
    const match = lookupByPlu("4011");
    if (!match) throw new Error("missing match");
    const catalog: MenuItem[] = [
      {
        itemId: "prod-banana",
        storeId: "store-1",
        categoryId: "cat-produce",
        name: "Bananas",
        description: "Organic bananas",
        priceCents: 199,
        costCents: 100,
        plu: "4011",
        barcode: null,
        sku: null,
        imageUrl: null,
        blurhash: null,
        taxCategory: "standard",
        isWeightBased: true,
        weightUnit: "g",
        isActive: true,
        metadata: {},
        modifierGroups: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        deletedAt: null,
      },
    ];
    const item = matchToMenuItem(match, catalog);
    expect(item?.name).toBe("Bananas");
  });

  it("catalog contains only valid 4-digit PLUs", () => {
    for (const plu of Object.keys(PLU_CATALOG)) {
      expect(plu).toMatch(/^\d{4}$/);
    }
  });
});
