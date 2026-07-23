package rr

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPaginationParams(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		p := PaginationParams{}
		if p.GetPage() != 1 {
			t.Fatalf("page = %d", p.GetPage())
		}
		if p.GetLimit() != 10 {
			t.Fatalf("limit = %d", p.GetLimit())
		}
		if p.GetSkip() != 0 {
			t.Fatalf("skip = %d", p.GetSkip())
		}
		if p.GetOffset() != p.GetSkip() {
			t.Fatal("offset should equal skip")
		}
	})

	t.Run("custom", func(t *testing.T) {
		p := PaginationParams{Page: 2, Limit: 20}
		if p.GetPage() != 2 || p.GetLimit() != 20 || p.GetSkip() != 20 {
			t.Fatalf("p = page=%d limit=%d skip=%d", p.GetPage(), p.GetLimit(), p.GetSkip())
		}
	})

	t.Run("invalid falls back", func(t *testing.T) {
		p := PaginationParams{Page: -1, Limit: 0}
		if p.GetPage() != 1 || p.GetLimit() != 10 {
			t.Fatalf("page=%d limit=%d", p.GetPage(), p.GetLimit())
		}
	})

	t.Run("max limit clamp", func(t *testing.T) {
		p := PaginationParams{Page: 1, Limit: 500, MaxLimit: 50}
		if p.GetLimit() != 50 {
			t.Fatalf("limit = %d, want 50", p.GetLimit())
		}
	})

	t.Run("config max limit", func(t *testing.T) {
		old := Config.MaxLimit
		Config.MaxLimit = 30
		defer func() { Config.MaxLimit = old }()

		p := PaginationParams{Limit: 999}
		if p.GetLimit() != 30 {
			t.Fatalf("limit = %d, want 30", p.GetLimit())
		}
	})

	t.Run("WithMaxLimit", func(t *testing.T) {
		p := PaginationParams{Limit: 200}.WithMaxLimit(25)
		if p.GetLimit() != 25 {
			t.Fatalf("limit = %d", p.GetLimit())
		}
	})
}

func TestSortParams(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		p := SortParams{}
		if p.GetSortBy() != "" {
			t.Fatalf("sortBy = %q", p.GetSortBy())
		}
		if p.GetOrder() != "asc" {
			t.Fatalf("order = %q", p.GetOrder())
		}
		if p.IsDescending() {
			t.Fatal("expected ascending")
		}
	})

	t.Run("desc", func(t *testing.T) {
		p := SortParams{SortBy: "created_at", IsDesc: true}
		if p.GetSortBy() != "created_at" || p.GetOrder() != "desc" {
			t.Fatalf("sortBy=%q order=%q", p.GetSortBy(), p.GetOrder())
		}
	})

	t.Run("allowed field", func(t *testing.T) {
		p := SortParams{
			SortBy:        "price",
			AllowedSortBy: []string{"created_at", "price", "name"},
			DefaultSortBy: "created_at",
		}
		if p.GetSortBy() != "price" {
			t.Fatalf("sortBy = %q", p.GetSortBy())
		}
	})

	t.Run("disallowed falls back", func(t *testing.T) {
		p := SortParams{
			SortBy:        "invalid_field",
			AllowedSortBy: []string{"created_at", "price"},
			DefaultSortBy: "created_at",
		}
		if p.GetSortBy() != "created_at" {
			t.Fatalf("sortBy = %q", p.GetSortBy())
		}
	})

	t.Run("WithAllowedSort", func(t *testing.T) {
		p := SortParams{SortBy: "name"}.WithAllowedSort([]string{"name", "age"}, "age")
		if p.GetSortBy() != "name" {
			t.Fatalf("sortBy = %q", p.GetSortBy())
		}
		p2 := SortParams{SortBy: "hack"}.WithAllowedSort([]string{"name"}, "name")
		if p2.GetSortBy() != "name" {
			t.Fatalf("sortBy = %q", p2.GetSortBy())
		}
	})
}

func TestParsePagination(t *testing.T) {
	q := url.Values{}
	q.Set("page", "3")
	q.Set("limit", "25")

	p := ParsePagination(q)
	if p.GetPage() != 3 || p.GetLimit() != 25 {
		t.Fatalf("page=%d limit=%d", p.GetPage(), p.GetLimit())
	}

	// empty / garbage
	q2 := url.Values{}
	q2.Set("page", "abc")
	p2 := ParsePagination(q2)
	if p2.GetPage() != 1 {
		t.Fatalf("page = %d", p2.GetPage())
	}
}

func TestParseSort(t *testing.T) {
	t.Run("isDesc", func(t *testing.T) {
		q := url.Values{}
		q.Set("sortBy", "name")
		q.Set("isDesc", "true")
		p := ParseSort(q)
		if p.SortBy != "name" || !p.IsDesc {
			t.Fatalf("%#v", p)
		}
	})

	t.Run("order=desc", func(t *testing.T) {
		q := url.Values{}
		q.Set("sortBy", "created_at")
		q.Set("order", "DESC")
		p := ParseSort(q)
		if !p.IsDesc || p.GetOrder() != "desc" {
			t.Fatalf("%#v", p)
		}
	})

	t.Run("order=asc", func(t *testing.T) {
		q := url.Values{}
		q.Set("order", "asc")
		p := ParseSort(q)
		if p.IsDesc {
			t.Fatalf("%#v", p)
		}
	})
}

func TestParseListFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?page=2&limit=15&sortBy=price&order=desc", nil)
	list := ParseListFromRequest(req)

	if list.GetPage() != 2 || list.GetLimit() != 15 {
		t.Fatalf("page=%d limit=%d", list.GetPage(), list.GetLimit())
	}
	if list.GetSortBy() != "price" || list.GetOrder() != "desc" {
		t.Fatalf("sortBy=%q order=%q", list.GetSortBy(), list.GetOrder())
	}
}

func TestListParamsEmbed(t *testing.T) {
	// Simulate a user DTO embedding ListParams.
	type ListUsersReq struct {
		ListParams
		Status string
	}

	req := ListUsersReq{
		ListParams: ListParams{
			PaginationParams: PaginationParams{Page: 1, Limit: 10},
			SortParams:       SortParams{SortBy: "name", AllowedSortBy: []string{"name", "id"}, DefaultSortBy: "id"},
		},
		Status: "active",
	}

	if req.GetPage() != 1 || req.GetSortBy() != "name" || req.Status != "active" {
		t.Fatalf("%#v", req)
	}
}
