// Astra-Design-System - Core design system package
//
// Retail-adaptated Living Weave biophilic design system
//
// Exports:
// - Color tokens (light/dark theme support)
// - Typography scale (9:16 optimized)
// - Spacing system (8px grid)
// - Theme provider for runtime theme switching
// - Basic global styles

export { colors, darkColors } from './tokens/colors';
export { default as typography } from './tokens/typography';
export { default as spacing } from './tokens/spacing';

// Re-export design-system components and utilities from the main entry
// (also available via subpath exports `@astra/design-system/components`
// and `@astra/design-system/utils`). `export *` does not re-export
// default exports, so the providers are re-exported explicitly below.
export * from './components';
export * from './utils';
export { default as TailwindProvider } from './components/TailwindProvider';
export { default as ThemeProvider } from './components/ThemeProvider';

// Type exports
export type { ColorTokens } from './tokens/colors';
export type { TypographyScale } from './tokens/typography';
export type { SpacingScale } from './tokens/spacing';
