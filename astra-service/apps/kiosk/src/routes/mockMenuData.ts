import type { MenuItem, Category, MenuResponse } from "@astra/shared-types";

const storeId = "store-astra-001";
const now = new Date("2026-07-07T00:00:00Z").toISOString();

const categories = [
  { categoryId: "cat-coffee", name: "Coffee", displayOrder: 1 },
  { categoryId: "cat-pastry", name: "Pastry", displayOrder: 2 },
  { categoryId: "cat-sandwich", name: "Sandwiches", displayOrder: 3 },
  { categoryId: "cat-salad", name: "Salads & Bowls", displayOrder: 4 },
  { categoryId: "cat-beverage", name: "Beverages", displayOrder: 5 },
];

const blobs = [
  "L6Pl]}t7~qof~qt7Rjoe#XB?bI%1x]~q",
  "L5HhXy?b~qj[WBayfQfQfQfQfQf",
  "L4KpL_s8~qj[WBayfQfQfQfQfQf",
  "L3JrK_t7~qj[WBayfQfQfQfQfQf",
];

function item(
  idx: number,
  name: string,
  desc: string,
  priceCents: number,
  catIdx: number,
  hasModifiers = false,
): MenuItem {
  const cat = categories[catIdx];
  if (!cat) {
    throw new Error(`mockMenuData: no category at index ${String(catIdx)}`);
  }
  return {
    itemId: `item-${String(idx).padStart(3, "0")}`,
    storeId,
    categoryId: cat.categoryId,
    name,
    description: desc,
    priceCents,
    costCents: Math.round(priceCents * 0.35),
    plu: null,
    barcode: null,
    sku: `SKU-${String(idx).padStart(4, "0")}`,
    imageUrl: null,
    blurhash: blobs[idx % blobs.length] ?? null,
    taxCategory: "standard",
    isWeightBased: false,
    weightUnit: null,
    isActive: true,
    metadata: null,
    createdAt: now,
    updatedAt: now,
    deletedAt: null,
    modifierGroups: hasModifiers
      ? [
          {
            modifierGroupId: `mod-group-${idx}`,
            storeId,
            name: "Add-ons",
            description: null,
            minSelect: 0,
            maxSelect: 3,
            displayOrder: 1,
            isActive: true,
            createdAt: now,
            updatedAt: now,
            deletedAt: null,
            options: [
              {
                modifierOptionId: `mod-${idx}-1`,
                modifierGroupId: `mod-group-${idx}`,
                name: "Extra shot",
                priceDeltaCents: 75,
                isDefault: false,
                displayOrder: 1,
                isActive: true,
                createdAt: now,
                updatedAt: now,
                deletedAt: null,
              },
              {
                modifierOptionId: `mod-${idx}-2`,
                modifierGroupId: `mod-group-${idx}`,
                name: "Oat milk",
                priceDeltaCents: 50,
                isDefault: false,
                displayOrder: 2,
                isActive: true,
                createdAt: now,
                updatedAt: now,
                deletedAt: null,
              },
              {
                modifierOptionId: `mod-${idx}-3`,
                modifierGroupId: `mod-group-${idx}`,
                name: "Whipped cream",
                priceDeltaCents: 0,
                isDefault: true,
                displayOrder: 3,
                isActive: true,
                createdAt: now,
                updatedAt: now,
                deletedAt: null,
              },
            ],
          },
        ]
      : [],
    category: {
      categoryId: cat.categoryId,
      storeId,
      parentId: null,
      name: cat.name,
      description: null,
      displayOrder: cat.displayOrder,
      imageUrl: null,
      blurhash: null,
      isActive: true,
      createdAt: now,
      updatedAt: now,
      deletedAt: null,
    },
  };
}

export const mockMenuResponse: MenuResponse = {
  storeId,
  currency: "USD",
  taxRate: 0.0875,
  categories: categories as unknown as readonly Category[],
  items: [
    item(1, "Flat White", "Double ristretto with steamed oat milk", 550, 0, true),
    item(2, "Cold Brew", "24-hour steeped, served over ice", 480, 0, true),
    item(3, "Pour Over", "Single-origin, made to order", 620, 0, false),
    item(4, "Matcha Latte", "Ceremonial matcha with your choice of milk", 580, 0, true),
    item(5, "Espresso", "Double shot of our house blend", 350, 0, false),
    item(6, "Cortado", "Equal parts espresso and steamed milk", 450, 0, false),
    item(7, "Croissant", "Buttery, flaky, baked fresh", 420, 1, false),
    item(8, "Almond Croissant", "Frangipane-filled with sliced almonds", 520, 1, false),
    item(9, "Morning Bun", "Cinnamon-spiced with cream cheese glaze", 480, 1, false),
    item(10, "Banana Bread", "Sourdough banana loaf, walnut crumble", 390, 1, false),
    item(11, "Chocolate Chip Cookie", "Brown butter, sea salt, served warm", 320, 1, false),
    item(12, "Blueberry Scone", "Buttermilk scone with wild blueberries", 450, 1, false),
    item(13, "Turkey Brie Sandwich", "Roasted turkey, brie, arugula on ciabatta", 1290, 2, false),
    item(14, "Caprese Panini", "Fresh mozzarella, tomato, basil pesto", 1190, 2, false),
    item(
      15,
      "Egg & Avocado Toast",
      "Soft scrambled eggs, smashed avocado, sourdough",
      1090,
      2,
      false,
    ),
    item(16, "Ham & Swiss Croissant", "Black forest ham, gruyère, dijon butter", 1240, 2, false),
    item(17, "Kale Caesar Salad", "Shaved kale, parmesan, sourdough croutons", 1190, 3, false),
    item(18, "Harvest Bowl", "Quinoa, sweet potato, chickpea, tahini dressing", 1340, 3, false),
    item(19, "Noodle Salad", "Rice noodles, edamame, carrot, peanut dressing", 1240, 3, false),
    item(20, "Berry Smoothie", "Mixed berry, banana, yogurt, honey", 680, 4, false),
    item(21, "Fresh Orange Juice", "Squeezed to order", 550, 4, false),
    item(22, "Sparkling Water", "San Pellegrino, 330ml", 280, 4, false),
    item(23, "Kombucha", "House-brewed, ginger & turmeric", 520, 4, false),
    item(24, "Chai Latte", "House spiced chai concentrate, steamed milk", 540, 0, true),
  ],
};
