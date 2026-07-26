# rr API reference

Complete exported surface of `github.com/riceball-tw/rr`. Stdlib only.

## Request

### Types

```go
type PaginationParams struct {
	Page     int `form:"page" json:"page" query:"page"`
	Limit    int `form:"limit" json:"limit" query:"limit"`
	MaxLimit int `form:"-" json:"-" query:"-"` // 0 = use Config.MaxLimit
}

type SortParams struct {
	SortBy        string   `form:"sortBy" json:"sortBy" query:"sortBy"`
	IsDesc        bool     `form:"isDesc" json:"isDesc" query:"isDesc"`
	Order         string   `form:"order"  json:"order"  query:"order"` // "asc"|"desc"|"descending"|"d"
	AllowedSortBy []string `form:"-" json:"-" query:"-"`               // empty = allow anything
	DefaultSortBy string   `form:"-" json:"-" query:"-"`
}

type ListParams struct {
	PaginationParams
	SortParams
}
```

`form`/`json`/`query` tags mean these bind directly in gin (`ShouldBindQuery`), echo (`Bind`), and similar. Embed them in your own DTO to add filters:

```go
type ListUsersReq struct {
	rr.ListParams
	Status string `form:"status" json:"status" query:"status"`
}
```

### Getters — pointer receivers

`GetPage`, `GetLimit`, `GetSkip`, `GetOffset`, `GetSortBy`, `GetOrder`, `IsDescending` all have pointer receivers, so the value must be **addressable**. Chaining off a function call is a compile error:

```go
page := rr.ParseList(q).GetPage()  // cannot call pointer method GetPage on rr.ListParams
list := rr.ParseList(q); page := list.GetPage()  // OK
```

| Method | Returns | Behavior |
|---|---|---|
| `GetPage()` | `int` | `Page <= 0` → `Config.DefaultPage` (1) |
| `GetLimit()` | `int` | `Limit <= 0` → `Config.DefaultLimit` (10); clamped to `MaxLimit` field, else `Config.MaxLimit` (100) |
| `GetSkip()` | `int64` | `(GetPage()-1) * GetLimit()` |
| `GetOffset()` | `int64` | alias of `GetSkip()` |
| `GetSortBy()` | `string` | if `AllowedSortBy` non-empty and `SortBy` not in it → `DefaultSortBy`; empty `SortBy` → `DefaultSortBy` |
| `GetOrder()` | `string` | `"desc"` if `IsDesc` or `Order` ∈ {desc, descending, d} (case-insensitive, trimmed); else `"asc"` |
| `IsDescending()` | `bool` | same predicate as `GetOrder() == "desc"` |

### Builders — value receivers, return copies

```go
func (p PaginationParams) WithMaxLimit(max int) PaginationParams
func (p SortParams) WithAllowedSort(allowed []string, defaultSort string) SortParams
```

Must be reassigned: `sortP := list.SortParams.WithAllowedSort(...)`.

### Parsers

```go
func ParsePagination(q url.Values) PaginationParams
func ParseSort(q url.Values) SortParams
func ParseList(q url.Values) ListParams

func ParsePaginationFromRequest(r *http.Request) PaginationParams
func ParseSortFromRequest(r *http.Request) SortParams
func ParseListFromRequest(r *http.Request) ListParams
```

Never return an error — unparseable input falls back to defaults. Ints via `strconv.Atoi` on the trimmed value. `isDesc` truthy set: `1 t true yes y on` (case-insensitive); everything else is false.

Precedence inside `ParseSort`: if `isDesc` is present it sets the `IsDesc` field; otherwise `order` does. But `Order` is *also* stored raw, and `IsDescending()` ORs the two — so `?isDesc=false&order=desc` is descending.

### Config

```go
const (
	DefaultPage  = 1
	DefaultLimit = 10
	DefaultMax   = 100
)

var Config = struct {
	DefaultPage  int    // 1
	DefaultLimit int    // 10
	MaxLimit     int    // 100
	PageKey      string // "page"
	LimitKey     string // "limit"
	SortByKey    string // "sortBy"
	IsDescKey    string // "isDesc"
	OrderKey     string // "order"
}{...}
```

Global and unsynchronized — set in `init`/`main` before serving. Rename query keys here for snake_case APIs (`Config.SortByKey = "sort_by"`).

## Response

### Types

```go
type Response[T any] struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg,omitempty"`
	Data    T      `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`    // HTTP status or business code — your convention
	Message string `json:"message"`
}

type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	TotalCount int64 `json:"total_count,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

func NewMeta(page, perPage int, totalCount int64) *Meta // TotalPages = ceil(total/perPage) when perPage > 0
```

`omitempty` on `Data` means empty slices, `0`, `""`, `false`, and `nil` drop the `data` key entirely. Empty structs still render (`"data":{}`).

### Builders (no I/O)

```go
func OK[T any](data T) Response[T]                                  // success, msg "ok"
func OKMsg[T any](data T, msg string) Response[T]
func OKMeta[T any](data T, meta *Meta) Response[T]                  // success, msg "ok"
func OKMsgMeta[T any](data T, msg string, meta *Meta) Response[T]
func Fail(code int, message string) Response[any]
func FailData[T any](code int, message string, data T) Response[T]  // failure carrying partial data
func JSON[T any](resp Response[T]) ([]byte, error)
```

### Writers

```go
func Write[T any](w http.ResponseWriter, status int, resp Response[T])
func WriteOK[T any](w http.ResponseWriter, data T)                                        // 200
func WriteOKMsg[T any](w http.ResponseWriter, data T, msg string)                         // 200
func WriteOKMeta[T any](w http.ResponseWriter, data T, meta *Meta)                        // 200
func WriteOKMsgMeta[T any](w http.ResponseWriter, data T, msg string, meta *Meta)         // 200
func WriteError(w http.ResponseWriter, status, code int, message string)
```

`Write` sets `Content-Type: application/json; charset=utf-8`, marshals, then `WriteHeader(status)` + body. If `json.Marshal` fails it emits `500` with `{"success":false,"error":{"code":500,"message":"response encoding failed"}}` instead of a partial body. No return value — write errors are not surfaced.

### Error shortcuts

Each sets the body `code` equal to the HTTP status.

| Func | Status |
|---|---|
| `BadRequest(w, msg)` | 400 |
| `Unauthorized(w, msg)` | 401 |
| `Forbidden(w, msg)` | 403 |
| `NotFound(w, msg)` | 404 |
| `Conflict(w, msg)` | 409 |
| `UnprocessableEntity(w, msg)` | 422 |
| `TooManyRequests(w, msg)` | 429 |
| `InternalError(w, msg)` | 500 |

Any other status, or a body code that differs from the status: `WriteError(w, status, code, msg)`.

## Wire format

```jsonc
// success
{"success":true,"msg":"ok","data":[{"id":1}],"meta":{"page":1,"per_page":10,"total_count":42,"total_pages":5}}

// success, zero rows — note the absent "data", "total_count", "total_pages"
{"success":true,"msg":"ok","meta":{"page":1,"per_page":10}}

// failure
{"success":false,"error":{"code":40001,"message":"name is required"}}
```
