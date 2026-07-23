package rr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOK(t *testing.T) {
	data := map[string]string{"foo": "bar"}
	resp := OK(data)

	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Msg != "ok" {
		t.Fatalf("msg = %q, want ok", resp.Msg)
	}
	if resp.Data["foo"] != "bar" {
		t.Fatalf("data = %#v", resp.Data)
	}
	if resp.Error != nil {
		t.Fatal("expected nil error")
	}
}

func TestOKMeta(t *testing.T) {
	meta := NewMeta(1, 10, 25)
	resp := OKMeta([]string{"a", "b"}, meta)

	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Meta == nil {
		t.Fatal("expected meta")
	}
	if resp.Meta.Page != 1 || resp.Meta.PerPage != 10 || resp.Meta.TotalCount != 25 {
		t.Fatalf("meta = %#v", resp.Meta)
	}
	if resp.Meta.TotalPages != 3 {
		t.Fatalf("TotalPages = %d, want 3", resp.Meta.TotalPages)
	}
}

func TestFail(t *testing.T) {
	resp := Fail(400, "missing id")
	if resp.Success {
		t.Fatal("expected failure")
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != 400 || resp.Error.Message != "missing id" {
		t.Fatalf("error = %#v", resp.Error)
	}
}

func TestWriteOK(t *testing.T) {
	w := httptest.NewRecorder()
	WriteOK(w, map[string]string{"foo": "bar"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}

	var resp Response[map[string]any]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Data["foo"] != "bar" {
		t.Fatalf("resp = %#v", resp)
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, 40001, "missing id")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}

	var resp Response[any]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Success || resp.Error == nil {
		t.Fatalf("resp = %#v", resp)
	}
	if resp.Error.Code != 40001 || resp.Error.Message != "missing id" {
		t.Fatalf("error = %#v", resp.Error)
	}
}

func TestWriteOKMeta(t *testing.T) {
	w := httptest.NewRecorder()
	meta := NewMeta(2, 10, 42)
	WriteOKMeta(w, []string{"a", "b"}, meta)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp Response[[]string]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Meta == nil || resp.Meta.TotalCount != 42 {
		t.Fatalf("resp = %#v", resp)
	}
	if resp.Meta.TotalPages != 5 {
		t.Fatalf("TotalPages = %d, want 5", resp.Meta.TotalPages)
	}
}

func TestShortcuts(t *testing.T) {
	cases := []struct {
		name   string
		fn     func(http.ResponseWriter, string)
		status int
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized},
		{"Forbidden", Forbidden, http.StatusForbidden},
		{"NotFound", NotFound, http.StatusNotFound},
		{"Conflict", Conflict, http.StatusConflict},
		{"UnprocessableEntity", UnprocessableEntity, http.StatusUnprocessableEntity},
		{"TooManyRequests", TooManyRequests, http.StatusTooManyRequests},
		{"InternalError", InternalError, http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.fn(w, "boom")
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			var resp Response[any]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			if resp.Success || resp.Error == nil || resp.Error.Message != "boom" {
				t.Fatalf("resp = %#v", resp)
			}
			if resp.Error.Code != tc.status {
				t.Fatalf("code = %d, want %d", resp.Error.Code, tc.status)
			}
		})
	}
}

func TestJSON(t *testing.T) {
	b, err := JSON(OK("hello"))
	if err != nil {
		t.Fatal(err)
	}
	var resp Response[string]
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Data != "hello" {
		t.Fatalf("resp = %#v", resp)
	}
}

func TestHandlerIntegration(t *testing.T) {
	// Prove it works as a plain net/http handler — no framework required.
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		WriteOK(w, []map[string]any{{"id": 1, "name": "alice"}})
	})
	mux.HandleFunc("/missing", func(w http.ResponseWriter, r *http.Request) {
		NotFound(w, "user not found")
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/missing", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestOKMsg(t *testing.T) {
	resp := OKMsg("hello", "created")
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Msg != "created" {
		t.Fatalf("msg = %q, want created", resp.Msg)
	}
	if resp.Data != "hello" {
		t.Fatalf("data = %q", resp.Data)
	}
}

func TestOKMsgMeta(t *testing.T) {
	meta := NewMeta(1, 10, 25)
	resp := OKMsgMeta([]string{"a", "b"}, "custom msg", meta)

	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Msg != "custom msg" {
		t.Fatalf("msg = %q", resp.Msg)
	}
	if resp.Meta == nil || resp.Meta.TotalCount != 25 {
		t.Fatal("meta missing or wrong")
	}
	if resp.Meta.TotalPages != 3 {
		t.Fatalf("TotalPages = %d, want 3", resp.Meta.TotalPages)
	}
}

func TestFailData(t *testing.T) {
	resp := FailData(400, "validation error", map[string]string{"field": "name"})
	if resp.Success {
		t.Fatal("expected failure")
	}
	if resp.Error == nil || resp.Error.Code != 400 || resp.Error.Message != "validation error" {
		t.Fatalf("error = %#v", resp.Error)
	}
	if resp.Data["field"] != "name" {
		t.Fatalf("data = %#v", resp.Data)
	}
}

func TestWriteOKMsg(t *testing.T) {
	w := httptest.NewRecorder()
	WriteOKMsg(w, "hello", "created")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp Response[string]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Data != "hello" || resp.Msg != "created" {
		t.Fatalf("resp = %#v", resp)
	}
}

func TestWriteOKMsgMeta(t *testing.T) {
	w := httptest.NewRecorder()
	meta := NewMeta(2, 10, 42)
	WriteOKMsgMeta(w, []string{"a", "b"}, "custom msg", meta)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp Response[[]string]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Msg != "custom msg" || resp.Meta == nil || resp.Meta.TotalCount != 42 {
		t.Fatalf("resp = %#v", resp)
	}
	if resp.Meta.TotalPages != 5 {
		t.Fatalf("TotalPages = %d, want 5", resp.Meta.TotalPages)
	}
}
