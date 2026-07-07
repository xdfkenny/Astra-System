import { useState } from "react";
import type { MenuItem } from "@astra/shared-types";

export interface MenuItemCardProps {
  readonly item: MenuItem;
  readonly assistHighlight: boolean;
  readonly onSelectItem: (item: MenuItem) => void;
}

/**
 * A single menu tile. Uses a CSS-only BlurHash-style gradient placeholder
 * (the real BlurHash decode-to-canvas step happens in `blurhashToDataUrl`
 * in production; inlined here as a deterministic CSS gradient keyed off the
 * hash string so this file has zero binary/canvas dependency for the
 * reference implementation) shown until the AVIF/WebP `<img>` fires `onLoad`.
 */
export function MenuItemCard({ item, assistHighlight, onSelectItem }: MenuItemCardProps): React.JSX.Element {
  const [loaded, setLoaded] = useState(false);

  const handleSelect = (): void => {
    onSelectItem(item);
  };

  return (
    <button
      type="button"
      onClick={handleSelect}
      disabled={!item.isActive}
      className={`hairline flex flex-col overflow-hidden rounded-md bg-surface text-left shadow-sm transition-transform active:scale-[0.98] disabled:opacity-40 ${
        assistHighlight ? "astra-assist-pulse" : ""
      }`}
      aria-label={`Select ${item.name}, $${(item.priceCents / 100).toFixed(2)}`}
    >
      <div className="relative h-32 w-full overflow-hidden bg-surface-sunken">
        {!loaded && item.blurhash && (
          <div
            className="absolute inset-0"
            style={{ background: hashToGradient(item.blurhash) }}
            aria-hidden="true"
          />
        )}
        {item.imageUrl ? (
          <img
            src={item.imageUrl}
            alt=""
            loading="lazy"
            decoding="async"
            onLoad={() => { setLoaded(true); }}
            className={`h-full w-full object-cover transition-opacity duration-300 ${loaded ? "opacity-100" : "opacity-0"}`}
          />
        ) : null}
      </div>
      <div className="flex flex-1 flex-col gap-1 p-3">
        <p className="line-clamp-1 font-medium text-ink">{item.name}</p>
        <p className="line-clamp-2 text-sm text-ink-muted">{item.description}</p>
        <p className="mt-auto font-heading text-lg font-bold text-ink">
          ${(item.priceCents / 100).toFixed(2)}
        </p>
      </div>
    </button>
  );
}

/** Deterministic gradient derived from the blurhash string — cheap visual placeholder. */
function hashToGradient(hash: string): string {
  let h1 = 0;
  let h2 = 0;
  for (let i = 0; i < hash.length; i++) {
    h1 = (h1 * 31 + hash.charCodeAt(i)) % 360;
    h2 = (h2 * 17 + hash.charCodeAt(i)) % 360;
  }
  return `linear-gradient(135deg, hsl(${String(h1)} 30% 88%), hsl(${String(h2)} 25% 94%))`;
}
