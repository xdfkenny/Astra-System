import { useMemo, useState, useCallback } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import type { MenuItem, ModifierOption } from "@astra/shared-types";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { cartService } from "../state/cartService";

function initializeSelections(item: MenuItem | null): Record<string, readonly string[]> {
  if (!item) return {};
  const defaults: Record<string, string[]> = {};
  for (const group of item.modifierGroups) {
    const preselected = group.options
      .filter((opt) => opt.isDefault)
      .map((opt) => opt.modifierOptionId)
      .slice(0, group.maxSelect);
    defaults[group.modifierGroupId] = preselected;
  }
  return defaults;
}

function toggleOption(
  groupId: string,
  option: ModifierOption,
  maxSelect: number,
  setSelections: React.Dispatch<React.SetStateAction<Record<string, readonly string[]>>>,
): void {
  setSelections((prev) => {
    const current = [...(prev[groupId] ?? [])];
    const idx = current.indexOf(option.modifierOptionId);
    if (idx >= 0) {
      current.splice(idx, 1);
    } else if (current.length < maxSelect) {
      current.push(option.modifierOptionId);
    }
    return { ...prev, [groupId]: current };
  });
}

function buildModifierSelections(
  item: MenuItem,
  selections: Record<string, readonly string[]>,
): { modifierId: string; optionId: string; priceDeltaCents: number }[] {
  const result: { modifierId: string; optionId: string; priceDeltaCents: number }[] = [];
  for (const group of item.modifierGroups) {
    const selected = selections[group.modifierGroupId] ?? [];
    for (const optionId of selected) {
      const option = group.options.find((o) => o.modifierOptionId === optionId);
      if (option) {
        result.push({
          modifierId: group.modifierGroupId,
          optionId,
          priceDeltaCents: option.priceDeltaCents,
        });
      }
    }
  }
  return result;
}

export function ItemModal(): React.JSX.Element | null {
  const { state, send } = useKioskMachine();
  const item = state.context.selectedItem;
  const [selections, setSelections] = useState<Record<string, readonly string[]>>(() =>
    initializeSelections(item),
  );
  const [quantity, setQuantity] = useState(1);

  const totalCents = useMemo(() => {
    if (!item) return 0;
    const modifiersDelta = item.modifierGroups.reduce((sum, group) => {
      const selected = selections[group.modifierGroupId] ?? [];
      return (
        sum +
        group.options
          .filter((opt) => selected.includes(opt.modifierOptionId))
          .reduce((g, opt) => g + opt.priceDeltaCents, 0)
      );
    }, 0);
    return (item.priceCents + modifiersDelta) * quantity;
  }, [item, selections, quantity]);

  const isValid = useMemo(() => {
    if (!item) return false;
    return item.modifierGroups.every((group) => {
      const selected = selections[group.modifierGroupId]?.length ?? 0;
      return selected >= group.minSelect && selected <= group.maxSelect;
    });
  }, [item, selections]);

  const handleAdd = useCallback((): void => {
    if (!item || !isValid) return;
    const modifierSelections = buildModifierSelections(item, selections);

    try {
      cartService.addItem(
        item.itemId,
        item.name,
        item.priceCents,
        quantity,
        modifierSelections,
      );
      send({ type: "ADD_TO_CART" });
    } catch (error) {
      console.error("Failed to add item to cart:", error);
      // Continue anyway - the item was added to local cart
      send({ type: "ADD_TO_CART" });
    }
  }, [item, isValid, selections, quantity, send]);

  const handleClose = useCallback((): void => {
    send({ type: "CLOSE_ITEM_DETAIL" });
  }, [send]);

  const handleDragEnd = useCallback(
    (_e: MouseEvent | TouchEvent | PointerEvent, info: { offset: { x: number; y: number } }) => {
      if (info.offset.y > 80) {
        handleClose();
      }
    },
    [handleClose],
  );

  if (!item) return null;

  return (
    <div className="absolute inset-0 z-30 flex items-end justify-center pointer-events-none">
      {/* Backdrop */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.2 }}
        className="absolute inset-0 bg-charcoal/20 pointer-events-auto"
        onClick={handleClose}
        aria-hidden="true"
      />

      {/* Bottom sheet */}
      <motion.div
        drag="y"
        dragConstraints={{ top: 0, bottom: 200 }}
        dragElastic={0.2}
        onDragEnd={handleDragEnd}
        initial={{ y: "100%" }}
        animate={{ y: 0 }}
        exit={{ y: "100%" }}
        transition={{
          duration: 0.3,
          ease: motionTokens.easeOutExpo,
        }}
        className="relative z-10 flex max-h-[90%] w-full flex-col rounded-t-[24px] bg-white/95 backdrop-blur-[12px] pointer-events-auto"
        role="dialog"
        aria-modal="true"
        aria-label={`Customize ${item.name}`}
      >
        {/* Handle */}
        <div
          className="mx-auto mt-3 h-1 w-10 rounded bg-taupe cursor-grab active:cursor-grabbing"
          aria-hidden="true"
        />

        {/* Image — top 40% of sheet */}
        <div className="h-[40%] min-h-[200px] w-full overflow-hidden rounded-t-[24px] bg-stone/10">
          <div
            className="h-full w-full object-cover"
            style={{
              background: item.imageUrl
                ? `url(${item.imageUrl}) center/cover`
                : "linear-gradient(135deg, rgba(107,104,98,0.06), rgba(196,184,168,0.06))",
            }}
            aria-hidden="true"
          />
        </div>

        {/* Content area */}
        <div className="flex flex-1 flex-col overflow-hidden px-4 pb-4">
          {/* Title, description, price */}
          <div className="flex items-start justify-between py-3">
            <div>
              <h2 className="font-heading text-[24px] font-semibold text-charcoal">
                {item.name}
              </h2>
            {item.description && (
              <p className="mt-1 font-sans text-[16px] text-stone">
                {item.description}
              </p>
            )}
              <p
                className="mt-2 font-sans text-[28px] font-semibold text-charcoal tabular-nums"
                aria-label={`Price $${(totalCents / 100).toFixed(2)}`}
              >
                ${(item.priceCents / 100).toFixed(2)}
              </p>
            </div>
            <button
              type="button"
              onClick={handleClose}
              className="flex h-10 w-10 items-center justify-center rounded-full bg-linen border border-taupe"
              aria-label="Close item details"
            >
              <svg viewBox="0 0 20 20" className="h-5 w-5 text-charcoal" fill="none" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                <path d="M5 5l10 10M15 5L5 15" strokeLinecap="round" />
              </svg>
            </button>
          </div>

          {/* Modifiers */}
          <div className="flex-1 overflow-y-auto">
            {item.modifierGroups.map((group) => (
              <div key={group.modifierGroupId} className="mb-4">
                <div className="mb-2 flex items-center gap-2">
                  <h3 className="font-sans text-[16px] font-medium text-charcoal">
                    {group.name}
                  </h3>
                  <span className="font-sans text-caption text-stone">
                    {group.minSelect === group.maxSelect
                      ? `Choose ${String(group.minSelect)}`
                      : `Choose ${String(group.minSelect)}-${String(group.maxSelect)}`}
                  </span>
                </div>
                <div className="flex flex-col gap-2">
                  {group.options.map((option) => {
                    const selected = (selections[group.modifierGroupId] ?? []).includes(
                      option.modifierOptionId,
                    );
                    return (
                      <button
                        key={option.modifierOptionId}
                        type="button"
                        onClick={() =>
                          { toggleOption(group.modifierGroupId, option, group.maxSelect, setSelections); }
                        }
                        aria-pressed={selected}
                        className={`flex min-h-[56px] items-center justify-between rounded-[12px] px-4 py-3 transition-colors duration-100 ${
                          selected
                            ? "border-moss bg-pale-mint/30 border"
                            : "bg-white/50 border border-taupe"
                        }`}
                      >
                        <span className="font-sans text-[16px] text-charcoal">
                          {option.name}
                        </span>
                        <span className="font-sans text-[16px] text-stone tabular-nums">
                          {option.priceDeltaCents > 0
                            ? `+$${(option.priceDeltaCents / 100).toFixed(2)}`
                            : "Included"}
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>

          {/* Quantity stepper */}
          <div className="flex items-center justify-center gap-4 py-3">
            <button
              type="button"
              onClick={() => { setQuantity((q) => Math.max(1, q - 1)); }}
              className="h-12 w-12 rounded-full bg-linen border border-taupe flex items-center justify-center"
              aria-label="Decrease quantity"
            >
              <svg
                viewBox="0 0 20 20"
                className="h-5 w-5 text-charcoal"
                fill="none"
                stroke="currentColor"
                strokeWidth={2}
                aria-hidden="true"
              >
                <path d="M5 10h10" strokeLinecap="round" />
              </svg>
            </button>
            <span
              className="font-sans text-[20px] font-semibold text-charcoal tabular-nums text-center min-w-[48px]"
              aria-label={`Quantity: ${String(quantity)}`}
            >
              {quantity}
            </span>
            <button
              type="button"
              onClick={() => { setQuantity((q) => q + 1); }}
              className="h-12 w-12 rounded-full bg-linen border border-taupe flex items-center justify-center"
              aria-label="Increase quantity"
            >
              <svg
                viewBox="0 0 20 20"
                className="h-5 w-5 text-charcoal"
                fill="none"
                stroke="currentColor"
                strokeWidth={2}
                aria-hidden="true"
              >
                <path d="M10 5v10M5 10h10" strokeLinecap="round" />
              </svg>
            </button>
          </div>

          {/* Primary button */}
          <button
            type="button"
            disabled={!isValid}
            onClick={handleAdd}
            className="mt-2 h-16 w-full rounded-full bg-amber text-white font-sans text-[18px] font-medium shadow-[0_4px_16px_rgba(184,126,107,0.3)] disabled:opacity-50 disabled:grayscale-[0.5] transition-all duration-100 active:scale-[0.98] active:translate-y-[1px]"
            aria-label={`Add ${item.name} to cart — $${(totalCents / 100).toFixed(2)}`}
          >
            Add to cart — ${(totalCents / 100).toFixed(2)}
          </button>
        </div>
      </motion.div>
    </div>
  );
}

