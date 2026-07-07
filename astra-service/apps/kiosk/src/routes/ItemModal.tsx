import { useMemo, useState } from "react";
import { motion } from "framer-motion";
import { IconButton, Badge } from "@astra/design-system";
import { PrimaryButton } from "@astra/ui-kit";
import { motion as motionTokens } from "@astra/design-tokens";
import type { MenuItem, ModifierOption } from "@astra/shared-types";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { addLineItem } from "@astra/kiosk-state";

/**
 * Item customization modal. Rendered as an overlay on top of the menu. Supports
 * modifier groups and enforces min/max selection rules before allowing Add to Cart.
 */
export function ItemModal(): React.JSX.Element | null {
  const { state, send } = useKioskMachine();
  const item = state.context.selectedItem;
  const [selections, setSelections] = useState<Record<string, readonly string[]>>(() =>
    initializeSelections(item),
  );

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
    return item.priceCents + modifiersDelta;
  }, [item, selections]);

  const isValid = useMemo(() => {
    if (!item) return false;
    return item.modifierGroups.every((group) => {
      const selected = selections[group.modifierGroupId]?.length ?? 0;
      return selected >= group.minSelect && selected <= group.maxSelect;
    });
  }, [item, selections]);

  if (!item) return null;

  const handleAdd = (): void => {
    if (!isValid) return;
    const modifierSelections = buildModifierSelections(item, selections);
    addLineItem({
      menuItemId: item.itemId,
      nameSnapshot: item.name,
      unitPriceCentsSnapshot: item.priceCents,
      quantity: 1,
      modifiers: modifierSelections,
    });
    send({ type: "ADD_TO_CART" });
  };

  return (
    <div className="absolute inset-0 z-modal flex items-end justify-center bg-overlay">
      <motion.div
        initial={{ y: "100%" }}
        animate={{ y: 0 }}
        exit={{ y: "100%" }}
        transition={{
          duration: motionTokens.durationBase,
          ease: motionTokens.easeStandard,
        }}
        className="flex max-h-[80%] w-full flex-col rounded-t-xl bg-surface p-6"
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <h2 className="font-heading text-2xl font-bold text-ink">{item.name}</h2>
            <p className="text-lg font-semibold text-primary" aria-label={`Price $${(totalCents / 100).toFixed(2)}`}>
              ${(totalCents / 100).toFixed(2)}
            </p>
          </div>
          <IconButton
            label="Close item details"
            onClick={() => {
              send({ type: "CLOSE_ITEM_MODAL" });
            }}
          >
            ×
          </IconButton>
        </div>

        <div className="flex-1 overflow-y-auto">
          {item.modifierGroups.map((group) => (
            <div key={group.modifierGroupId} className="mb-4">
              <div className="mb-2 flex items-center gap-2">
                <h3 className="font-heading text-lg font-semibold">{group.name}</h3>
                <Badge variant={(selections[group.modifierGroupId]?.length ?? 0) > group.maxSelect ? "error" : "default"}>
                  {group.minSelect === group.maxSelect
                    ? `Choose ${String(group.minSelect)}`
                    : `Choose ${String(group.minSelect)}-${String(group.maxSelect)}`}
                </Badge>
              </div>
              <div className="flex flex-col gap-2">
                {group.options.map((option) => {
                  const selected = (selections[group.modifierGroupId] ?? []).includes(option.modifierOptionId);
                  return (
                    <button
                      key={option.modifierOptionId}
                      type="button"
                      onClick={() => {
                        toggleOption(group.modifierGroupId, option, group.maxSelect, setSelections);
                      }}
                      aria-pressed={selected}
                      className={`hairline flex min-h-[var(--touch-min)] items-center justify-between rounded-md px-4 ${
                        selected ? "border-primary bg-primary/10" : "bg-surface"
                      }`}
                    >
                      <span className="font-medium">{option.name}</span>
                      <span className="tabular-nums text-ink-muted">
                        {option.priceDeltaCents > 0 ? `+$${(option.priceDeltaCents / 100).toFixed(2)}` : "Included"}
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>

        <PrimaryButton
          variant="primary"
          className="mt-4 w-full"
          style={{ minHeight: "72px" }}
          disabled={!isValid}
          onClick={handleAdd}
          aria-label={`Add ${item.name} to cart`}
        >
          Add to Cart — ${(totalCents / 100).toFixed(2)}
        </PrimaryButton>
      </motion.div>
    </div>
  );
}

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
