---
name: rr
description: >-
  Write Go HTTP handlers with github.com/riceball-tw/rr — list-request parsing
  (page/limit/sortBy/order, bounds clamping, sort whitelisting) and the unified
  {success,msg,data,error,meta} JSON envelope. Use when adding, editing, or
  reviewing handlers in a Go project that imports rr, or when the task mentions
  rr.ParseList*, rr.Write*, rr.OK/rr.Fail, pagination metadata, or the rr
  response envelope. Covers net/http, chi, gin, echo, and fiber.
---

# rr

Zero-dependency Go package doing two jobs:

1. **Parse** list query params (`page`, `limit`, `sortBy`, `isDesc`, `order`) with safe defaults and clamping.
2. **Write** one JSON envelope everywhere: `{success, msg, data, error, meta}`.

Import path: `github.com/riceball-tw/rr`. Full signature list: [reference.md](reference.md).

## Canonical list handler

```go
func handleListUsers(w http.ResponseWriter, r *http.Request) {
	list := rr.ParseListFromRequest(r)
	// Whitelist BEFORE the value can reach SQL. Reassign — value receiver.
	sortP := list.SortParams.WithAllowedSort([]string{"id", "name", "created_at"}, "id")

	users, total, err := svc.List(r.Context(), list.GetOffset(), list.GetLimit(),
		sortP.GetSortBy(), sortP.IsDescending())
	if err != nil {
		rr.InternalError(w, "failed to list users")
		return
	}

	rr.WriteOKMeta(w, users, rr.NewMeta(list.GetPage(), list.GetLimit(), total))
}
```

Single-resource handler:

```go
u, ok := svc.Get(id)
if !ok {
	rr.NotFound(w, "user not found")
	return
}
rr.WriteOK(w, u)
```

## Rules

1. **Reassign the `With*` result.** `WithAllowedSort` and `WithMaxLimit` are value receivers returning a copy. `list.SortParams.WithAllowedSort(...)` on its own line does nothing.
2. **Bind the parse result to a variable first.** Getters have pointer receivers, so `rr.ParseList(q).GetPage()` is a compile error (`cannot call pointer method GetPage on rr.ListParams`). Assign, then call.
3. **Always whitelist `sortBy`** before it reaches a query builder — it is raw client input. `GetSortBy()` without `AllowedSortBy` returns the raw string.
4. **`GetSkip()` / `GetOffset()` return `int64`.** Cast for slice indexing: `int(list.GetSkip())`.
5. **`WriteError(w, httpStatus, bodyCode, msg)`** — HTTP status first, app code second. Shortcuts (`BadRequest`, `NotFound`, …) set both to the same value; use `WriteError` when the business code differs (`400` + `40001`).
6. **Write once per request.** `rr.Write*` sets `Content-Type`, calls `WriteHeader`, and writes the body. Always `return` after it. Never call `w.WriteHeader` first.
7. **`rr.Config` is startup-only.** Mutating it after serving starts is a data race.
8. **No `WriteCreated`.** For non-200: `rr.Write(w, http.StatusCreated, rr.OKMsg(u, "created"))`.
9. **Pass real `total_count`** into `NewMeta` — the DB count, not `len(page)`, or `total_pages` is wrong.

## Gotchas

Verified behavior — do not guess around these.

| Input | Result | Why |
|---|---|---|
| `data` is an empty or nil slice | **`data` key absent from JSON** | `Data T` has `omitempty`. Clients doing `res.data.map()` break. Wrap in a named struct (`struct{ Items []U \`json:"items"\` }`) when the key must always exist. |
| `total_count` is `0` | `total_count` and `total_pages` absent from `meta` | `omitempty` on every `Meta` field. |
| `?isDesc=false&order=desc` | **descending** | `IsDescending()` is `IsDesc \|\| order∈{desc,descending,d}`. `order` can't be cancelled by `isDesc=false`. Send one param, not both. |
| `sortBy` not in whitelist, `DefaultSortBy` empty | `GetSortBy()` returns `""` | Pass a non-empty default unless the caller handles empty. |
| `?limit=99999` | clamped to `MaxLimit` (100) | Per-request override: `list.PaginationParams.WithMaxLimit(500)`. |
| `?page=-3` / `?page=abc` | `1` | Junk falls back to defaults; never errors. |
| `data` is `0` / `""` / `false` | `data` key absent | Same `omitempty`. Non-issue for structs. |
| `json.Marshal` fails | 500 + `{"success":false,"error":{"code":500,...}}` | `Write` never emits a truncated body. |

Defaults: `page=1`, `limit=10`, `MaxLimit=100`. Query keys are configurable via `rr.Config`.

## Frameworks

`rr.Write*` needs an `http.ResponseWriter`. Anything else uses the pure builders (`rr.OK`, `rr.OKMeta`, `rr.Fail`) with the framework's own renderer.

**net/http, chi, gorilla/mux** — writers directly, as above.

**gin** — either style; don't mix in one handler.

```go
// A: bind via struct tags (rr.ListParams carries form/json/query tags)
type listUsersReq struct{ rr.ListParams }

func listUsers(c *gin.Context) {
	var req listUsersReq
	_ = c.ShouldBindQuery(&req) // binding errors leave zero values -> safe defaults
	sortP := req.SortParams.WithAllowedSort([]string{"id", "name"}, "id")
	...
	rr.WriteOKMeta(c.Writer, users, rr.NewMeta(req.GetPage(), req.GetLimit(), total))
}

// B: builder + gin renderer
c.JSON(http.StatusNotFound, rr.Fail(http.StatusNotFound, "user not found"))
```

Or skip binding entirely: `rr.ParseListFromRequest(c.Request)`.

**echo** — `c.Response()` is an `http.ResponseWriter`: `rr.WriteOK(c.Response(), data)`. Or `c.JSON(200, rr.OK(data))`.

**fiber** — fasthttp, no `http.ResponseWriter`. Use builders: `c.Status(fiber.StatusNotFound).JSON(rr.Fail(404, "not found"))`.

## Choosing a helper

| Need | Call |
|---|---|
| 200 + data | `rr.WriteOK(w, data)` |
| 200 + data + custom msg | `rr.WriteOKMsg(w, data, "created")` |
| 200 + paged data | `rr.WriteOKMeta(w, items, rr.NewMeta(page, limit, total))` |
| Any other status | `rr.Write(w, status, rr.OKMsg(data, msg))` |
| Standard HTTP error | `rr.BadRequest/Unauthorized/Forbidden/NotFound/Conflict/UnprocessableEntity/TooManyRequests/InternalError(w, msg)` |
| Error with business code | `rr.WriteError(w, http.StatusBadRequest, 40001, msg)` |
| Envelope as a value (non-`http.ResponseWriter` renderer, tests) | `rr.OK` / `rr.OKMeta` / `rr.Fail` / `rr.FailData`, or `rr.JSON(resp)` for bytes |

## Before finishing

- [ ] Every `sortBy` that reaches a query is whitelisted, and the `With*` result was reassigned.
- [ ] `return` after every `rr.Write*`; exactly one write per path.
- [ ] `NewMeta` gets the total row count, not the page length.
- [ ] `int(...)` around `GetSkip()` where an `int` is required.
- [ ] Empty-result responses are acceptable without a `data` key (or the payload is wrapped).
- [ ] `go vet ./... && go test ./...` clean.
