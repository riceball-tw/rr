# rr — request & response

A tiny, **framework-agnostic** Go package for unified HTTP request params and response envelopes.

Works with the standard library and any framework that speaks `http.ResponseWriter` / `*http.Request` (net/http, chi, gorilla/mux, gin, echo, fiber via adaptor, …).

## Install

```bash
go get github.com/riceball-tw/rr
```

```go
import "github.com/riceball-tw/rr"
```

## Layout

Idiomatic single-module library layout: one public package at the module root
(`package rr`), tests co-located, CI under `.github/workflows/`.

```
rr/
├── go.mod
├── doc.go              # package docs
├── request.go          # pagination, sort, query parsing
├── request_test.go
├── response.go         # envelope + HTTP writers
├── response_test.go
├── examples/
│   ├── nethttp/        # runnable + unit-tested net/http demo
│   └── gin/            # separate module (gin dep stays out of rr)
├── README.md
└── .github/workflows/ci.yml
```

## Demos (runnable + unit-tested)

Small “users API” demos exercise list pagination/sort, get-by-id, create, and
error envelopes. They are ordinary Go tests (`httptest`), so `go test` is the demo.

```bash
# stdlib net/http (part of this module — no extra deps)
go test ./examples/nethttp/ -v
go run  ./examples/nethttp/          # listen :8080

# gin (own go.mod so rr stays framework-free)
cd examples/gin && go test . -v
cd examples/gin && go run .          # listen :8081
```

## Response envelope

```json
// success
{
  "success": true,
  "msg": "ok",
  "data": { "id": 1 },
  "meta": { "page": 1, "per_page": 10, "total_count": 42, "total_pages": 5 }
}

// error
{
  "success": false,
  "error": { "code": 400, "message": "invalid id" }
}
```

### Pure builders (no I/O)

```go
import "github.com/riceball-tw/rr"

resp := rr.OK(user)
resp = rr.OKMsg(user, "created")
resp = rr.OKMeta(users, rr.NewMeta(page, limit, total))
resp = rr.Fail(400, "invalid id")
```

### Write to any `http.ResponseWriter`

```go
rr.WriteOK(w, user)
rr.WriteOKMeta(w, users, rr.NewMeta(1, 10, total))
rr.WriteError(w, http.StatusBadRequest, 40001, "invalid id")

// shortcuts
rr.BadRequest(w, "missing field")
rr.Unauthorized(w, "login required")
rr.Forbidden(w, "no access")
rr.NotFound(w, "user not found")
rr.Conflict(w, "already exists")
rr.UnprocessableEntity(w, "validation failed")
rr.TooManyRequests(w, "slow down")
rr.InternalError(w, "unexpected error")
```

## Request helpers

### Embed in your DTOs

```go
import "github.com/riceball-tw/rr"

type ListUsersReq struct {
    rr.PaginationParams
    rr.SortParams
    Status string `form:"status" json:"status" query:"status"`
}

// Or use the combined type:
type ListUsersReq struct {
    rr.ListParams
    Status string `form:"status" json:"status" query:"status"`
}
```

```go
page  := req.GetPage()   // >= 1
limit := req.GetLimit()  // default 10, clamped by MaxLimit
skip  := req.GetSkip()   // (page-1)*limit  — also GetOffset()
sort  := req.GetSortBy() // whitelist-aware
order := req.GetOrder()  // "asc" | "desc"
```

### Parse from query (stdlib)

```go
list := rr.ParseListFromRequest(r)
// or
list := rr.ParseList(r.URL.Query())

page := list.GetPage()
sort := list.SortParams.WithAllowedSort(
    []string{"name", "created_at"},
    "created_at",
)
```

Query keys (configurable via `rr.Config`):

| Key      | Meaning                          |
|----------|----------------------------------|
| `page`   | page number                      |
| `limit`  | page size                        |
| `sortBy` | sort field                       |
| `isDesc` | `true` / `1` / `yes` for desc    |
| `order`  | alternative: `asc` / `desc`      |

### Defaults & limits

```go
rr.Config.DefaultPage  = 1
rr.Config.DefaultLimit = 20
rr.Config.MaxLimit     = 100
// rr.Config.PageKey = "p"  // rename query keys if needed
```

Per-request max:

```go
p := rr.ParsePagination(q).WithMaxLimit(50)
```

## Framework examples

### net/http / chi

```go
http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
    list := rr.ParseListFromRequest(r)
    users, total := svc.List(list.GetSkip(), list.GetLimit())
    rr.WriteOKMeta(w, users, rr.NewMeta(list.GetPage(), list.GetLimit(), total))
})
```

### gin

```go
r.GET("/users", func(c *gin.Context) {
    var req rr.ListParams
    _ = c.ShouldBindQuery(&req)

    users, total := svc.List(req.GetSkip(), req.GetLimit())
    // gin's c.Writer is an http.ResponseWriter
    rr.WriteOKMeta(c.Writer, users, rr.NewMeta(req.GetPage(), req.GetLimit(), total))
})
```

### echo

```go
e.GET("/users", func(c echo.Context) error {
    var req rr.ListParams
    _ = c.Bind(&req)

    users, total := svc.List(req.GetSkip(), req.GetLimit())
    rr.WriteOKMeta(c.Response().Writer, users, rr.NewMeta(req.GetPage(), req.GetLimit(), total))
    return nil
})
```

### When you only need the struct (custom render)

```go
payload := rr.OK(user)
c.JSON(http.StatusOK, payload) // gin / echo / fiber style
```

## Design notes

- **Zero framework deps** — only `net/http`, `encoding/json`, `net/url`.
- **Generics** for typed `data` without `interface{}` casting.
- **HTTP status** and **body error code** are separate in `WriteError`, so you can return business codes (`40001`) under HTTP 400.
- Struct tags use `form` / `json` / `query` so the same DTO binds across popular frameworks.
- **Single public package** at the module root — short import path, package name matches the module base name.

## License

MIT
