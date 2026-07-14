import { useCallback, useEffect, useRef, useState } from "react";

interface UseScrollSpyOptions {
  readonly containerRef: React.RefObject<HTMLElement | null>;
  readonly sectionSelector: string;
  readonly threshold?: number;
  readonly rootMargin?: string;
  readonly onActiveChange?: (activeId: string | null) => void;
}

interface UseScrollSpyResult {
  readonly activeId: string | null;
  readonly observe: () => void;
  readonly disconnect: () => void;
}

/**
 * Observes section elements inside a scrollable container and reports the
 * most-visible section. Uses IntersectionObserver with a configurable threshold
 * (defaults to 0.5) so that the category tab reflects the category the user is
 * actually reading, not the one whose header merely grazed the top edge.
 */
export function useScrollSpy({
  containerRef,
  sectionSelector,
  threshold = 0.5,
  rootMargin,
  onActiveChange,
}: UseScrollSpyOptions): UseScrollSpyResult {
  const [activeId, setActiveId] = useState<string | null>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const onActiveChangeRef = useRef(onActiveChange);
  const activeIdRef = useRef(activeId);

  useEffect(() => {
    onActiveChangeRef.current = onActiveChange;
  }, [onActiveChange]);

  useEffect(() => {
    activeIdRef.current = activeId;
  }, [activeId]);

  const disconnect = useCallback((): void => {
    if (observerRef.current) {
      observerRef.current.disconnect();
      observerRef.current = null;
    }
  }, []);

  const observe = useCallback((): void => {
    const container = containerRef.current;
    if (!container) return;

    disconnect();

    const sections = container.querySelectorAll(sectionSelector);
    if (sections.length === 0) return;

    observerRef.current = new IntersectionObserver(
      (entries) => {
        // Pick the entry with the largest visible ratio. This is the category
        // currently occupying most of the viewport.
        let best: IntersectionObserverEntry | null = null;

        for (const entry of entries) {
          if (entry.isIntersecting) {
            if (best === null || entry.intersectionRatio > best.intersectionRatio) {
              best = entry;
            }
          }
        }

        if (best === null) {
          return;
        }

        const id = best.target.getAttribute("data-category-id");
        if (!id || id === activeIdRef.current) {
          return;
        }

        setActiveId(id);
        onActiveChangeRef.current?.(id);
      },
      {
        root: container,
        threshold,
        ...(rootMargin !== undefined && { rootMargin }),
      },
    );

    for (const section of sections) {
      observerRef.current.observe(section);
    }
  }, [containerRef, sectionSelector, threshold, rootMargin, disconnect]);

  useEffect(() => {
    observe();
    return disconnect;
  }, [observe, disconnect]);

  return { activeId, observe, disconnect };
}

