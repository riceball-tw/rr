package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/riceball-tw/rr"
)

// ---------------------------------------------------------------------------
// ListUsers
// ---------------------------------------------------------------------------

func TestListUsers(t *testing.T) {
	t.Run("returns first page sorted ascending", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users?page=1&limit=2&sortBy=name&order=asc", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}

		var resp rr.Response[[]user]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if !resp.Success {
			t.Fatalf("expected success, got %#v", resp)
		}
		if len(resp.Data) != 2 {
			t.Fatalf("data len = %d, want 2; data=%#v", len(resp.Data), resp.Data)
		}
		if resp.Data[0].Name != "alice" || resp.Data[1].Name != "bob" {
			t.Fatalf("unexpected order: %#v", resp.Data)
		}
		if resp.Meta == nil || resp.Meta.TotalCount != 3 || resp.Meta.TotalPages != 2 {
			t.Fatalf("meta = %#v", resp.Meta)
		}
	})

	t.Run("returns second page", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users?page=2&limit=2&sortBy=name&order=asc", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp rr.Response[[]user]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Data) != 1 || resp.Data[0].Name != "carol" {
			t.Fatalf("data = %#v", resp.Data)
		}
	})

	t.Run("sorts descending when order=desc", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users?sortBy=name&order=desc", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp rr.Response[[]user]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if !resp.Success {
			t.Fatalf("expected success, got %#v", resp)
		}
		if len(resp.Data) != 3 {
			t.Fatalf("data len = %d, want 3", len(resp.Data))
		}
		if resp.Data[0].Name != "carol" || resp.Data[1].Name != "bob" || resp.Data[2].Name != "alice" {
			t.Fatalf("unexpected desc order: %#v", resp.Data)
		}
	})

	t.Run("falls back to default sort when field is not allowed", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users?sortBy=email&order=asc", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp rr.Response[[]user]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Data) < 2 {
			t.Fatalf("data = %#v", resp.Data)
		}
		if resp.Data[0].ID != 1 || resp.Data[1].ID != 2 {
			t.Fatalf("expected id order, got %#v", resp.Data)
		}
	})

	t.Run("returns empty results when page exceeds total", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users?page=10&limit=2", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp rr.Response[[]user]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if !resp.Success {
			t.Fatalf("expected success, got %#v", resp)
		}
		if len(resp.Data) != 0 {
			t.Fatalf("data len = %d, want 0", len(resp.Data))
		}
		if resp.Meta == nil || resp.Meta.Page != 10 || resp.Meta.PerPage != 2 || resp.Meta.TotalCount != 3 {
			t.Fatalf("meta = %#v", resp.Meta)
		}
	})
}

// ---------------------------------------------------------------------------
// GetUser
// ---------------------------------------------------------------------------

func TestGetUser(t *testing.T) {
	t.Run("returns user by id", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
		var resp rr.Response[user]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if !resp.Success || resp.Data.Name != "alice" {
			t.Fatalf("resp = %#v", resp)
		}
	})

	t.Run("returns 404 for nonexistent id", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d", w.Code)
		}
		var resp rr.Response[any]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Success || resp.Error == nil || resp.Error.Message != "user not found" {
			t.Fatalf("resp = %#v", resp)
		}
	})

	t.Run("returns 400 for invalid id format", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		var resp rr.Response[any]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Error == nil || resp.Error.Message != "invalid user id" {
			t.Fatalf("error = %#v", resp.Error)
		}
	})

	t.Run("returns 400 for zero id", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users/0", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("returns 400 for negative id", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodGet, "/users/-1", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// CreateUser
// ---------------------------------------------------------------------------

func TestCreateUser(t *testing.T) {
	t.Run("creates user and returns success message", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"dave"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
		}
		var resp rr.Response[user]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if !resp.Success || resp.Msg != "created" || resp.Data.Name != "dave" {
			t.Fatalf("resp = %#v", resp)
		}
	})

	t.Run("returns 400 for invalid JSON body", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`not json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		var resp rr.Response[any]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Error == nil || resp.Error.Message != "invalid json body" {
			t.Fatalf("error = %#v", resp.Error)
		}
	})

	t.Run("returns business error for empty name", func(t *testing.T) {
		mux := newMux(newStore())

		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":""}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", w.Code)
		}
		var resp rr.Response[any]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Error == nil || resp.Error.Code != 40001 || resp.Error.Message != "name is required" {
			t.Fatalf("error = %#v", resp.Error)
		}
	})
}
