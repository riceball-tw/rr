// Command nethttp is a small demo API built with net/http + rr.
//
// Run:
//
//	go run ./examples/nethttp
//
// Then try:
//
//	curl 'http://localhost:8080/users?page=1&limit=2&sortBy=name&order=asc'
//	curl 'http://localhost:8080/users/1'
//	curl -X POST http://localhost:8080/users -d '{"name":"dave"}' -H 'Content-Type: application/json'
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/riceball-tw/rr"
)

type user struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// store is a tiny in-memory user store for the demo.
type store struct {
	mu    sync.RWMutex
	seq   int
	users map[int]user
}

func newStore() *store {
	s := &store{users: make(map[int]user), seq: 3}
	s.users[1] = user{ID: 1, Name: "alice"}
	s.users[2] = user{ID: 2, Name: "bob"}
	s.users[3] = user{ID: 3, Name: "carol"}
	return s
}

func (s *store) list() []user {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]user, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

func (s *store) get(id int) (user, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *store) create(name string) user {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	u := user{ID: s.seq, Name: name}
	s.users[u.ID] = u
	return u
}

// newMux wires demo routes. Exported for unit tests via httptest.
func newMux(s *store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", handleListUsers(s))
	mux.HandleFunc("GET /users/{id}", handleGetUser(s))
	mux.HandleFunc("POST /users", handleCreateUser(s))
	return mux
}

func handleListUsers(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list := rr.ParseListFromRequest(r)
		// Whitelist sort fields after parsing query params.
		sortP := list.SortParams.WithAllowedSort([]string{"id", "name"}, "id")

		all := s.list()
		sortUsers(all, sortP.GetSortBy(), sortP.IsDescending())

		page := list.GetPage()
		limit := list.GetLimit()
		skip := int(list.GetSkip())
		total := int64(len(all))

		if skip > len(all) {
			skip = len(all)
		}
		end := skip + limit
		if end > len(all) {
			end = len(all)
		}
		pageItems := all[skip:end]

		rr.WriteOKMeta(w, pageItems, rr.NewMeta(page, limit, total))
	}
}

func handleGetUser(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id <= 0 {
			rr.BadRequest(w, "invalid user id")
			return
		}
		u, ok := s.get(id)
		if !ok {
			rr.NotFound(w, "user not found")
			return
		}
		rr.WriteOK(w, u)
	}
}

func handleCreateUser(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			rr.BadRequest(w, "invalid json body")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			// Business code (40001) can differ from HTTP status (400).
			rr.WriteError(w, http.StatusBadRequest, 40001, "name is required")
			return
		}
		u := s.create(name)
		rr.WriteOKMsg(w, u, "created")
	}
}

func sortUsers(users []user, by string, desc bool) {
	sort.SliceStable(users, func(i, j int) bool {
		var less bool
		switch by {
		case "name":
			less = users[i].Name < users[j].Name
		default: // id
			less = users[i].ID < users[j].ID
		}
		if desc {
			return !less
		}
		return less
	})
}

func main() {
	mux := newMux(newStore())
	addr := ":8080"
	log.Printf("rr net/http demo listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
