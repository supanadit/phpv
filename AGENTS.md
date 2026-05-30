<!-- gitnexus:start -->

# GitNexus — Code Intelligence

This project is indexed by GitNexus as **phpv** (3076 symbols, 9597 relationships, 269 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource                              | Use for                                  |
| ------------------------------------- | ---------------------------------------- |
| `gitnexus://repo/phpv/context`        | Codebase overview, check index freshness |
| `gitnexus://repo/phpv/clusters`       | All functional areas                     |
| `gitnexus://repo/phpv/processes`      | All execution flows                      |
| `gitnexus://repo/phpv/process/{name}` | Step-by-step execution trace             |

## CLI

| Task                                         | Read this skill file                                        |
| -------------------------------------------- | ----------------------------------------------------------- |
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md`       |
| Blast radius / "What breaks if I change X?"  | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?"             | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md`       |
| Rename / extract / split / refactor          | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md`     |
| Tools, resources, schema reference           | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md`           |
| Index, status, clean, wiki CLI commands      | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md`             |

<!-- gitnexus:end -->

---

# Clean Architecture (Go)

This project follows Clean Architecture. The dependency rule is absolute: **source code dependencies point inward only. Outer layers depend on inner layers. Inner layers know nothing about outer layers.**

## Layer Map (Innermost → Outermost)

```
┌─────────────────────────────────────────────┐
│                  domain/                     │  ← ENTITIES
│  Pure data structs. Zero framework imports.  │
│  Only stdlib allowed (errors, fmt, time).    │
├─────────────────────────────────────────────┤
│                {usecase}/                    │  ← USE CASES
│  Business logic. Declares repository         │
│  interfaces where they are consumed.         │
│  Imports: domain, stdlib. NEVER: internal/,  │
│  database drivers, HTTP/CLI frameworks.      │
├─────────────────────────────────────────────┤
│            internal/{delivery}/              │  ← DELIVERY
│  HTTP handlers, CLI commands, gRPC servers.  │
│  Translates external input to use-case calls.│
│  Maps domain errors to status codes.         │
│  Declares its own narrow service interfaces. │
├─────────────────────────────────────────────┤
│         internal/repository/{impl}/          │  ← INFRASTRUCTURE
│  Database, filesystem, HTTP clients.         │
│  Concrete implementations of repository      │
│  interfaces. Imports domain types only.      │
├─────────────────────────────────────────────┤
│               cmd/ or app/                   │  ← COMPOSITION ROOT
│  main.go ONLY. Wires concrete impls into     │
│  interfaces. No business logic allowed.      │
│  The single place concrete types meet        │
│  interfaces.                                 │
└─────────────────────────────────────────────┘
```

## Layer Directories and Naming

| Layer            | Directory                                                                                                                        | Package naming convention                                                     |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Entities         | `domain/`                                                                                                                        | Single `domain` package with one file per aggregate root                      |
| Use Cases        | `{feature}/` (e.g. `user/`, `order/`, `payment/`)                                                                                | One package per business feature. Each declares its own repository interfaces |
| Delivery         | `internal/{protocol}/` (e.g. `internal/rest/`, `internal/grpc/`, `internal/cli/`, `internal/kafka/`)                             | One package per delivery mechanism. Multiple delivery mechanisms coexist      |
| Infrastructure   | `internal/repository/{backend}/` (e.g. `internal/repository/mysql/`, `internal/repository/postgres/`, `internal/repository/s3/`) | One package per storage backend. Never imports use-case packages              |
| Composition Root | `cmd/{app}/` or `app/`                                                                                                           | Contains only `main.go`. All wiring, no logic                                 |

## Layer Rules

### domain/ — Entities

| ✅ Allowed                                     | ❌ Prohibited                                                         |
| ---------------------------------------------- | --------------------------------------------------------------------- |
| Struct definitions with JSON/DB tags           | Any import beyond stdlib (`errors`, `fmt`, `time`, `strings`, `sync`) |
| Domain enums, constants, value types           | Framework imports (Echo, Gin, Cobra, fx, wire)                        |
| Sentinel errors (`var ErrX = errors.New(...)`) | Database drivers (`database/sql`, `afero`)                            |
| Pure functions that operate on domain types    | HTTP/gRPC/CLI libraries                                               |
|                                                | Logging libraries (logrus, zap, slog)                                 |
|                                                | Repository or service interfaces                                      |
|                                                | Business logic methods that call external services                    |

**Compile check:** Domain must compile in isolation. Deleting every other directory must not break `domain/`.

### Use Cases — Service Layer

Each use-case package MUST follow this structure:

```go
package {feature}

// Repository interface — declared WHERE CONSUMED
type Repository interface {
    GetByID(ctx context.Context, id int64) (domain.Entity, error)
    Store(ctx context.Context, e *domain.Entity) error
}

type Service struct {
    repo Repository   // depends on interface, never concrete impl
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) DoSomething(ctx context.Context, input Input) (domain.Entity, error) {
    // business logic here
    // return domain.ErrX for business rule violations
}
```

| ✅ Allowed                                                     | ❌ Prohibited                                                   |
| -------------------------------------------------------------- | --------------------------------------------------------------- |
| Declaring repository interfaces in the package that calls them | Defining interfaces in `domain/`                                |
| Business logic methods on Service struct                       | Importing `internal/repository/` or any concrete implementation |
| Constructor accepts interfaces, returns concrete Service       | Importing `internal/` delivery packages                         |
| Imports: `domain` + stdlib + `context`                         | Importing HTTP/CLI frameworks (Echo, Cobra, fx)                 |
| Returning `domain.ErrX` sentinel errors                        | Importing `database/sql` or any driver                          |
| `context.Context` as first parameter                           | Creating concrete repository instances                          |
|                                                                | Reading environment variables directly                          |
|                                                                | Returning HTTP status codes or exit codes                       |

**Testability check:** Every use-case package must be fully testable with mock repositories. No real database, no real filesystem, no HTTP server needed to run its tests.

### Delivery Layer

| ✅ Allowed                                                  | ❌ Prohibited                                                |
| ----------------------------------------------------------- | ------------------------------------------------------------ |
| Importing use-case packages via their service interfaces    | Calling repository constructors directly                     |
| Translating domain errors to HTTP status codes / exit codes | Bypassing service layer to access repositories               |
| Declaring its own narrow service interface per feature      | Writing business logic or validation rules                   |
| Parsing request bodies, query params, CLI flags             | Importing `internal/repository/` or concrete implementations |
| Formatting responses and user output                        | Accessing database connections or filesystem directly        |

**Pattern:** Delivery declares its own interface of what it needs:

```go
// In internal/rest/ — the HTTP handler declares what IT needs:
type EntityService interface {
    GetByID(ctx context.Context, id int64) (domain.Entity, error)
    Store(ctx context.Context, e *domain.Entity) error
}

type Handler struct {
    Service EntityService   // depends on interface
}
```

The concrete `*usecase.Service` satisfies this interface implicitly via Go structural typing. Delivery never knows use-case package internals, only the interface shape.

**Multiple deliveries coexist:** The same `*Service` instance is injected into REST handler, gRPC server, Kafka consumer, and CLI command simultaneously. Adding a new delivery mechanism creates one new package and one new line in `main.go`. Zero changes to use-case or domain layers.

### Infrastructure Layer

| ✅ Allowed                                               | ❌ Prohibited                                                         |
| -------------------------------------------------------- | --------------------------------------------------------------------- |
| Importing `domain` types                                 | Importing use-case packages — infrastructure never knows who calls it |
| Satisfying repository interfaces from use-case packages  | Business logic or domain rule validation                              |
| Database drivers, filesystem APIs, HTTP clients          | Importing delivery packages                                           |
| Utility packages (`internal/config/`, `internal/utils/`) | Orchestrating multiple repositories                                   |
| Private query-builder helpers                            |                                                                       |

**Pattern:** Infrastructure satisfies use-case interfaces through Go's implicit interface satisfaction. The infrastructure package **never imports** the use-case package. It simply writes methods whose signatures match:

```go
// In internal/repository/mysql/ — no import of usecase package
type EntityRepository struct {
    DB *sql.DB
}

// These methods happen to match usecase.Repository interface
func (r *EntityRepository) GetByID(ctx context.Context, id int64) (domain.Entity, error) { ... }
func (r *EntityRepository) Store(ctx context.Context, e *domain.Entity) error { ... }
```

The compiler verifies at the call site in `main.go` that shapes match. No `implements` keyword exists in Go.

### Composition Root

| ✅ Allowed                                             | ❌ Prohibited                              |
| ------------------------------------------------------ | ------------------------------------------ |
| Creating concrete infrastructure instances             | Business logic (if/switch on domain rules) |
| Injecting implementations into use-case constructors   | Parsing CLI args or formatting output      |
| Injecting use-case services into delivery constructors | Anything beyond wiring and server startup  |
| Reading environment variables and configuration        |                                            |

**Pattern:** The composition root is the **only** file where concrete types meet interfaces:

```go
func main() {
    // 1. Create infrastructure (outer → inner)
    repo := mysql.NewEntityRepository(db)

    // 2. Create use cases (inject interfaces)
    svc := feature.NewService(repo)

    // 3. Wire deliveries (same svc injected everywhere)
    rest.NewHandler(e, svc)
    grpc.NewServer(s, svc)
    kafka.NewConsumer(c, svc)

    // 4. Start
    e.Start(":8080")
}
```

Swapping MySQL for PostgreSQL changes **one line**: the import and the constructor call. Swapping REST for gRPC changes **one line**. Running both simultaneously adds **one line**. The service and domain layers never change.

## Interface Rules

### Rule 1: Interfaces belong to the CONSUMER

```go
// ✅ CORRECT — interface in the package that calls it
// In feature/service.go:
type Repository interface {
    GetByID(ctx context.Context, id int64) (domain.Entity, error)
}

// In internal/repository/mysql/entity.go — no import of feature package:
type EntityRepository struct { DB *sql.DB }
func (r *EntityRepository) GetByID(ctx context.Context, id int64) (domain.Entity, error) { ... }
```

```go
// ❌ PROHIBITED — interface in the producer or domain package
// In domain/repository.go:
type Repository interface { ... }  // Domain never consumes repositories

// In internal/repository/mysql/entity.go:
import "project/feature"  // Infrastructure must not import use cases
```

### Rule 2: Go implicit interface satisfaction

The compiler verifies interface satisfaction at the **call site**, not at the definition site. No `implements` keyword. No import from infrastructure → use case needed. If the struct has methods with matching signatures, it satisfies the interface.

### Rule 3: Each delivery declares its own service interface

HTTP handlers, gRPC servers, CLI commands, and Kafka consumers each define their own narrow interface describing what they need from the service layer. A single concrete `*Service` satisfies all of them simultaneously through Go structural typing. Adding a new delivery never requires changing existing code.

## Testing Rules

| Layer                    | Test type   | Mock strategy                                                                   |
| ------------------------ | ----------- | ------------------------------------------------------------------------------- |
| `domain/`                | Unit        | No mocks — pure data and functions                                              |
| `{usecase}/`             | Unit        | Mock the Repository interface. Never touch real I/O                             |
| `internal/{delivery}/`   | Unit        | Mock the Service interface. Use httptest for HTTP, never real infra             |
| `internal/repository/*/` | Integration | Use real backend (temp filesystem, test database). Test I/O, not business logic |

## Prohibited Patterns

These patterns break the dependency rule and MUST NOT appear in code:

1. **`domain/` importing any third-party module.** `domain/` files import only stdlib.
2. **Use-case package importing infrastructure.** If `feature/service.go` imports `internal/repository/mysql`, it is permanently coupled to MySQL.
3. **Delivery bypassing use cases.** If a handler calls a repository constructor or method directly, it skips all business rules.
4. **Infrastructure containing business logic.** Repository implementations execute queries. They never validate business rules, check permissions, or enforce constraints.
5. **Interfaces defined in `domain/`.** Domain packages contain data structures. Repository and service interfaces belong in the packages that consume them.
6. **`main.go` containing business logic.** The composition root wires dependencies. It contains no `if` statements about domain rules.
7. **Circular imports.** If package A imports B and B imports A, the dependency rule is violated. Break the cycle by defining an interface in the inner layer.
8. **Use-case packages importing delivery.** Use cases never know how they are invoked. If a service imports `internal/rest/` or `internal/cli/`, it cannot be reused with a different delivery mechanism.
