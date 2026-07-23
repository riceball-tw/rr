# rr

`rr` is a small, framework-agnostic Go package for HTTP request parsing and consistent JSON responses. It works with `net/http` and any framework that uses `http.ResponseWriter`.

## Install

```bash
go get github.com/riceball-tw/rr
```

```go
import "github.com/riceball-tw/rr"
```

## Quick start

```go
list := rr.ParseListFromRequest(r)
page, limit := list.GetPage(), list.GetLimit()

users, total := service.List(list.GetOffset(), limit)
rr.WriteOKMeta(w, users, rr.NewMeta(page, limit, total))
```

Use `GetOffset()` when your data store requires an offset. To restrict client-provided sort fields:

```go
sortBy := list.SortParams.WithAllowedSort(
	[]string{"name", "created_at"}, "created_at",
).GetSortBy()
```

For errors:

```go
rr.WriteError(w, http.StatusBadRequest, 40001, "invalid request")
```

## Response format

```json
{
  "success": true,
  "msg": "ok",
  "data": {},
  "meta": { "page": 1, "per_page": 10, "total_count": 42, "total_pages": 5 }
}
```

Error responses use `"success": false` and an `error` object containing an application-defined `code` and `message`.

## Defaults

Set package defaults during application startup:

```go
rr.Config.DefaultLimit = 20
rr.Config.MaxLimit = 100
```

Query parameters: `page`, `limit`, `sortBy`, `isDesc`, and `order`.

## Development

```bash
go test ./...
```

Runnable examples are available in `examples/nethttp` and `examples/gin`.

## License

MIT
