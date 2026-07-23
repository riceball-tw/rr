// Command gin-demo is a small demo API built with gin + rr.
//
// Gin is intentionally a dependency of this example only — the rr library
// itself stays framework-free. gin's ResponseWriter is an http.ResponseWriter,
// so rr.Write* helpers plug in directly; for binding you can also build the
// envelope with rr.OK / rr.Fail and call c.JSON.
//
// Run (from this directory):
//
//	go run .
//
// Then try:
//
//	curl 'http://localhost:8081/users?page=1&limit=2&sortBy=name&order=asc'
//	curl 'http://localhost:8081/users/1'
//	curl -X POST http://localhost:8081/users -d '{"name":"dave"}' -H 'Content-Type: application/json'
package main

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/riceball-tw/rr"
)

type user struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

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

// listUsersReq embeds rr.ListParams so gin can bind page/limit/sortBy/isDesc.
type listUsersReq struct {
	rr.ListParams
}

// newEngine wires demo routes. Safe to call from unit tests via httptest.
func newEngine(s *store) *gin.Engine {
	r := gin.New()

	r.GET("/users", func(c *gin.Context) {
		var req listUsersReq
		// form/json/query tags on rr.ListParams bind with ShouldBindQuery.
		_ = c.ShouldBindQuery(&req)
		sortP := req.SortParams.WithAllowedSort([]string{"id", "name"}, "id")

		all := s.list()
		sortUsers(all, sortP.GetSortBy(), sortP.IsDescending())

		page := req.GetPage()
		limit := req.GetLimit()
		skip := int(req.GetSkip())
		total := int64(len(all))

		if skip > len(all) {
			skip = len(all)
		}
		end := skip + limit
		if end > len(all) {
			end = len(all)
		}

		// Style A: write through c.Writer (http.ResponseWriter).
		rr.WriteOKMeta(c.Writer, all[skip:end], rr.NewMeta(page, limit, total))
	})

	r.GET("/users/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			rr.BadRequest(c.Writer, "invalid user id")
			return
		}
		u, ok := s.get(id)
		if !ok {
			// Style B: build envelope, let gin render JSON.
			c.JSON(http.StatusNotFound, rr.Fail(http.StatusNotFound, "user not found"))
			return
		}
		c.JSON(http.StatusOK, rr.OK(u))
	})

	r.POST("/users", func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			rr.BadRequest(c.Writer, "invalid json body")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			rr.WriteError(c.Writer, http.StatusBadRequest, 40001, "name is required")
			return
		}
		u := s.create(name)
		// Custom message via pure builder + gin renderer.
		c.JSON(http.StatusOK, rr.OKMsg(u, "created"))
	})

	return r
}

func sortUsers(users []user, by string, desc bool) {
	sort.SliceStable(users, func(i, j int) bool {
		var less bool
		switch by {
		case "name":
			less = users[i].Name < users[j].Name
		default:
			less = users[i].ID < users[j].ID
		}
		if desc {
			return !less
		}
		return less
	})
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	addr := ":8081"
	if err := newEngine(newStore()).Run(addr); err != nil {
		panic(err)
	}
}
