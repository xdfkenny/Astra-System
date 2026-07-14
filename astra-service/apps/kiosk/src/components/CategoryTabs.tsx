import { useCallback, useRef, useState } from "react";
import { motion } from "framer-motion";

export interface CategoryTab {
  readonly id: string;
  readonly label: string;
}

interface CategoryTabsProps {
  readonly categories: readonly CategoryTab[];
  readonly activeCategory: string | null;
  readonly onSelectCategory: (categoryId: string) => void;
  readonly "aria-label"?: string;
}

const TAP_DEBOUNCE_MS = 400;

/**
 * Horizontal, sticky category tabs for the kiosk menu.
 *
 * Uses onPointerDown as the primary input to eliminate the 300 ms touch delay
 * on mobile, with onClick as a fallback for keyboard / screen-reader
 * activation. A time-based debounce prevents the emulated click from
 * double-firing the action while still allowing horizontal scrolling of the
 * tab bar.
 */
export function CategoryTabs({
  categories,
  activeCategory,
  onSelectCategory,
  "aria-label": ariaLabel = "Menu categories",
}: CategoryTabsProps): React.JSX.Element {
  const scrollRef = useRef<HTMLDivElement>(null);
  const lastTapRef = useRef<number>(0);
  const [pressedId, setPressedId] = useState<string | null>(null);

  const scrollTabIntoView = useCallback((categoryId: string): void => {
    const scrollContainer = scrollRef.current;
    if (!scrollContainer) return;
    const tab = scrollContainer.querySelector(`[data-category-tab="${categoryId}"]`);
    if (tab instanceof HTMLElement) {
      tab.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
    }
  }, []);

  const handleSelect = useCallback(
    (categoryId: string): void => {
      const now = Date.now();
      if (now - lastTapRef.current < TAP_DEBOUNCE_MS) return;
      lastTapRef.current = now;

      setPressedId(categoryId);
      window.setTimeout(() => {
        setPressedId((current) => (current === categoryId ? null : current));
      }, 100);

      onSelectCategory(categoryId);
      scrollTabIntoView(categoryId);
    },
    [onSelectCategory, scrollTabIntoView],
  );

  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLButtonElement>, categoryId: string): void => {
      // Only handle primary pointer (mouse left button, touch, stylus).
      // We intentionally do NOT call e.preventDefault() so the tab bar remains
      // horizontally scrollable by dragging on a tab.
      if (e.button !== 0 && e.button !== -1) return;
      handleSelect(categoryId);
    },
    [handleSelect],
  );

  const handleClick = useCallback(
    (_e: React.MouseEvent<HTMLButtonElement>, categoryId: string): void => {
      // This fires for keyboard / screen-reader activation and for the
      // emulated mouse event after a touch. The debounce in handleSelect
      // swallows the 300 ms delayed emulated click.
      handleSelect(categoryId);
    },
    [handleSelect],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLButtonElement>, categoryId: string): void => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        handleSelect(categoryId);
      }
    },
    [handleSelect],
  );

  return (
    <nav
      className="category-tabs flex gap-2 px-3 py-2"
      aria-label={ariaLabel}
      ref={scrollRef}
    >
      {categories.map((cat) => {
        const isActive = activeCategory === cat.id;
        const isPressed = pressedId === cat.id;

        return (
          <button
            key={cat.id}
            type="button"
            data-category-tab={cat.id}
            role="tab"
            aria-selected={isActive}
            aria-controls={`category-section-${cat.id}`}
            onPointerDown={(e) => {
              handlePointerDown(e, cat.id);
            }}
            onClick={(e) => {
              handleClick(e, cat.id);
            }}
            onKeyDown={(e) => {
              handleKeyDown(e, cat.id);
            }}
            className={`category-tab tap-feedback relative shrink-0 rounded-full px-4 py-2 font-sans text-[13px] font-medium uppercase tracking-[0.08em] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-moss focus-visible:ring-offset-2 ${
              isActive
                ? "bg-moss text-white border-moss shadow-[0_2px_8px_rgba(90,122,92,0.25)]"
                : "bg-white/60 border border-taupe text-stone"
            } ${isPressed ? "scale-[0.97]" : ""}`}
            style={{
              minWidth: "56px",
              minHeight: "44px",
            }}
          >
            <span className="pointer-events-none">{cat.label}</span>
            {isActive && (
              <motion.span
                layoutId="active-tab-indicator"
                className="absolute inset-0 rounded-full border-2 border-white/30"
                transition={{ duration: 0.15 }}
                aria-hidden="true"
              />
            )}
          </button>
        );
      })}
    </nav>
  );
}

