# Skill: Building an Adele Framework Package

This skill documents how to build, structure, and ship a first-party package for the [Adele framework](https://github.com/Cidekar/adele-framework). Use this as the blueprint when starting a new package repo under Cidekar.

> **NOTE TO MAINTAINERS:** the sections below up to and including `## Consumer Integration` are the **existing skill content**. The new sections start at `## First-Party Packages` and continue through the end of the document. When committing upstream, replace this file with the existing content + the new sections.

## References

- **pkg.go.dev** (auto-generated from source — ground truth for signatures):
  [adele-framework](https://pkg.go.dev/github.com/cidekar/adele-framework),
  [adele-queue](https://pkg.go.dev/github.com/cidekar/adele-queue),
  [adele-oauth2](https://pkg.go.dev/github.com/cidekar/adele-oauth2).
  Pin the version with `@<version>` (e.g. `@v1.0.3`); default URL serves latest.
  Offline equivalent: `go doc github.com/cidekar/adele-framework <Symbol>`.
- [Provider doc](https://github.com/Cidekar/adele-documentation/blob/main/guides/provider.md)
- [OAuth doc](https://github.com/Cidekar/adele-documentation/blob/main/guides/oauth.md)
- [Adele framework source](https://github.com/Cidekar/adele-framework)

---

(... existing sections preserved verbatim: Package Structure, The Provider Pattern, Provider Implementation Template, Barrel Export Pattern, Scaffold Templates, Module Naming, Makefile, GitHub Actions, Release Flow, Security Checklist, `.nancy-ignore` Format, Consumer Integration ...)

---

## First-Party Packages

The Cidekar org ships two production packages today: `adele-queue` and `adele-oauth2`. Both follow the blueprint above. Their public API surface and version pins are documented below so consumer code does not need to read the source to wire them.

Pin to:

| Package | Version | Why |
|---|---|---|
| `github.com/cidekar/adele-framework` | v1.0.3 | Stable router/Mux, BootstrapMux middleware chain, Helpers.Render |
| `github.com/cidekar/adele-queue` | v1.0.2 | Adds `(*ServiceProvider).Service()` accessor |
| `github.com/cidekar/adele-oauth2` | v1.0.5 | Adds `(*ServiceProvider).Service()` accessor — required to mount middleware on a non-default subrouter |

Toolchain: Go 1.25.0+. Both packages declare `go 1.25.0`.

---

## adele-queue

Repo: `github.com/cidekar/adele-queue`. Provider name: `"queue"`. Priority: 30 (core-services tier).
Live API reference: [pkg.go.dev/github.com/cidekar/adele-queue](https://pkg.go.dev/github.com/cidekar/adele-queue).

### Wiring

```go
import (
    _ "github.com/cidekar/adele-queue"             // blank import auto-registers provider
    queue "github.com/cidekar/adele-queue"
    "github.com/cidekar/adele-framework/provider"
)

// Configure BEFORE provider.LoadProviders runs.
a.Provider.SetProviderConfig("queue", map[string]interface{}{
    "backend":               "redis",              // "memory" | "redis"
    "worker_count":          4,
    "max_attempts":          5,
    "high_water_mark":       10000,
    "queue_channels":        []string{"default"},
    "queue_channel_default": "default",
    "redis_prefix":          "myapp",
    "redis_scan_interval":   1,
    "lock_timeout":          300,
    "reaper_interval":       30,
})

// LoadProviders calls Register on every blank-imported provider.
// adele-queue's Register constructs the queue. Boot starts the worker pool + reaper.
_ = a.Provider.LoadProviders(app.App)

// Resolve the live queue handle:
var q *queue.Queue
for _, sp := range provider.GetRegisteredProviders() {
    if sp.Name() == "queue" {
        if qp, ok := sp.(*queue.ServiceProvider); ok {
            q = qp.Service()
        }
    }
}
```

### API surface

```go
// Constructors
func New(a *adele.Adele) (*Queue, error)
func NewWithConfig(a *adele.Adele, config Configuration) (*Queue, error)

// Lifecycle
func (q *Queue) Listen()                                    // start workers + reaper goroutine
func (q *Queue) Close(mWG *sync.WaitGroup)                  // shutdown; mWG may be nil

// Dispatch
func (q *Queue) Dispatch(job Job) (string, error)           // returns job UUID
func (q *Queue) DispatchIn(job Job, delay time.Duration) (string, error)

// Handler registration
func (q *Queue) RegisterHandler(name string, fn func(payload interface{}) error) error
func (q *Queue) RegisterHandlerCtx(name string, fn func(ctx context.Context, payload interface{}) error) error

// Introspection
func (q *Queue) Depth() (int, error)                        // pending count
func (q *Queue) GetFailedJobs() (*[]Job, error)             // from failed_jobs table
func (q *Queue) GetCompletedJobs() (*[]Job, error)
func (q *Queue) ReaperStats() ReaperStats                   // {Ticks, ScannedKeys, Requeued, Permafailed, ...}
func (q *Queue) UnmarshalPayload(cachedJob []byte) (*Job, error)
```

`Job` is a concrete struct (not an interface):

```go
type Job struct {
    ID             string                          `db:"job_id" json:"id"`
    Handler        func(payload interface{}) error `redis:"-"`
    Name           string
    Payload        []byte
    Retry          bool
    RetryInSeconds int
    RetryCounter   int                             `db:"attempts"`
    DispatchAt     string
    Queue          string
    LockFor        int
    Exception      string
    Status         string
    // ... created_at, updated_at, completed_at, failed_at, locked_at
}
```

### Backends

- `memory` — channel-based; `Depth()` reads `q.pendingCount` (atomic). Single-process only.
- `redis` — keyspace `queues:<channel>:<state>:<job-id>`; states `pending`, `locked`, `completed`, `failed`. Workers `SCAN` pending keys and `RENAME` to `locked` to claim. Stale-lock reaper requeues jobs whose `LockedAt` exceeds `LockTimeout`.

The redis pool is taken from `a.Cache` (must be `*redisdriver.RedisCache`). The queue does not configure Redis itself.

### Migrations

- Postgres: `migrations/queue_tables.postgres.sql` — creates `jobs`, `failed_jobs`. Apply via `adele migrate up` after copying into your app's `migrations/` dir.
- MySQL: not shipped.

### Key rules

- Pin to v1.0.2+. Earlier versions don't expose `(*ServiceProvider).Service()`.
- Configure via `SetProviderConfig` BEFORE `LoadProviders`. Configuration is applied in `Register`.
- Set `a.Cache` to a Redis cache before setting `backend: redis`. The queue grabs its pool from `a.Cache`. If `a.Cache` is Badger or nil, redis dispatch silently fails.
- The README's `q := queue.New(app)` example is wrong. Real signature is `(*Queue, error)`. Always handle the error.
- The queue dispatches handlers **in-process**. The `RPC_SERVER_PORT` / `RPC_PORT` env vars are NOT consumed despite docs claims. Treat the docs paragraph on RPC dispatch as stale.
- Configuration value types must match: `int` keys won't accept `int64` or `float64` from YAML decoders that promote numbers. The `Configure` method silently zero-defaults type-mismatched keys.
- Scope keys must match `^[a-zA-Z0-9-]+$` — applies to `oauth` config, not queue, but the same regex strictness is shared across config-load.
- The package-level `wg sync.WaitGroup` and `systemShutdown bool` make `*Queue` effectively a singleton per process. Don't construct two.

---

## adele-oauth2

Repo: `github.com/cidekar/adele-oauth2`. Provider name: **`"oauth"`** (not `"oauth2"`). Priority: 51 (security tier).
Live API reference: [pkg.go.dev/github.com/cidekar/adele-oauth2](https://pkg.go.dev/github.com/cidekar/adele-oauth2)
([api subpackage](https://pkg.go.dev/github.com/cidekar/adele-oauth2/api) is where the bearer middleware + Service type live).

### Wiring

```go
import (
    _ "github.com/cidekar/adele-oauth2"
    adeleoauth2 "github.com/cidekar/adele-oauth2"
    api "github.com/cidekar/adele-oauth2/api"
    "github.com/cidekar/adele-framework/provider"
)

// Configure BEFORE LoadProviders.
a.Provider.SetProviderConfig("oauth", map[string]interface{}{
    "guarded_route_groups": []string{"/api"},
    "unguarded_routes":     []string{"/api/health", "/api/ping"},
    "scopes": map[string]string{
        "user-create": "Permission to create users",
        "user-read":   "Permission to read users",
    },
})

_ = a.Provider.LoadProviders(app.App)

// Resolve the live service:
var oauthSvc *api.Service
for _, sp := range provider.GetRegisteredProviders() {
    if sp.Name() == "oauth" {
        if op, ok := sp.(*adeleoauth2.ServiceProvider); ok {
            oauthSvc = op.Service()
        }
    }
}
```

`Register` auto-mounts on `a.Routes`:

| Method | Path | Purpose |
|---|---|---|
| POST | `/oauth/token` | Access-token grant exchange |
| POST | `/oauth/token/refresh` | Refresh-token exchange |
| GET | `/oauth/authorize` | Authorization request (renders consent) |
| POST | `/oauth/authorize` | Authorization grant exchange |
| GET | `/api/ping` | Test endpoint for bearer-middleware validation |

`Boot` is a no-op.

### Bearer middleware

```go
// Mount on protected route groups:
r.Use(oauthSvc.AuthenticationTokenMiddleware())
r.Use(oauthSvc.AuthenticationCheckForScopes())

// Per-route scope annotation:
r.Post("/api/users[scopes:user-create]", a.Handlers.CreateUser)
```

After successful validation the middleware stamps:

```go
api.ContextKeyClientID    contextKey = {name: "adele-oauth2:client-id"}    // int
api.ContextKeyClientName  contextKey = {name: "adele-oauth2:client-name"}  // string
api.ContextKeyAccessToken contextKey = {name: "adele-oauth2:access-token"} // string

// Plus a legacy string key (one major release of backward compat):
"accessToken"  // string — read by ScopeHandler
```

### Service API

```go
// Token operations
func (o *Service) AuthenticateToken(r *http.Request) (bool, *OauthToken, error)
func (o *Service) GetByToken(plainText string) (*OauthToken, error)        // sha256 lookup against tokens.token_hash (bytea)
func (o *Service) GetAuthTokenFromHeader(r *http.Request) (*OauthToken, error)
func (o *Service) GenerateOauthToken() (*OauthToken, error)                // 16 random bytes → base32 unpadded → 26-char plaintext
func (o *Service) InsertOauthToken(token *OauthToken) error
func (o *Service) DeleteOauthToken(id int) error

// Refresh tokens (DIFFERENT scheme: base64-encoded sha1 of plaintext, not sha256)
func (o *Service) GenerateRefreshToken(userID, accessTokenID, clientID int) (*RefreshToken, error)
func (o *Service) GetRefreshByToken(plainText string) (*RefreshToken, error)
func (o *Service) DeleteRefreshTokenByToken(plainText string) error

// Authorization codes (PKCE flow)
func (o *Service) GenerateAuthorizationToken() (*AuthorizationToken, error)
func (o *Service) GetAuthorizationTokenByToken(plainText string) (*AuthorizationToken, error)
func (o *Service) ConsumeAuthorizationToken(plainText string) (*AuthorizationToken, error) // atomic find+delete in tx
```

### Configuration

`config/oauth.yml` is auto-seeded from the embedded default on first run. Top-level keys:

```yaml
GuardedRouteGroups: []      # []string — routes whose paths *contain* one of these strings get bearer-checked
UnguardedRoutes: []         # []string — exact-match path bypass
AuthorizationTokenTTL: 60m
OauthTokenTTL: 24h
RefreshTokenTokenTTL: 24h
PkceImplicitTTL: 300s
Scopes: {}                  # map[string]string; keys MUST match ^[a-zA-Z0-9-]+$ or panic
PkceImplicitAuthorizationScopes: {}  # same regex constraint
VerifyTemplatePath: ""
```

### Migrations

Apply BOTH for postgres:

```
migrations/oauth_tables.postgres.sql   # creates oauth_clients, tokens, refresh_tokens, authorization_tokens
migrations/add_flow_column.sql         # adds oauth_clients.flow column — REQUIRED
```

Schema highlights — every token table stores `token_hash bytea NOT NULL` (no plaintext column). `oauth_clients.flow` is `character varying(255) NOT NULL DEFAULT ''`; valid values are `plain`, `pkce`, `pkce-implicit`.

### Key rules

- Pin to v1.0.5+. Earlier versions lack `(*ServiceProvider).Service()`.
- `GuardedRouteGroups` MUST be non-empty when `AuthenticationTokenMiddleware()` is mounted. **An empty list returns HTTP 500 on every request, including healthchecks.** This is the most common bootstrap trap.
- `GuardedRouteGroups` uses `strings.Contains`, NOT `strings.HasPrefix`. `/api` matches `/foo/api/bar`. Either accept this or fork the middleware.
- Apply BOTH migrations. Without `add_flow_column.sql`, every authorization_code client silently returns `unsupported_grant_type`.
- Scope keys MUST match `^[a-zA-Z0-9-]+$`. Underscores, colons, dots all panic at config-load. Use hyphens.
- Provider name is `"oauth"`. `GetRegisteredProviders()` returns by `Name()`. `"oauth2"` will never match.
- Postgres only. The MySQL migration is broken as shipped (uses `bytea`, declares `token NOT NULL` columns the Go code doesn't write to).
- The legacy string context key `"accessToken"` is what `ScopeHandler` reads. The typed `ContextKeyAccessToken` is also stamped today, but if you write your own scope checker it must read the legacy key for one more major version.
- Access-token plaintext is hardcoded to 26 characters (base32-unpadded of 16 bytes). `GetAuthTokenFromHeader` rejects any other length. Migrating from a pre-existing token store will not work without a re-issue cycle.
- Refresh tokens use a DIFFERENT scheme: plaintext is base64-encoded SHA-1 string of `(userID, randomBytes)`. `GetRefreshByToken` decodes the plaintext as base64 and queries the decoded bytes against `token_hash`. Don't apply access-token sha256 logic to refresh tokens.

---

## Cross-package compatibility

Both first-party packages share these transitive deps at the recommended pins:

| Dep | Version |
|---|---|
| `github.com/upper/db/v4` | v4.10.0 |
| `github.com/gomodule/redigo` | v1.9.2 |
| `github.com/alexedwards/scs/v2` | v2.9.0 |
| `github.com/go-chi/chi/v5` | v5.2.5 |

No module-graph conflicts at the recommended pins. If a downstream consumer pins differently and forces a downgrade of `upper/db`, expect query-builder API drift in adele-oauth2's `up.Cond{"token_hash": ...}` paths.

---

## What's NOT in the first-party set

The starter-kit (`adele install starter-kit`) ships form-based session auth via the **aerra** scaffold. It is NOT OAuth2. The two are independent paths:

| Need | Use |
|---|---|
| Login forms, session cookies, `/login` `/registration` views | `adele install starter-kit --with-auth` |
| Bearer tokens, OAuth client_credentials, scoped APIs, third-party SSO | `adele-oauth2` package + manual wiring |

There is no `adele install starter-kit --with-oauth` — adele-oauth2 is a separate `go get` and `_` import.

---

## Consumer integration checklist

For consumers wiring the first-party packages, in order:

- [ ] `go get github.com/cidekar/adele-framework@v1.0.3`
- [ ] `go get github.com/cidekar/adele-queue@v1.0.2` (if queueing)
- [ ] `go get github.com/cidekar/adele-oauth2@v1.0.5` (if OAuth)
- [ ] Blank-import each in the binary's main package (or any imported package — `init()` runs once globally)
- [ ] `a.Provider.SetProviderConfig(name, ...)` BEFORE `LoadProviders`
- [ ] Configure `a.Cache` as Redis if queue backend is `redis`
- [ ] Apply both adele-oauth2 migrations to Postgres (NOT MySQL)
- [ ] `GuardedRouteGroups` non-empty before mounting `AuthenticationTokenMiddleware`
- [ ] Scope keys match `^[a-zA-Z0-9-]+$`
- [ ] Run `go test -race ./...` — both packages have package-level state (queue's `wg`/`systemShutdown`, oauth's provider registry) that can surface ordering bugs

If you skip any of these, expect the failure modes documented in the per-package "Key rules" sections above.
