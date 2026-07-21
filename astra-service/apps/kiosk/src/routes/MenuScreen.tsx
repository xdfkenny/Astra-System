// Main menu browse screen with category tabs and item list.
// Horizontal category scroll. Infinite scroll for performance.
// Contains search functionality (pull to reveal).
import { useCallback, useEffect, useMemo, useState } from "react";
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import type { Category, MenuItem } from "@astra/shared-types";
import { MenuItemCard } from "../components/MenuItemCard";
import { CartSummary } from "../components/CartSummary";
import { CategoryTabs } from "../components/CategoryTabs";
import { useDebouncedCallback } from "../hooks/useDebouncedCallback";
import { useTranslation } from "../i18n";

const ITEMS_PER_PAGE = 20;
const ANDINO_PROXY_URL = "http://localhost:3001/v1/menu";

export function MenuScreen(): React.JSX.Element {
  const { t } = useTranslation();
  const { state, send } = useKioskMachine();
  const reduceMotion = useReducedMotion();
  const [allItems, setAllItems] = useState<MenuItem[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [displayedItems, setDisplayedItems] = useState<MenuItem[]>([]);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [isApiError, setIsApiError] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchFocused, setSearchFocused] = useState(false);
  const [isSearching, setIsSearching] = useState(false);
  const [activeCategory, setActiveCategory] = useState<string | null>("all");

  useEffect(() => {
    let cancelled = false;
    async function fetchMenu() {
      try {
        const res = await fetch(ANDINO_PROXY_URL);
        if (!res.ok) throw new Error("Menu fetch failed");
        const data = await res.json();
        if (cancelled) return;
        setAllItems(data.items as MenuItem[]);
        setCategories(data.categories as Category[]);
      } catch {
        if (!cancelled) setIsApiError(true);
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    }
    fetchMenu();
    return () => { cancelled = true; };
  }, []);

  const filteredMenuItems = useMemo<MenuItem[]>(
    () =>
      activeCategory === "all"
        ? allItems
        : allItems.filter((item) => item.categoryId === activeCategory),
    [activeCategory, allItems],
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

  const translatedCategories = useMemo(
    () => {
      const cats = [{ id: "all", label: t("menu.categories.all") }];
      for (const c of categories) {
        cats.push({ id: c.categoryId, label: c.name });
      }
      return cats;
    },
    [t, categories],
  );

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center bg-linen">
        <div className="h-10 w-10 animate-spin rounded-full border-b-2 border-moss" />
      </div>
    );
  }

  if (isApiError) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 bg-linen p-8">
        <p className="font-heading text-[24px] font-medium text-charcoal">
          {t("menu.noItems")}
        </p>
        <p className="font-sans text-[16px] text-stone text-center">
          {t("menu.noItemsHint")}
        </p>
      </div>
    );
  }

  const handleGoToCart = () => {
    send({ type: "GO_TO_CART" });
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-linen safe-top safe-bottom">
      <div className="flex-shrink-0 p-3">
        <CategoryTabs
          categories={translatedCategories}
          activeCategory={activeCategory}
          onSelectCategory={handleCategoryChange}
          aria-label={t("menu.categoriesLabel")}
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
                  placeholder={t("menu.search")}
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
                    aria-label={t("menu.clearSearch")}
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
              initial={reduceMotion ? { opacity: 0 } : { opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ delay: reduceMotion ? 0 : Math.min(index * 0.03, 0.4) }}
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
              {t("menu.noItems")}
            </h3>
            <p className="font-sans text-[16px] text-stone text-center">
              {t("menu.noItemsHint")}
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
                💼 {t("menu.itemsInCart", { count: Number(state.context.cartHasItems) })}
              </p>
            </div>
          </motion.div>
        </div>
      )}
    </div>
  );
}
