# Development Workflow

## Git Workflow

### Branch Strategy

- `main` — stable, production-ready code
- `feat/*` — feature branches
- `fix/*` — bug fixes
- `docs/*` — documentation changes
- `chore/*` — maintenance, dependencies

### Commit Convention

[Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]
[optional footer]
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`, `perf`

**Scopes:** `kiosk`, `admin`, `menu-service`, `cart-service`, `order-service`, `inventory-service`, `payment`, `sync`, `syncd`, `installer`, `proto`, `infra`, `docs`

### Pre-commit Hooks

Hooks managed by Lefthook (`lefthook.yml`):
- Lint staged files
- Format check
- Commit message lint (commitlint)

## Common Tasks

### Adding a New Feature

```
1. Create branch: git checkout -b feat/my-feature
2. Implement changes
3. Add tests
4. pnpm lint && pnpm typecheck && pnpm test
5. Create changeset: pnpm changeset
6. Commit: git commit -m "feat(scope): description"
7. Push: git push origin feat/my-feature
8. Open PR → CI validates
```

### Running Individual Services

```bash
# Single kiosk app with hot reload
pnpm turbo run dev --filter=@astra/kiosk

# Single Go service
cd astra-service/services/gateway
go run ./cmd/gateway/main.go

# Rust sync daemon
cd astra-service/sync-daemon
cargo run --release
```

### Working with Protobuf

```bash
# Generate Go code from .proto files
cd proto
buf generate

# Lint proto files
buf lint

# Check breaking changes
buf breaking --against '../../.git#branch=main'
```

## Building

```bash
# Build all packages
pnpm build

# Build specific package
pnpm turbo run build --filter=@astra/kiosk

# Build Go service
cd astra-service/services/gateway && go build ./...

# Build Rust sync daemon
cd astra-service/sync-daemon && cargo build --release

# Build all Docker images
docker compose build
```

## Changesets

The project uses [Changesets](https://github.com/changesets/changesets) for versioning:

```bash
# Create a changeset (prompts for type and message)
pnpm changeset

# Version packages (bumps versions, updates changelogs)
pnpm changeset version

# Publish (not typically done in dev)
pnpm release
```
