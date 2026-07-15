// Cart review screen with items, modifiers, and checkout.
// Expandable summary at top. Sticky bottom action bar.
// Dwell assist on primary button (>40s gentle pulse).
import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { cn } from "@/utils/cn";
import { PrimaryButton } from "../components/PrimaryButton";
import { SecondaryButton } from "../components/SecondaryButton";
import { QuantityStepper } from "../components/QuantityStepper";

const MOCK_CART_ITEMS = [
  {
    id: "cart-item-1",
    itemId: "item-1",
    name: "Artisan Burrito",
    description: "Hand-crafted with local ingredients",
    price: "$12.99",
    quantity: 1,
    modifiers: [
      { name: "Protein", option: "Chicken" },
      { name: "Rice", option: "Brown" },
    ],
    imageUrl: "https://picsum.photos/seed/burrito/200/200",
  },
  {
    id: "cart-item-2",
    itemId: "item-2",
    name: "Farm Fresh Salad",
    description: "Organic greens, seasonal vegetables",
    price: "$9.99",
    quantity: 2,
    modifiers: [],
    imageUrl: "https://picsum.photos/seed/salad/200/200",
  },
];

export function CartScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const [expanded, setExpanded] = useState(false);
  const [dwellTimer, setDwellTimer] = useState<ReturnType<typeof setTimeout> | null>(null);
  const [dwellCount, setDwellCount] = useState(0);

  const cartItems = MOCK_CART_ITEMS;
  const subtotal = cartItems.reduce(
    (sum, item) => sum + parseFloat(item.price.replace('$', '')) * item.quantity,
    0
  );
  const tax = subtotal * 0.0825;
  const total = subtotal + tax;

  const handleBackToMenu = () => {
    send({ type: "BACK_TO_MENU" });
  };

  const handleProceedToPayment = () => {
    send({ type: "PROCEED_TO_PAYMENT" });
  };

  const handleQuantityChange = (itemId: string, newQuantity: number) => {
    if (newQuantity === 0) {
      send({ type: "CART_UPDATED", cartHasItems: cartItems.length > 1 });
    }
  };

  const handleItemPress = () => {
    if (dwellTimer) clearTimeout(dwellTimer);

    const timer = setTimeout(() => {
      setDwellCount((count) => count + 1);
      if (dwellCount >= 2) {
        setExpanded(true);
      }
    }, 40_000);

    setDwellTimer(timer);
  };

  const handleItemRelease = () => {
    if (dwellTimer) {
      clearTimeout(dwellTimer);
      setDwellTimer(null);
    }
  };

  useEffect(() => {
    return () => {
      if (dwellTimer) clearTimeout(dwellTimer);
    };
  }, [dwellTimer]);

  useEffect(() => {
    if (dwellCount <= 0) {
      return undefined;
    }
    const pulseTimer = setTimeout(() => {
      setDwellCount(0);
    }, 5000);

    return () => { clearTimeout(pulseTimer); };
  }, [dwellCount]);

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-linen safe-top safe-bottom">
      <div className="flex-shrink-0 p-4">
        <h1 className="font-heading text-[36px] font-semibold text-charcoal">
          Your cart
        </h1>

        <motion.button
          className={cn(
            "mt-4 w-full rounded-[12px] bg-warm-cream/90 backdrop-blur-[8px] border border-taupe/20 p-4",
            "flex items-center justify-between transition-all duration-200",
            expanded && "ring-2 ring-moss ring-offset-2"
          )}
          onClick={() => { setExpanded(!expanded); }}
          whileTap={{ scale: 0.98 }}
        >
          <div>
            <p className="font-sans text-[14px] uppercase tracking-wider text-stone">
              Cart summary
            </p>
            <p className="font-sans text-[18px] font-medium text-charcoal">
              {cartItems.length} items • ${total.toFixed(2)}
            </p>
          </div>
          <motion.div
            animate={{ rotate: expanded ? 180 : 0 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
          >
            <svg
              className="h-5 w-5 text-stone"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path d="M19 9l-7 7-7-7" strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} />
            </svg>
          </motion.div>
        </motion.button>

        <AnimatePresence>
          {expanded && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.3, ease: "easeInOut" }}
              className="mt-3 overflow-hidden"
            >
              <div className="rounded-[12px] bg-white/80 border border-taupe/20 p-4">
                {cartItems.map((item) => (
                  <div key={item.id} className="flex items-center py-3 border-b border-taupe/10 last:border-b-0">
                    <img
                      src={item.imageUrl}
                      alt={item.name}
                      className="h-16 w-16 rounded-[8px] object-cover mr-4"
                    />
                    <div className="flex-1">
                      <h3 className="font-sans text-[16px] font-medium text-charcoal">
                        {item.name}
                      </h3>
                      <p className="font-sans text-[14px] text-stone">
                        ${item.price} × {item.quantity}
                      </p>
                      {item.modifiers.length > 0 && (
                        <div className="mt-1 flex flex-wrap gap-1">
                          {item.modifiers.map((modifier, idx) => (
                            <span
                              key={idx}
                              className="rounded-full bg-moss/10 px-2 py-0.5 text-[10px] font-medium text-moss"
                            >
                              {modifier.name}: {modifier.option}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="ml-4">
                      <QuantityStepper
                        value={item.quantity}
                        onChange={(newQty) => { handleQuantityChange(item.id, newQty); }}
                        min={1}
                        max={10}
                        size="sm"
                      />
                    </div>
                  </div>
                ))}

                <div className="mt-4 pt-4 border-t border-taupe/20">
                  <div className="flex justify-between text-[16px] font-medium">
                    <span className="text-stone">Subtotal</span>
                    <span className="text-charcoal">${subtotal.toFixed(2)}</span>
                  </div>
                  <div className="flex justify-between text-[16px] font-medium mt-1">
                    <span className="text-stone">Tax (8.25%)</span>
                    <span className="text-charcoal">${tax.toFixed(2)}</span>
                  </div>
                  <div className="flex justify-between text-[20px] font-semibold mt-2">
                    <span className="text-charcoal">Total</span>
                    <span className="text-amber">${total.toFixed(2)}</span>
                  </div>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <div className="flex-1 overflow-y-auto px-4 pb-24">
        <AnimatePresence mode="popLayout">
          {cartItems.map((item, index) => (
            <motion.div
              key={item.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ delay: Math.min(index * 0.05, 0.3) }}
              className={cn(
                "mb-4 rounded-[16px] bg-white/85 backdrop-blur-[4px] border border-taupe shadow-[0_2px_12px_rgba(45,42,38,0.06)]",
                "p-4",
                dwellCount > 1 && index === 0 && "ring-2 ring-moss ring-offset-2"
              )}
              onPointerDown={handleItemPress}
              onPointerUp={handleItemRelease}
              onTouchStart={handleItemPress}
              onTouchEnd={handleItemRelease}
            >
              <div className="flex items-start">
                <img
                  src={item.imageUrl}
                  alt={item.name}
                  className="h-20 w-20 rounded-[12px] object-cover mr-4 flex-shrink-0"
                />
                <div className="flex-1 min-w-0">
                  <h3 className="font-heading text-[20px] font-medium text-charcoal truncate">
                    {item.name}
                  </h3>
                  <p className="font-sans text-[14px] text-stone mt-1 line-clamp-2">
                    {item.description}
                  </p>
                  <p className="font-sans text-[18px] font-semibold text-charcoal mt-2">
                    ${item.price}
                  </p>
                  {item.modifiers.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                      {item.modifiers.map((modifier, idx) => (
                        <span
                          key={idx}
                          className="rounded-full bg-moss/10 px-2 py-0.5 text-[10px] font-medium text-moss"
                        >
                          {modifier.name}: {modifier.option}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
                <div className="ml-4 flex-shrink-0">
                  <QuantityStepper
                    value={item.quantity}
                    onChange={(newQty) => { handleQuantityChange(item.id, newQty); }}
                    min={1}
                    max={10}
                    size="md"
                  />
                </div>
              </div>
            </motion.div>
          ))}
        </AnimatePresence>

        <div className="h-20" />
      </div>

      <div className="absolute bottom-0 left-0 right-0 z-20 bg-linen border-t border-taupe/20 px-4 py-4">
        <div className="mx-auto max-w-md">
          <div className="flex gap-3">
            <SecondaryButton
              onClick={handleBackToMenu}
              className="flex-1"
              aria-label="Back to menu"
            >
              ← Back
            </SecondaryButton>
            <PrimaryButton
              onClick={handleProceedToPayment}
              className={cn("flex-2", dwellCount > 1 && "animate-pulse")}
              aria-label="Proceed to payment"
            >
              Pay ${total.toFixed(2)} →
            </PrimaryButton>
          </div>
        </div>
      </div>
    </div>
  );
}
