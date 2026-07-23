package rr

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Default pagination values. Override via Config or per-call options.
const (
	DefaultPage  = 1
	DefaultLimit = 10
	DefaultMax   = 100
)

// Config holds package-level defaults. Safe to mutate at startup only.
var Config = struct {
	DefaultPage  int
	DefaultLimit int
	MaxLimit     int
	// Query key names used by Parse* helpers.
	PageKey   string
	LimitKey  string
	SortByKey string
	IsDescKey string
	// OrderKey is an alternative to IsDescKey: "asc" / "desc".
	OrderKey string
}{
	DefaultPage:  DefaultPage,
	DefaultLimit: DefaultLimit,
	MaxLimit:     DefaultMax,
	PageKey:      "page",
	LimitKey:     "limit",
	SortByKey:    "sortBy",
	IsDescKey:    "isDesc",
	OrderKey:     "order",
}

// PaginationParams is a reusable page/limit pair for list endpoints.
//
// Embed into your own request DTOs:
//
//	type ListUsersReq struct {
//	    rr.PaginationParams
//	    Status string `form:"status" json:"status" query:"status"`
//	}
type PaginationParams struct {
	Page  int `form:"page" json:"page" query:"page"`
	Limit int `form:"limit" json:"limit" query:"limit"`
	// MaxLimit caps GetLimit(). Zero means use Config.MaxLimit.
	MaxLimit int `form:"-" json:"-" query:"-"`
}

// GetPage returns a safe page number (>= 1).
func (p *PaginationParams) GetPage() int {
	if p.Page <= 0 {
		return Config.DefaultPage
	}
	return p.Page
}

// GetLimit returns a safe page size, clamped to MaxLimit.
func (p *PaginationParams) GetLimit() int {
	limit := p.Limit
	if limit <= 0 {
		limit = Config.DefaultLimit
	}
	max := p.MaxLimit
	if max <= 0 {
		max = Config.MaxLimit
	}
	if max > 0 && limit > max {
		return max
	}
	return limit
}

// GetSkip returns the offset for SQL OFFSET / Mongo skip style queries.
func (p *PaginationParams) GetSkip() int64 {
	return int64((p.GetPage() - 1) * p.GetLimit())
}

// GetOffset is an alias of GetSkip for naming conventions that prefer "offset".
func (p *PaginationParams) GetOffset() int64 {
	return p.GetSkip()
}

// SortParams is a reusable sort-by / order pair for list endpoints.
type SortParams struct {
	// SortBy is the client-requested sort field.
	SortBy string `form:"sortBy" json:"sortBy" query:"sortBy"`
	// IsDesc requests descending order when true.
	IsDesc bool `form:"isDesc" json:"isDesc" query:"isDesc"`
	// AllowedSortBy restricts SortBy to a whitelist. Empty = allow any.
	AllowedSortBy []string `form:"-" json:"-" query:"-"`
	// DefaultSortBy is used when SortBy is empty or not allowed.
	DefaultSortBy string `form:"-" json:"-" query:"-"`
}

// GetSortBy returns a validated sort field (or DefaultSortBy / empty).
func (p *SortParams) GetSortBy() string {
	sortBy := p.SortBy

	if len(p.AllowedSortBy) > 0 {
		found := false
		for _, v := range p.AllowedSortBy {
			if v == sortBy {
				found = true
				break
			}
		}
		if !found {
			sortBy = ""
		}
	}

	if sortBy == "" {
		sortBy = p.DefaultSortBy
	}
	return sortBy
}

// GetOrder returns "asc" or "desc".
func (p *SortParams) GetOrder() string {
	if p.IsDesc {
		return "desc"
	}
	return "asc"
}

// IsDescending reports whether sort order is descending.
func (p *SortParams) IsDescending() bool {
	return p.IsDesc
}

// ListParams combines pagination and sorting for typical list endpoints.
type ListParams struct {
	PaginationParams
	SortParams
}

// ---------------------------------------------------------------------------
// Parsing helpers (stdlib only)
// ---------------------------------------------------------------------------

// ParsePagination reads page/limit from url.Values (query or form).
func ParsePagination(q url.Values) PaginationParams {
	return PaginationParams{
		Page:  queryInt(q, Config.PageKey, 0),
		Limit: queryInt(q, Config.LimitKey, 0),
	}
}

// ParseSort reads sortBy / isDesc (or order) from url.Values.
//
// Supported inputs for direction:
//   - isDesc=true|1|yes
//   - order=desc|asc  (case-insensitive)
func ParseSort(q url.Values) SortParams {
	p := SortParams{
		SortBy: strings.TrimSpace(q.Get(Config.SortByKey)),
	}

	if raw := q.Get(Config.IsDescKey); raw != "" {
		p.IsDesc = parseBool(raw)
	} else if order := strings.ToLower(strings.TrimSpace(q.Get(Config.OrderKey))); order != "" {
		p.IsDesc = order == "desc" || order == "descending" || order == "d"
	}

	return p
}

// ParseList reads both pagination and sort params from url.Values.
func ParseList(q url.Values) ListParams {
	return ListParams{
		PaginationParams: ParsePagination(q),
		SortParams:       ParseSort(q),
	}
}

// ParseListFromRequest reads list params from r.URL.Query().
func ParseListFromRequest(r *http.Request) ListParams {
	return ParseList(r.URL.Query())
}

// ParsePaginationFromRequest reads pagination from r.URL.Query().
func ParsePaginationFromRequest(r *http.Request) PaginationParams {
	return ParsePagination(r.URL.Query())
}

// ParseSortFromRequest reads sort params from r.URL.Query().
func ParseSortFromRequest(r *http.Request) SortParams {
	return ParseSort(r.URL.Query())
}

// WithAllowedSort returns a copy of p with AllowedSortBy / DefaultSortBy set.
// Useful after ParseSort:
//
//	sort := rr.ParseSort(q).WithAllowedSort([]string{"name","created_at"}, "created_at")
func (p SortParams) WithAllowedSort(allowed []string, defaultSort string) SortParams {
	p.AllowedSortBy = allowed
	p.DefaultSortBy = defaultSort
	return p
}

// WithMaxLimit returns a copy of p with MaxLimit set.
func (p PaginationParams) WithMaxLimit(max int) PaginationParams {
	p.MaxLimit = max
	return p
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

func queryInt(q url.Values, key string, fallback int) int {
	raw := strings.TrimSpace(q.Get(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
