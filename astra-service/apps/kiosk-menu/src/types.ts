import type { MenuItem, Category as MenuCategory } from "@astra/shared-types";

export type { MenuItem, MenuCategory };

export interface MenuCatalogResponse {
  readonly categories: readonly MenuCategory[];
  readonly items: readonly MenuItem[];
}
