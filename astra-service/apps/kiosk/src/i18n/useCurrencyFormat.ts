import { useMemo } from "react";
import { useLocaleStore } from "./index";

export function useCurrencyFormat(): {
  formatCurrency: (cents: number) => string;
} {
  const locale = useLocaleStore((s) => s.locale);

  const formatter = useMemo(
    () =>
      new Intl.NumberFormat(locale, {
        style: "currency",
        currency: "USD",
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }),
    [locale],
  );

  const formatCurrency = useMemo(
    () => (cents: number) => formatter.format(cents / 100),
    [formatter],
  );

  return { formatCurrency };
}
