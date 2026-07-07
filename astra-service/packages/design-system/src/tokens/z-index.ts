export const zIndex = {
  base: 0,
  content: 10,
  stickyBar: 20,
  floatingCart: 20,
  modal: 30,
  bottomSheet: 30,
  toast: 40,
  offlineBanner: 40,
  attractLoop: 50,
  systemOverlay: 60,
} as const;

export const cssVariables: Record<string, string> = Object.fromEntries(
  Object.entries(zIndex).map(([key, value]) => [`--astra-z-${key}`, String(value)]),
);
