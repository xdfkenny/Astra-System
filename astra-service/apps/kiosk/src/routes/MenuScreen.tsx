import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useSnapshot } from "valtio";
import { cartProxy } from "@astra/kiosk-state";
import type { MenuItem } from "@astra/shared-types";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { mockMenuResponse } from "./mockMenuData";
import { CartSummary } from "../components/CartSummary";
import { CategoryTabs } from "../components/CategoryTabs";
import { BottomSheet } from "../components/BottomSheet";
import { useScrollSpy } from "../hooks/useScrollSpy";
import { apiClient } from "../state/apiClient";

interface CategoryGroup {
  readonly categoryId: string;
  readonly name: string;
  readonly items: readonly MenuItem[];
}

const SAFE_AREA_TOP = 8;

function useMenuCatalog() {
  return useQuery<{ readonly items: readonly MenuItem[] }>({
    queryKey: ["menu-catalog"],
    queryFn: async () => {
      try {
        const response = await apiClient.getMenuCatalog();
        return {
          items: response.items,
        };
      } catch (error) {
        console.error("Failed to fetch menu catalog:", error);
        return mockMenuResponse;
      }
    },
    placeholderData: mockMenuResponse,
    staleTime: 300_000,
  });
}

function buildCategoryGroups(items: readonly MenuItem[]): readonly CategoryGroup[] {
  const map = new Map<
    string,
    { categoryId: string; name: string; items: MenuItem[] }
  >();

  for (const item of items) {
    const cat = item.category;
    if (!cat) continue;
    const existing = map.get(cat.categoryId);
    if (existing) {
      existing.items.push(item);
    } else {
      map.set(cat.categoryId, {
        categoryId: cat.categoryId,
        name: cat.name,
        items: [item],
      });
    }
  }

  return Array.from(map.values()).sort(
    (a, b) =>
      (a.items[0]?.category?.displayOrder ?? 0) -
      (b.items[0]?.category?.displayOrder ?? 0),
  );
}

export function MenuScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const cart = useSnapshot(cartProxy);
  const { data, isLoading } = useMenuCatalog();

  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [ghostCartOpen, setGhostCartOpen] = useState(false);
  const [tabsHeight, setTabsHeight] = useState(0);
  const reducedMotion = useReducedMotion();

  const scrollRef = useRef<HTMLDivElement>(null);
  const tabsRef = useRef<HTMLElement>(null);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isScrollingFromTabRef = useRef(false);
  const itemTapGuardRef = useRef<Set<string>>(new Set());

  const categories = useMemo(() => buildCategoryGroups(data?.items ?? []), [data]);

  const filteredCategories = useMemo(() => {
    if (!searchQuery) return categories;
    const q = searchQuery.toLowerCase();
    return categories
      .map((cat) => ({
        ...cat,
        items: cat.items.filter(
          (item) =>
            item.name.toLowerCase().includes(q) ||
            (item.description ?? "").toLowerCase().includes(q),
        ),
      }))
      .filter((cat) => cat.items.length > 0);
  }, [categories, searchQuery]);

  const isEmpty = !isLoading && filteredCategories.length === 0;
  const itemCount = cart.lines.reduce((sum, l) => sum + l.quantity, 0);
  const totalCents = cart.lines.reduce(
    (sum, l) =>
      sum +
      l.quantity *
        (l.unitPriceCentsSnapshot +
          l.modifiers.reduce((m, mod) => m + mod.priceDeltaCents, 0)),
    0,
  );

  const headerOffset = tabsHeight + SAFE_AREA_TOP;

  // Measure the sticky tab bar height so we can subtract it from scroll targets.
  useEffect(() => {
    const tabs = tabsRef.current;
    if (!tabs) return;

    const updateHeight = (): void => {
      setTabsHeight(tabs.offsetHeight);
    };

    updateHeight();
    const resizeObserver = new ResizeObserver(updateHeight);
    resizeObserver.observe(tabs);

    return () => {
      resizeObserver.disconnect();
    };
  }, []);

  const handleSearchInput = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
      searchTimerRef.current = setTimeout(() => {
        setSearchQuery(value);
      }, 300);
    },
    [],
  );

  const handleScrollToCategory = useCallback(
    (categoryId: string): void => {
      const header = document.getElementById(`category-${categoryId}`);
      const container = scrollRef.current;
      if (!header || !container) return;

      const containerRect = container.getBoundingClientRect();
      const headerRect = header.getBoundingClientRect();
      const headerTop = headerRect.top - containerRect.top + container.scrollTop;

      isScrollingFromTabRef.current = true;
      container.scrollTo({
        top: Math.max(0, headerTop - headerOffset),
        behavior: "smooth",
      });

      // Clear the programmatic-scroll flag after the animation completes.
      window.setTimeout(() => {
        isScrollingFromTabRef.current = false;
      }, 400);
    },
    [headerOffset],
  );

  const handleSelectCategory = useCallback(
    (categoryId: string): void => {
      setActiveCategory(categoryId);
      handleScrollToCategory(categoryId);
    },
    [handleScrollToCategory],
  );

  const handleSelectItem = useCallback(
    (item: MenuItem): void => {
      if (itemTapGuardRef.current.has(item.itemId)) return;
      itemTapGuardRef.current.add(item.itemId);
      window.setTimeout(() => {
        itemTapGuardRef.current.delete(item.itemId);
      }, 400);
      send({ type: "SELECT_ITEM", item });
    },
    [send],
  );

  const handleScrollSpyChange = useCallback(
    (categoryId: string | null): void => {
      if (isScrollingFromTabRef.current || categoryId === null) return;
      setActiveCategory(categoryId);
    },
    [],
  );

  const { observe: observeScrollSpy } = useScrollSpy({
    containerRef: scrollRef,
    sectionSelector: "[data-category-section]",
    threshold: 0.5,
    rootMargin: `-${headerOffset}px 0px 0px 0px`,
    onActiveChange: handleScrollSpyChange,
  });

  useEffect(() => {
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, []);

  useEffect(() => {
    observeScrollSpy();
  }, [observeScrollSpy, filteredCategories]);

  return (
    <div className="relative flex flex-1 flex-col overflow-hidden bg-linen">
      {/* Menu items scroll area — the tab bar is sticky inside this container */}
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto overscroll-none kiosk-scroll-container"
        role="list"
        aria-label="Menu items"
      >
        {/* Pull-down to reveal search */}
        <div className="relative z-sticky-bar">
          <AnimatePresence>
            {searchOpen && (
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 56, opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{ duration: 0.25, ease: motionTokens.easeInOutSoft }}
                className="overflow-hidden"
              >
                <div className="mx-3 mb-2">
                  <input
                    type="search"
                    placeholder="Search menu..."
                    onChange={handleSearchInput}
                    className="w-full h-12 rounded-[12px] bg-white/60 border border-taupe px-4 font-sans text-body text-charcoal placeholder:text-stone focus:outline-none focus-visible:ring-2 focus-visible:ring-moss focus-visible:ring-offset-2"
                    aria-label="Search menu items"
                    autoFocus
                  />
                </div>
              </motion.div>
            )}
          </AnimatePresence>
          <motion.div
            drag="y"
            dragConstraints={{ top: 0, bottom: 60 }}
            dragElastic={0.2}
            onDragEnd={(_e, info) => {
              if (info.offset.y > 40) {
                setSearchOpen(true);
              } else if (info.offset.y < -20 && searchOpen) {
                setSearchOpen(false);
              }
            }}
            className="flex justify-center py-1 cursor-grab active:cursor-grabbing touch-target"
            aria-label="Pull down to search"
            role="button"
            tabIndex={0}
          >
            <div className="h-1 w-10 rounded bg-taupe/40" />
          </motion.div>
        </div>

        {/* Sticky category tabs — measured for scroll offset */}
        <nav
          ref={tabsRef}
          className="sticky top-0 z-sticky-bar bg-linen border-b border-taupe/30"
          aria-label="Menu categories"
        >
          <CategoryTabs
            categories={categories.map((cat) => ({
              id: cat.categoryId,
              label: cat.name,
            }))}
            activeCategory={activeCategory}
            onSelectCategory={handleSelectCategory}
          />
        </nav>

        {/* Content */}
        {filteredCategories.map((cat) => (
          <section
            key={cat.categoryId}
            id={`category-section-${cat.categoryId}`}
            data-category-section
            data-category-id={cat.categoryId}
            role="listitem"
          >
            <h3
              id={`category-${cat.categoryId}`}
              className="sticky z-content bg-linen/95 backdrop-blur-[4px] px-3 py-2 font-sans text-caption uppercase tracking-[0.08em] text-stone"
              style={{
                top: headerOffset,
                scrollMarginTop: headerOffset,
              }}
            >
              {cat.name}
            </h3>
            <div className="px-3 pb-2">
              {cat.items.map((item, itemIdx) => (
                <motion.button
                  key={item.itemId}
                  type="button"
                  initial={reducedMotion ? false : { opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{
                    duration: 0.25,
                    delay: reducedMotion ? 0 : Math.min(itemIdx * 0.04, 0.4),
                    ease: motionTokens.easeOutExpo,
                  }}
                  onClick={() => {
                    handleSelectItem(item);
                  }}
                  onPointerDown={() => {
                    handleSelectItem(item);
                  }}
                  className="card-surface menu-item-card tap-feedback mb-2 flex w-full items-start gap-3 p-2 text-left active:bg-warm-cream/50"
                  aria-label={`${item.name}, $${(item.priceCents / 100).toFixed(2)}`}
                >
                  {/* Thumbnail */}
                  <div className="h-24 w-24 shrink-0 rounded-[12px] bg-stone/10 overflow-hidden">
                    <div
                      className="h-full w-full object-cover"
                      style={{
                        background: item.imageUrl
                          ? `url(${item.imageUrl}) center/cover`
                          : "linear-gradient(135deg, rgba(107,104,98,0.08), rgba(196,184,168,0.08))",
                      }}
                      aria-hidden="true"
                    />
                  </div>

                  {/* Content */}
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <div className="flex items-start justify-between gap-2">
                      <span className="font-sans text-[18px] font-medium text-charcoal truncate">
                        {item.name}
                      </span>
                      <span className="font-sans text-[18px] font-semibold text-charcoal tabular-nums shrink-0">
                        ${(item.priceCents / 100).toFixed(2)}
                      </span>
                    </div>
                    {item.description && (
                      <p className="font-sans text-[14px] text-stone line-clamp-2">
                        {item.description}
                      </p>
                    )}
                    {item.modifierGroups.length > 0 && (
                      <span className="mt-1 font-sans text-[13px] text-denim">
                        Customize →
                      </span>
                    )}
                  </div>
                </motion.button>
              ))}
            </div>
          </section>
        ))}

        {/* Empty state */}
        {isEmpty && (
          <div className="flex flex-1 flex-col items-center justify-center py-16">
            <svg
              viewBox="0 0 48 48"
              className="h-16 w-16 text-stone opacity-[0.08]"
              fill="none"
              stroke="currentColor"
              strokeWidth={1.5}
              aria-hidden="true"
            >
              <path d="M24 4C20 4 16 8 16 12C16 16 20 18 24 18C28 18 32 16 32 12C32 8 28 4 24 4Z" />
              <path d="M12 44C12 36 18 28 24 28C30 28 36 36 36 44" />
              <path d="M8 20L24 24L40 20" />
            </svg>
            <p className="mt-3 font-sans text-body text-stone">No items found</p>
          </div>
        )}

        {/* Loading skeleton */}
        {isLoading && (
          <div className="px-3" aria-busy="true">
            {Array.from({ length: 6 }, (_, i) => (
              <div
                key={i}
                className="card-surface mb-2 h-24 animate-pulse bg-white/50"
              />
            ))}
          </div>
        )}
      </div>

      {/* Cart summary band */}
      <CartSummary />

      {/* Floating cart pill */}
      <AnimatePresence>
        {itemCount > 0 && (
          <motion.button
            type="button"
            initial={{ scale: 0, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: motionTokens.easeSpring }}
            onClick={() => {
              send({ type: "GO_TO_CART" });
            }}
            onPointerDown={() => {
              send({ type: "GO_TO_CART" });
            }}
            className="fixed right-3 top-1/2 z-floating-cart flex items-center gap-2 rounded-full bg-moss px-4 py-3 shadow-md tap-feedback"
            aria-label={`Cart: ${String(itemCount)} items, $${(totalCents / 100).toFixed(2)}`}
          >
            <svg
              viewBox="0 0 24 24"
              className="h-5 w-5 text-white"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              aria-hidden="true"
            >
              <circle cx="9" cy="21" r="1" />
              <circle cx="20" cy="21" r="1" />
              <path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6" />
            </svg>
            <span className="font-sans text-[14px] font-medium text-white tabular-nums">
              {itemCount}
            </span>
          </motion.button>
        )}
      </AnimatePresence>

      {/* Ghost Cart Transfer bottom sheet */}
      <BottomSheet open={ghostCartOpen} onClose={() => { setGhostCartOpen(false); }}>
        <div className="flex flex-col gap-4">
          <h2 className="font-heading text-[24px] font-semibold text-charcoal">
            Cart found on your phone
          </h2>
          <p className="font-sans text-body text-stone">
            Add to this kiosk?
          </p>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => { setGhostCartOpen(false); }}
              onPointerDown={() => { setGhostCartOpen(false); }}
              className="flex-1 h-14 rounded-[16px] bg-white/70 border border-taupe font-sans text-[16px] font-medium text-charcoal tap-feedback"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => { setGhostCartOpen(false); }}
              onPointerDown={() => { setGhostCartOpen(false); }}
              className="flex-1 h-14 rounded-full bg-amber text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(184,126,107,0.3)] tap-feedback"
            >
              Transfer to kiosk
            </button>
          </div>
        </div>
      </BottomSheet>
    </div>
  );
}
