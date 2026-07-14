// Main menu browse screen with category tabs and item list.
// Horizontal category scroll. Infinite scroll for performance.
// Contains search functionality (pull to reveal).
import { useCallback, useEffect, useMemo, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import type { MenuItem } from "@astra/shared-types";
import { MenuItemCard } from "../components/MenuItemCard";
import { CartSummary } from "../components/CartSummary";
import { CategoryTabs } from "../components/CategoryTabs";
import { useDebouncedCallback } from "../hooks/useDebouncedCallback";

const ITEMS_PER_PAGE = 20;

const MENU_CATEGORIES: readonly { readonly id: string; readonly label: string }[] = [
  { id: "all", label: "All" },
  { id: "mains", label: "Mains" },
  { id: "drinks", label: "Drinks" },
];

const DEMO_STORE_ID = "store-demo";

function makeMenuItem(
  overrides: Pick<MenuItem, "itemId" | "name" | "priceCents" | "imageUrl" | "categoryId"> &
    Partial<Omit<MenuItem, "itemId" | "name" | "priceCents" | "imageUrl" | "categoryId">>,
): MenuItem {
  return {
    storeId: DEMO_STORE_ID,
    description: null,
    costCents: null,
    plu: null,
    barcode: null,
    sku: null,
    blurhash: null,
    taxCategory: "standard",
    isWeightBased: false,
    weightUnit: null,
    isActive: true,
    metadata: null,
    createdAt: "2024-01-01T00:00:00.000Z",
    updatedAt: "2024-01-01T00:00:00.000Z",
    deletedAt: null,
    modifierGroups: [],
    ...overrides,
  };
}

const ALL_ITEMS: MenuItem[] = [
  makeMenuItem({ itemId: "item-1", categoryId: "mains", name: "Artisan Burrito", priceCents: 1299, imageUrl: "https://picsum.photos/seed/burrito/200/200" }),
  makeMenuItem({ itemId: "item-2", categoryId: "mains", name: "Farm Fresh Salad", priceCents: 999, imageUrl: "https://picsum.photos/seed/salad/200/200" }),
  makeMenuItem({ itemId: "item-3", categoryId: "mains", name: "House Special Taco", priceCents: 399, imageUrl: "https://picsum.photos/seed/taco/200/200" }),
  makeMenuItem({ itemId: "item-4", categoryId: "drinks", name: "Craft Beer", priceCents: 599, imageUrl: "https://picsum.photos/seed/beer/200/200" }),
  makeMenuItem({ itemId: "item-5", categoryId: "drinks", name: "Organic Coffee", priceCents: 299, imageUrl: "https://picsum.photos/seed/coffee/200/200" }),
];

export function MenuScreen(): React.JSX.Element {
  const { state, send } = useKioskMachine();
  const [displayedItems, setDisplayedItems] = useState<MenuItem[]>([]);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchFocused, setSearchFocused] = useState(false);
  const [isSearching, setIsSearching] = useState(false);
  const [activeCategory, setActiveCategory] = useState<string | null>("all");

  const filteredMenuItems = useMemo<MenuItem[]>(
    () =>
      activeCategory === "all"
        ? ALL_ITEMS
        : ALL_ITEMS.filter((item) => item.categoryId === activeCategory),
    [activeCategory],
  );

  const handleCategoryChange = (categoryId: string): void => {
    setActiveCategory(categoryId);
    setDisplayedItems([]);
    setPage(1);
  };

  useEffect(() => {
    loadMoreItems();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (searchQuery) {
      setIsSearching(true);
      const query = searchQuery.toLowerCase();
      const filtered = filteredMenuItems.filter(
        (item) =>
          item.name.toLowerCase().includes(query) ||
          (item.description ?? "").toLowerCase().includes(query),
      );
      setDisplayedItems(filtered);
    } else {
      setIsSearching(false);
      loadMoreItems();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery, filteredMenuItems]);

  const loadMoreItems = useCallback(() => {
    if (isLoading || isSearching) return;

    setIsLoading(true);
    setTimeout(() => {
      const startIndex = (page - 1) * ITEMS_PER_PAGE;
      const endIndex = startIndex + ITEMS_PER_PAGE;
      const newItems = filteredMenuItems.slice(startIndex, endIndex);

      if (page === 1) {
        setDisplayedItems(newItems);
      } else {
        setDisplayedItems((prev) => [...prev, ...newItems]);
      }

      setPage((prev) => prev + 1);
      setIsLoading(false);
    }, 300);
  }, [page, isLoading, isSearching, filteredMenuItems]);

  const handleItemClick = (item: MenuItem) => {
    send({ type: "SELECT_ITEM", item });
  };

  const handleScroll = useDebouncedCallback(() => {
    const scrollPosition = window.scrollY + window.innerHeight;
    const threshold = document.documentElement.scrollHeight - 500;

    if (scrollPosition >= threshold && !isLoading && !isSearching) {
      loadMoreItems();
    }
  }, 200);

  useEffect(() => {
    window.addEventListener("scroll", handleScroll);
    return () => { window.removeEventListener("scroll", handleScroll); };
  }, [handleScroll]);

  const handleSearchFocus = () => {
    setSearchFocused(true);
  }

  const handleSearchBlur = () => {
    if (searchQuery === "") {
      setSearchFocused(false);
    }
  }

  const handleGoToCart = () => {
    send({ type: "GO_TO_CART" });
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-linen safe-top safe-bottom">
      <div className="flex-shrink-0 p-3">
        <CategoryTabs
          categories={MENU_CATEGORIES}
          activeCategory={activeCategory}
          onSelectCategory={handleCategoryChange}
          aria-label="Menu categories"
        />

        <div className="mt-3 relative">
          <AnimatePresence>
            {searchFocused || searchQuery ? (
              <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                className="absolute inset-x-0 z-20 bg-white/95 backdrop-blur-[12px] rounded-[16px] border border-taupe shadow-[0_4px_24px_rgba(45,42,38,0.08)] p-3"
                onClick={(e) => { e.stopPropagation(); }}
              >
                <input
                  type="search"
                  placeholder="Search menu..."
                  className="w-full px-4 py-3 rounded-full border border-taupe bg-white/80 font-sans text-[16px] focus:outline-none focus:ring-2 focus:ring-moss focus:border-transparent"
                  value={searchQuery}
                  onChange={(e) => { setSearchQuery(e.target.value); }}
                  onFocus={handleSearchFocus}
                  onBlur={handleSearchBlur}
                  autoFocus
                />
                {searchQuery && (
                  <button
                    type="button"
                    className="absolute right-4 top-1/2 -translate-y-1/2 p-1 text-stone hover:text-charcoal"
                    onClick={() => { setSearchQuery(""); }}
                    aria-label="Clear search"
                  >
                    ×
                  </button>
                )}
              </motion.div>
            ) : null}
          </AnimatePresence>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-3 pb-24">
        <AnimatePresence mode="popLayout">
          {displayedItems.map((item, index) => (
            <motion.div
              key={item.itemId}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ delay: Math.min(index * 0.02, 0.5) }}
              className="mb-3"
            >
              <MenuItemCard item={item} onClick={() => { handleItemClick(item); }} />
            </motion.div>
          ))}
        </AnimatePresence>

        {isLoading && (
          <div className="flex justify-center py-8">
            <div className="h-8 w-8 animate-spin rounded-full border-b-2 border-moss" />
          </div>
        )}

        {!isSearching && displayedItems.length === 0 && !isLoading && (
          <div className="flex flex-col items-center justify-center py-12">
            <div className="h-16 w-16 rounded-full bg-stone/10 flex items-center justify-center mb-4">
              <span className="text-2xl">🍽️</span>
            </div>
            <h3 className="font-heading text-[24px] font-medium text-charcoal mb-2">
              No items found
            </h3>
            <p className="font-sans text-[16px] text-stone text-center">
              Try a different search term or check all categories
            </p>
          </div>
        )}

        <div className="h-4" />
      </div>

      <CartSummary className="absolute bottom-0 left-0 right-0 z-10" onCheckout={handleGoToCart} />

      {state.context.cartHasItems && (
        <div className="absolute bottom-20 left-0 right-0 z-20 px-3 pointer-events-none">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="pointer-events-auto mx-auto max-w-sm"
          >
            <div className="rounded-[12px] bg-moss/10 border border-moss/20 px-4 py-3 backdrop-blur-[8px]">
              <p className="font-sans text-[14px] text-moss text-center">
                💼 {state.context.cartHasItems} items in cart
              </p>
            </div>
          </motion.div>
        </div>
      )}
    </div>
  );
}
