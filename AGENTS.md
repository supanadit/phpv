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
│            internal/{protocol}/              │  ← DELIVERY
│  HTTP handlers, CLI commands, gRPC servers.  │
│  Translates external input to use-case calls.│
│  Maps domain errors to status codes.         │
│  Declares its own narrow service interfaces. │
│  Sits directly under internal/, no nesting.  │
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
| Delivery         | `internal/{protocol}/` (e.g. `internal/rest/`, `internal/grpc/`, `internal/cli/`, `internal/kafka/`, `internal/graphql/`, `internal/websocket/`) | One package per delivery mechanism directly under `internal/`. Multiple delivery mechanisms coexist |
| Infrastructure   | `internal/repository/{backend}/` (e.g. `internal/repository/mysql/`, `internal/repository/postgres/`, `internal/repository/s3/`) | One package per storage backend. Never imports use-case packages              |
| Internal Workers | `internal/worker/`                                                                                                               | Background job processors, scheduled tasks, event consumers. A form of delivery for internal service-to-service communication |
| Composition Root | `cmd/{app}/` or `app/`                                                                                                           | Contains only `main.go`. All wiring, no logic                                 |

## The `internal/` Directory — Strict Classification

Go's `internal/` package provides compiler-enforced visibility: no external project can import it. But within the project, `internal/` is NOT a dumping ground. It has exactly three valid categories:

### ✅ What BELONGS in `internal/`

| Category | Directory | Contains |
|----------|-----------|----------|
| Delivery mechanisms | `internal/{protocol}/` | HTTP handlers, gRPC servers, CLI commands, WebSocket hubs, Kafka consumers/producers, GraphQL resolvers. Placed directly under `internal/` — no extra `delivery/` nesting. |
| Infrastructure implementations | `internal/repository/{backend}/` | Database repos, filesystem repos, cache repos, external API clients, message queue drivers |
| Internal workers | `internal/worker/` | Background job processors, cron-like scheduled tasks, event subscribers that react to internal service events |

Every package in `internal/` must be one of these three. If a package does not handle external I/O (network, disk, database, message queue) it does not belong in `internal/`.

### ❌ What DOES NOT belong in `internal/`

| Misplaced package | Why it's wrong | Where it belongs |
|-------------------|---------------|-----------------|
| `internal/config/` | Configuration is a cross-cutting concern loaded at startup. It should not be buried where use cases cannot cleanly depend on it. | Root-level `config/` — loaded in `main.go`, injected as a dependency into services that need it. Never accessed via global singleton. |
| `internal/{service}/` where `{service}` is not a delivery or repository (e.g. `internal/notifier/`, `internal/pricer/`, `internal/validator/`) | These are use-case / business logic services. They contain behavior, decisions, and orchestration — not I/O. | Root-level `{feature}/` package alongside other use cases. |
| `internal/utils/`, `internal/helpers/`, `internal/common/` | Utility dumping grounds hide the real dependencies of each layer and invite circular imports. | Split by what they actually do: pure functions in `domain/` or `pkg/`; logging wrappers injected as interfaces; OS or environment detection extracted into its own root-level use-case package. |
| `internal/models/`, `internal/types/` | If these are domain types, they belong in `domain/`. If they are request/response DTOs, they belong in the delivery package that uses them. | `domain/` for entity structs; delivery package for request/response types. |

### Decision Heuristic

To decide whether package `X` belongs in `internal/`, ask:

> Does this package directly perform I/O (disk, network, database, message queue, OS syscalls for files/processes)?

- **Yes** → `internal/` is correct (delivery, infrastructure, or worker)
- **No** → it is a use case, domain type, or configuration — move it out of `internal/`

### `internal/` Directory Anti-Patterns

```
// ❌ VIOLATION — services and config dumped into internal/
internal/
├── config/          // WRONG: config should be root-level, injected
├── notifier/        // WRONG: this is a use case, move to notifier/
├── pricer/          // WRONG: this is a use case, move to pricer/
├── utils/           // WRONG: utilities dumping ground
├── cli/             // OK: CLI delivery
└── repository/      // OK: infrastructure

// ✅ CORRECT — internal/ contains only delivery, infrastructure, workers
//           Delivery packages sit directly under internal/, not nested.
internal/
├── rest/
├── grpc/
├── cli/
├── graphql/
├── websocket/
├── kafka/
├── repository/
│   ├── mysql/
│   ├── postgres/
│   └── s3/
└── worker/
    └── cleanup/
```

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
| Private query-builder helpers                            | Orchestrating multiple repositories                                   |

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

## File-Level Relationship Map

For a feature named `order`, the concrete files and how they connect across layers:

```
domain/
└── order.go                # domain.Order struct (entity, pure data)

order/
└── service.go              # Repository interface (declared here — consumed by business logic)
                            # Service struct (business logic methods live here)
                            # Imports: domain, stdlib. NEVER imports internal/

internal/
├── rest/
│   └── order.go            # Service interface (declared here — what REST needs)
│                           # Handler struct (HTTP handler)
│                           # Imports: domain. Calls order.Service via interface
├── grpc/
│   └── order.go            # Own Service interface (what gRPC needs)
│                           # Server struct (gRPC handler)
│                           # Same *order.Service injected — zero changes to order/
├── kafka/
│   └── order.go            # Own Service interface (what Kafka needs)
│                           # Consumer struct (Kafka consumer)
├── repository/
│   └── mysql/
│       └── order.go        # Repository struct (concrete MySQL implementation)
│                           # Imports: domain. NEVER imports order/
│                           # Methods happen to match order.Repository interface
│                           # → compiler verifies shape match at call site in main.go
app/
└── main.go                 # Composition root — the ONLY file where concrete meets interface
                            # repo := mysql.NewOrderRepository(db)
                            # svc := order.NewService(repo)
                            # rest.NewOrderHandler(e, svc)
                            # grpc.NewOrderServer(s, svc)
                            # kafka.NewOrderConsumer(c, svc)
```

**Key rules from the map:**
- `order/service.go` **declares** the `Repository` interface. It **never** imports `internal/repository/mysql/`.
- `internal/repository/mysql/order.go` **implements** it by writing methods with matching signatures. It **never** imports `order/`.
- `internal/rest/order.go` declares its **own** narrow `Service` interface. It calls `order.Service` through that interface, not directly.
- `app/main.go` is the **only file** where `mysql.OrderRepository` meets `order.Repository` and `order.Service` meets `rest.Service`.
- Swapping MySQL for Postgres means creating `internal/repository/postgres/order.go` and changing **one import and one constructor call** in `main.go`.
- Adding gRPC means creating `internal/grpc/order.go` and **one new line** in `main.go`.

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
| `internal/{protocol}/`   | Unit        | Mock the Service interface. Use httptest for HTTP, never real infra             |
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
9. **Configuration accessed via global singleton.** `config.Get()` called from use cases or infrastructure creates hidden coupling and untestable code. Config structs belong in a root-level `config/` package. They are loaded in `main.go` and injected as constructor dependencies. No package other than `main.go` calls `config.Load()` or `config.Get()`.
10. **Business services inside `internal/`.** Any package in `internal/` that does not perform I/O (disk, network, database, message queue) is misplaced. Use cases, platform detection, compilers, validators, and other behavior/decision services belong in root-level `{feature}/` packages. `internal/` is exclusively for delivery mechanisms, repository implementations, and workers.

## uber-go/fx — Dependency Injection in Clean Architecture

`uber-go/fx` is a DI framework that replaces manual wiring in the composition root with declarative provider functions. It respects Go's implicit interface satisfaction while automating constructor call chains.

### How fx maps to Clean Architecture layers

| fx concept | Clean Architecture role |
|------------|------------------------|
| `fx.Provide(fn)` | Registers a constructor. `fn` returns a **concrete type** that fx tracks by its Go type identity. |
| Constructor parameters | fx resolves each parameter by finding a registered concrete type that satisfies it. If the parameter is an **interface**, fx finds a concrete type whose methods match. |
| `fx.Invoke(fn)` | The composition root. `fn` receives fully-resolved dependencies and starts the application. Equivalent to `main()` body. |
| `fx.Module(name, opts...)` | Groups related providers. Use one module per layer or per feature. |
| `fx.Lifecycle` | Start/stop hooks — HTTP servers, gRPC servers, DB connection pools, consumer group joins. |
| `fx.Annotate(fn, ...)` | Rewires return types: expose a concrete implementation `*mysql.Repo` **as** the interface `feature.Repository`. |

### Core Rule: Providers return CONCRETE types. Constructors accept INTERFACES.

fx matches by Go type identity. A provider returning `*mysql.OrderRepository` creates a value of type `*mysql.OrderRepository`. A constructor parameter of type `order.Repository` (an interface) will be satisfied by that concrete value **only if** `*mysql.OrderRepository` has all the methods — or if `fx.Annotate` explicitly casts it.

```go
// ✅ CORRECT — provider returns concrete type
func NewOrderRepository(db *sql.DB) *mysql.OrderRepository {
    return &mysql.OrderRepository{DB: db}
}

// Constructor accepts interface — fx resolves it from the concrete provider above
func NewService(repo order.Repository) *order.Service {
    return &order.Service{Repo: repo}
}
```

```go
// ❌ PROHIBITED — provider returning interface directly
func NewOrderRepository(db *sql.DB) order.Repository {
    return &mysql.OrderRepository{DB: db}  // fx loses the concrete type identity
}
```

When a provider returns an interface, fx cannot track which concrete type backs it. This breaks fx's ability to provide that same concrete value to other constructors that might need it (e.g., health checks, multiple interfaces). Always return the concrete struct, then use `fx.Annotate` or let Go's implicit satisfaction handle interface matching.

### Module Organization

Respect layer boundaries by grouping providers into modules:

```go
// Infrastructure module — provides concrete repository implementations
var RepositoryModule = fx.Module("repository",
    fx.Provide(
        mysql.NewOrderRepository,
        mysql.NewUserRepository,
        postgres.NewAuditRepository,
    ),
)

// Use-case module — provides service constructors (accept interfaces)
var UseCaseModule = fx.Module("usecase",
    fx.Provide(
        order.NewService,
        user.NewService,
    ),
)

// Delivery module — provides handlers, registers routes
var DeliveryModule = fx.Module("delivery",
    fx.Provide(
        rest.NewOrderHandler,
        grpc.NewOrderServer,
        kafka.NewOrderConsumer,
    ),
    fx.Invoke(rest.RegisterRoutes),
    fx.Invoke(grpc.StartServer),
)
```

```go
// app/main.go — the composition root assembles modules
func main() {
    app := fx.New(
        fx.Provide(
            NewConfig,
            NewDatabase,
        ),
        RepositoryModule,
        UseCaseModule,
        DeliveryModule,
        fx.NopLogger,  // suppress fx internal logging in production
    )

    app.Run()  // blocks until signal, handles lifecycle start/stop
}
```

### `fx.In` — Grouping Constructor Parameters

When a constructor needs many dependencies, use `fx.In` to group them into a struct instead of a long parameter list. This is purely organizational — it does not create new types in fx's container:

```go
// ✅ CORRECT — grouped parameters with fx.In
type ServiceParams struct {
    fx.In

    OrderRepo  order.Repository
    UserRepo   user.Repository
    AuditRepo  audit.Repository
    Config     *config.Config
}

func NewOrderService(p ServiceParams) *order.Service {
    return &order.Service{
        OrderRepo: p.OrderRepo,
        UserRepo:  p.UserRepo,
        AuditRepo: p.AuditRepo,
        Config:    p.Config,
    }
}
```

```go
// ❌ PROHIBITED — fx.In structs in domain/ or use-case packages
// fx.In imports "go.uber.org/fx". Domain and use-case packages
// must never import DI frameworks. fx.In structs belong in the
// same package as the composition root, or in a dedicated
// internal/di/ package that only the composition root imports.
```

### `fx.Annotate` — Exposing a Concrete Type as an Interface

When fx needs to provide the same concrete type under multiple interfaces, or when the concrete type is from a package that doesn't import the interface package, use `fx.Annotate`:

```go
fx.Provide(
    mysql.NewOrderRepository,  // returns *mysql.OrderRepository
    fx.Annotate(
        mysql.NewOrderRepository,
        fx.As(new(order.Repository)),    // expose as order.Repository
        fx.As(new(audit.OrderReader)),   // expose as audit.OrderReader
    ),
)
```

Without `fx.Annotate`, fx only knows `*mysql.OrderRepository`. If a constructor asks for `order.Repository`, fx won't match it unless `*mysql.OrderRepository` is explicitly annotated — or unless the call site uses implicit interface satisfaction at the provider level.

> **Go tip:** If the provider returns `*mysql.OrderRepository` and a constructor parameter is `order.Repository`, fx resolves it automatically via Go's implicit interface satisfaction — no `fx.Annotate` needed. Use `fx.Annotate` only when you need to expose the same concrete type under **multiple** interfaces, or when the provider package cannot import the interface package.

### Lifecycle Hooks

Use `fx.Lifecycle` to manage start/stop for servers, connections, and consumers:

```go
func NewRESTServer(lc fx.Lifecycle, cfg *config.Config, handler *rest.OrderHandler) *echo.Echo {
    e := echo.New()
    handler.Register(e)

    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            go e.Start(cfg.Address)  // non-blocking start
            return nil
        },
        OnStop: func(ctx context.Context) error {
            return e.Shutdown(ctx)   // graceful shutdown
        },
    })

    return e
}
```

### fx Anti-Patterns

1. **Providers returning interfaces.** Always return the concrete struct. Let fx (and Go) resolve interface satisfaction.
2. **`fx.In` structs outside the composition root.** `fx.In` pulls `go.uber.org/fx` into the import graph. Domain and use-case packages must never import DI frameworks.
3. **Business logic inside `fx.Invoke`.** Invoke functions start servers and register routes. They contain no if/switch on domain rules.
4. **Calling `fx.Provide` from outside `main.go` or module definitions.** Providers must be registered at app startup, not conditionally during runtime.
5. **Circular dependencies through fx.** If A needs B and B needs A, fx cannot resolve it. This is a design problem — break the cycle with an interface.
6. **Using fx as a service locator.** Never pass `*fx.App` around and call `app.Get(...)` deep in the call stack. Dependencies flow through constructors, not runtime lookups.
