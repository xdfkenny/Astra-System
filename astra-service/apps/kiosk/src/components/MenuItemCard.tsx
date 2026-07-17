// Menu item card for browsing the catalog.
// Horizontal layout: square image | title/price/description.
// Full tap target on the card.
import { useState } from "react";
import { cn } from "@/utils/cn";
import type { MenuItem } from "@astra/shared-types";
import { useTranslation } from "../i18n";
import { useCurrencyFormat } from "../i18n/useCurrencyFormat";

export interface MenuItemCardProps {
  readonly item: MenuItem;
  readonly onClick?: () => void;
  readonly className?: string;
}

export function MenuItemCard({ item, onClick, className }: MenuItemCardProps): React.JSX.Element {
  const { t } = useTranslation();
  const { formatCurrency } = useCurrencyFormat();
  const [isPressed, setIsPressed] = useState(false);
  const hasModifiers = item.modifierGroups.length > 0;
  const price = formatCurrency(item.priceCents);
  const description = item.description ?? "";

  const handleClick = () => {
    if (onClick) {
      onClick();
    }
  };

  return (
    <div
      className={cn(
        "card relative cursor-pointer overflow-hidden",
        "rounded-[16px] bg-white/85 backdrop-blur-[4px] border border-taupe shadow-[0_2px_12px_rgba(45,42,38,0.06)]",
        "transition-all duration-100",
        isPressed ? "scale-[0.98]" : "hover:shadow-[0_4px_24px_rgba(45,42,38,0.08)] hover:-translate-y-0.5",
        className,
      )}
      onClick={handleClick}
      onMouseDown={() => { setIsPressed(true); }}
      onMouseUp={() => { setIsPressed(false); }}
      onMouseLeave={() => { setIsPressed(false); }}
    >
      <div className="flex h-[96px] w-full">
        <div className="relative h-full w-[96px] flex-shrink-0 overflow-hidden">
          {item.imageUrl ? (
            <img
              src={item.imageUrl}
              alt={item.name}
              className="h-full w-full object-cover"
              loading="lazy"
            />
          ) : (
            <div className="h-full w-full bg-stone/10 flex items-center justify-center">
              <div className="h-12 w-12 rounded-full bg-stone/20" />
            </div>
          )}
        </div>

        <div className="flex flex-1 flex-col p-3">
          <div className="flex items-start justify-between">
            <h3 className="font-[Georgia] text-[18px] font-semibold text-charcoal line-clamp-1">
              {item.name}
            </h3>
          </div>

          <p className="mt-1 font-sans text-[14px] text-stone line-clamp-2">
            {description}
          </p>

          <div className="mt-auto flex items-center justify-between">
            <span className="font-[Georgia] text-[18px] font-semibold tabular-nums text-charcoal">
              {price}
            </span>
            {hasModifiers && (
              <span className="font-sans text-[13px] font-medium text-denim uppercase">
                {t("item.modifiers")} →
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
