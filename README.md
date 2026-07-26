# rr

Small, zero dependencies, framework-agnostic package for HTTP request parsing and consistent JSON responses. rr handles the boilerplate every endpoint repeats:
- computing query offset
- clamping bounds
- whitelisting sort columns
- building the JSON response

## Before vs after

```go
// Without rr — 30+ lines of manual parsing, bounds checking, and response building
func handleListUsers(w http.ResponseWriter, r *http.Request) {
    page := 1
    if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
        page = p
    }
    limit := 10
    if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
        if l > 100 { l = 100 }
        limit = l
    }
    sortBy := r.URL.Query().Get("sortBy")
    order := r.URL.Query().Get("order")
    isDesc := strings.ToLower(order) == "desc"

    allowedSorts := []string{"name", "created_at"}
    sortValid := false
    for _, s := range allowedSorts { if s == sortBy { sortValid = true; break } }
    if !sortValid { sortBy = "created_at" }

    users, total := service.List((page-1)*limit, limit, sortBy, isDesc)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true, "msg": "ok", "data": users,
        "meta": map[string]interface{}{
            "page": page, "per_page": limit,
            "total_count": total, "total_pages": int((total+int64(limit)-1)/int64(limit)),
        },
    })
}
```

```go
// net/http With rr — 4 lines
func handleListUsers(w http.ResponseWriter, r *http.Request) {
    list := rr.ParseListFromRequest(r)

    users, total := service.List(
        list.GetOffset(), 
        list.GetLimit(), 
        list.GetSortBy(), 
        list.IsDescending()
    )
    
    rr.WriteOKMeta(w, users, 
    rr.NewMeta(
        list.GetPage(), 
        list.GetLimit(), 
        total
        )
    )
}
```


```go
// Gin With rr — 3 lines
func handleListUsers(c *gin.Context) {
    list := rr.ParseListFromRequest(c.Request)
    users, total := service.List(list.GetOffset(), list.GetLimit(), list.GetSortBy(), list.IsDescending())
    rr.WriteOKMeta(c.Writer, users, rr.NewMeta(list.GetPage(), list.GetLimit(), total))
}
```

## Quick start

```go
import "github.com/riceball-tw/rr"

// Parse pagination + sorting from request
list := rr.ParseListFromRequest(r)

// Restrict sort fields to a whitelist
sortP := list.SortParams.WithAllowedSort([]string{"name", "created_at"}, "created_at")

// Apply to your query
users, total := service.List(list.GetOffset(), list.GetLimit(), sortP.GetSortBy(), sortP.IsDescending())

// Write consistent JSON response
rr.WriteOKMeta(w, users, rr.NewMeta(list.GetPage(), list.GetLimit(), total))
```

For errors:

```go
// HTTP 400 with business code 40001
rr.WriteError(w, http.StatusBadRequest, 40001, "name is required")

// Or use a shortcut for common HTTP errors
rr.NotFound(w, "user not found")
```

## Install

```bash
go get github.com/riceball-tw/rr
```

## Why rr?

- **~50 lines → ~4 lines.** No manual query parsing or slicing, or response building.
- **Safe by default.** `?limit=9999999` is clamped to `Config.MaxLimit`. Unknown sort fields fall back to your default. Zeroes/negatives become 1.
- **Consistent envelope.** Every response is `{success, msg, data, error, meta}`. Error payloads carry an app-level code plus HTTP status.
- **Marshal safe.** If `json.Marshal` panics, `rr` writes a proper 500 instead of a truncated body.
- **Framework agnostic.** Works with `net/http`, `chi`, `gin`, `echo`, `fiber` (via adaptor), and anything accepting `http.ResponseWriter`.
- **Zero dependencies.** Stdlib only.
- **Agent ready.** Ships a Claude Code skill so AI-written handlers match the intended patterns — see [Use with Claude Code](#use-with-claude-code).

## Response format

```json
{
  "success": true,
  "msg": "ok",
  "data": {},
  "meta": { "page": 1, "per_page": 10, "total_count": 42, "total_pages": 5 }
}
```

Error responses use `"success": false` and an `error` object:

```json
{
  "success": false,
  "error": { "code": 40001, "message": "name is required" }
}
```

## Defaults

Page size defaults to 10 and is capped at 100. Override at startup:

```go
rr.Config.DefaultLimit = 20  // default page size
rr.Config.MaxLimit = 500     // hard cap
```

Query parameters parsed: `page`, `limit`, `sortBy`, `isDesc`, `order`. Their names are configurable too (`rr.Config.SortByKey = "sort_by"`).

## Use with Claude Code

rr ships an [Agent Skill](https://docs.claude.com/en/docs/claude-code/skills) so Claude writes rr handlers correctly instead of guessing: the canonical list-handler shape, framework recipes for net/http, gin, echo and fiber, and the sharp edges — `omitempty` dropping the `data` key on empty pages, pointer-receiver getters that can't be chained, `With*` returning copies.

Install it into any project that imports rr:

```bash
mkdir -p .claude/skills/rr
curl -sL -o .claude/skills/rr/SKILL.md     https://raw.githubusercontent.com/riceball-tw/rr/main/.claude/skills/rr/SKILL.md
curl -sL -o .claude/skills/rr/reference.md https://raw.githubusercontent.com/riceball-tw/rr/main/.claude/skills/rr/reference.md
```

Claude picks it up automatically when a task touches list endpoints or JSON responses. Use `~/.claude/skills/rr/` instead to install it for every project.

## Development

```bash
go test ./...
```

Runnable examples in `examples/nethttp` and `examples/gin`. The Claude Code skill lives in `.claude/skills/rr/`.

## License

MIT
