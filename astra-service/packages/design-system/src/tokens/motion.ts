export const duration = {
  instant: "80ms",
  fast: "150ms",
  base: "250ms",
  slow: "350ms",
  sheet: "300ms",
  page: "300ms",
} as const;

export const easing = {
  easeOutExpo: "cubic-bezier(0.16, 1, 0.3, 1)",
  easeInOutSoft: "cubic-bezier(0.4, 0, 0.2, 1)",
  easeSpring: "cubic-bezier(0.34, 1.56, 0.64, 1)",
  easeStandard: "cubic-bezier(0.2, 0, 0, 1)",
  easeEmphasized: "cubic-bezier(0.05, 0.7, 0.1, 1)",
} as const;

export const cssVariables: Record<string, string> = {
  ...Object.fromEntries(
    Object.entries(duration).map(([key, value]) => [`--astra-duration-${key}`, value]),
  ),
  ...Object.fromEntries(
    Object.entries(easing).map(([key, value]) => [`--astra-ease-${key}`, value]),
  ),
};
