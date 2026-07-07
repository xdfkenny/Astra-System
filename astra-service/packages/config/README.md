# `@astra/config`

Shared configuration presets for the Astra-Service offline-first self-checkout platform.

## Install

This package is part of the workspace and is consumed via the `workspace:*` protocol.

```json
{
  "devDependencies": {
    "@astra/config": "workspace:*"
  }
}
```

## Presets

### TypeScript

- `@astra/config/typescript` (alias `@astra/config/typescript/base`) — strict base config with project references enabled.
- `@astra/config/typescript/react` — base config plus DOM libs and `jsx: react-jsx`.
- `@astra/config/typescript/node` — base config plus Node.js types.

In a package `tsconfig.json`:

```json
{
  "extends": "@astra/config/typescript/react",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"]
}
```

### ESLint

```js
import astraEslint from "@astra/config/eslint";

export default astraEslint;
```

The preset enables:

- TypeScript strict and stylistic type-checked rules
- `no-floating-promises`
- `no-explicit-any`
- `react-hooks/rules-of-hooks` and `react-hooks/exhaustive-deps`
- `react-refresh/only-export-components`
- `no-console` in production source files (allowed: `warn` and `error`)

### Tailwind CSS

```js
import astraTailwind from "@astra/config/tailwind";

export default {
  content: ["./src/**/*.{ts,tsx}"],
  presets: [astraTailwind],
};
```

The preset defines the Astra color system:

- Slate neutrals for surfaces, borders, and text
- `primary`: teal-600
- `cta`: amber-500
- `error`: rose-500

It also exposes matching `fontFamily`, `spacing`, `borderRadius`, `boxShadow`, and `zIndex` tokens.

### Vite

```ts
import { mergeConfig } from "vite";
import base from "@astra/config/vite";

export default mergeConfig(base, {
  // app-specific overrides
});
```

The shared config provides:

- React SWC plugin
- Native Federation plugin with empty `exposes`/`remotes` (add remotes in each app)
- `@/` path alias pointing to `src/`
- `es2022` build target, source maps, and vendor/ui chunking

### Vitest

```ts
import { mergeConfig } from "vitest/config";
import base from "@astra/config/vitest";

export default mergeConfig(base, {
  // app-specific overrides
});
```

The shared config provides:

- `environment: "jsdom"`
- `@vitest/coverage-v8` coverage provider
- `setupFiles` pointing to the bundled setup that registers Testing Library cleanup and `jest-dom` matchers
