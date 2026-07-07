import type { MenuItem } from "@astra/shared-types";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useProduceScanner } from "./useProduceScanner";

const baseItem: MenuItem = {
  itemId: "prod-banana",
  storeId: "store-1",
  categoryId: "cat-produce",
  name: "Bananas",
  description: "",
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
};

describe("useProduceScanner", () => {
  beforeEach(() => {
    vi.stubGlobal("navigator", {
      mediaDevices: {
        getUserMedia: vi.fn().mockResolvedValue({
          getTracks: () => [{ stop: vi.fn() }],
        }),
      },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("starts with scanning disabled and no errors", () => {
    const { result } = renderHook(() => useProduceScanner());
    expect(result.current.state.isScanning).toBe(false);
    expect(result.current.state.error).toBeNull();
  });

  it("submitPlu returns a catalog item for a known PLU", async () => {
    const { result } = renderHook(() => useProduceScanner());
    const catalog: readonly MenuItem[] = [baseItem];

    const item = await act(() => result.current.submitPlu("4011", catalog));

    expect(item?.name).toBe("Bananas");
    expect(result.current.state.lastMatch?.name).toBe("Bananas");
  });

  it("submitPlu returns null for an unknown PLU", () => {
    const { result } = renderHook(() => useProduceScanner());
    const item = result.current.submitPlu("9999", []);
    expect(item).toBeNull();
  });
});
