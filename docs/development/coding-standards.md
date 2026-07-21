# Coding Standards

## General Principles

- Follow the principle of **least surprise**
- Prefer **clarity over cleverness**
- Write **self-documenting code** — clear names over comments
- Keep **functions small** (under 30 lines where possible)
- Use **meaningful, descriptive names** for variables, functions, types
- Follow the **existing patterns** in the codebase

## TypeScript / React

### Style Guide

- **Indentation:** 2 spaces
- **Quotes:** Single quotes
- **Semicolons:** Required
- **Trailing commas:** ES5 (objects, arrays)
- **Line width:** 100 characters
- **Formatting:** Prettier (automated)

### Naming Conventions

| Construct | Convention | Example |
|-----------|------------|---------|
| Variables/functions | camelCase | `getItemById`, `cartItems` |
| Types/interfaces | PascalCase | `CartItem`, `MenuResponse` |
| React components | PascalCase | `KioskShell`, `RouteGuard` |
| Files | kebab-case | `cart-service.ts`, `query-client.ts` |
| Constants | UPPER_SNAKE | `MAX_RETRY_COUNT` |
| Enums | PascalCase | `enum CartStatus { Active }` |

### React Patterns

- Use **functional components** with hooks
- **Custom hooks** for reusable logic (`use*` naming)
- **Zustand** for global session state
- **Valtio** for reactive cart state
- **XState** for complex workflow state
- **TanStack Query** for server state
- Type all props with **TypeScript interfaces**
- Export components as **named exports**

### State Management

- Local state → `useState` / `useReducer`
- Lifting state → props drilling (max 2 levels), then context
- Global workflow → XState machine
- Global session → Zustand store
- Reactive cart → Valtio proxy
- Server cache → TanStack Query

## Go

### Style Guide

- **Indentation:** 4 spaces (tabs)
- Follow `gofmt` / `go vet` / `golangci-lint` rules
- **Error handling:** Always check errors, return early
- **Context:** Pass `context.Context` as first parameter

### Project Structure

```
service/
├── cmd/service/
│   └── main.go         # Entry point (thin)
├── internal/
│   ├── config/         # Configuration loading
│   ├── middleware/     # HTTP/gRPC middleware
│   ├── router/         # Route definitions
│   ├── service/        # Business logic
│   ├── repository/     # Data access layer
│   └── models/         # Domain models
```

### Conventions

- **Explicit over implicit** — no global state
- **Dependency injection** via constructor functions
- **Interfaces** for testability
- **Small interfaces** (1-3 methods preferred)
- **Table-driven tests** with `t.Run()`

## Rust

### Style Guide

- **Indentation:** 4 spaces
- Follow `clippy -- -D warnings` and `rustfmt`
- **No `unsafe`** unless absolutely necessary (audited)
- **Strong types** — avoid raw strings/numbers where domain types exist
- **Proper error handling** — use `thiserror` / `anyhow` as appropriate

### Conventions

- `Result<T, E>` for fallible functions
- `Option<T>` for optional values (never `null`)
- Derive common traits: `Debug`, `Clone`, `PartialEq` where applicable
- Document public APIs with `///` doc comments
- Group imports: `std` → external crates → internal modules

## Python

- **Indentation:** 4 spaces
- **Type hints** on all function signatures
- Follow `ruff` linter rules
- `mypy` strict mode for type checking
- `pytest` for tests (no unittest)

## Protobuf

- **Package naming:** `astra.{service}.v1`
- **Message naming:** PascalCase
- **Field naming:** snake_case
- **Enum values:** `UPPER_SNAKE` with package prefix
- **Service names:** PascalCase with `Service` suffix
- **RPC names:** PascalCase
- `buf` lint + format before commit

## Documentation

- **Code comments:** Explain "why", not "what" (the code shows "what")
- **Doc comments:** Required for all exported functions/types
- **READMEs:** Update when adding new features or changing workflows
- **Architecture docs:** Update when changing system behavior

## Linting & Formatting

| Language | Linter | Formatter |
|----------|--------|-----------|
| TypeScript | Biome + ESLint | Prettier |
| Go | golangci-lint + go vet | gofmt |
| Rust | clippy | rustfmt |
| Python | ruff + mypy | ruff format |
| Protobuf | buf lint | buf format |
| YAML | - | Prettier |
