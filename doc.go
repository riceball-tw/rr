// Package rr provides unified, framework-agnostic HTTP request helpers and
// response envelopes.
//
// Request helpers cover list-endpoint concerns (pagination, sorting) and work
// from url.Values or *http.Request. Struct tags use standard form / json /
// query keys so the same types bind in gin, echo, fiber, and similar
// frameworks.
//
// Response types and builders have no framework dependency. Write helpers use
// the standard library's http.ResponseWriter, so they plug into net/http, chi,
// gorilla/mux, gin, echo, fiber (via adaptor), and any other HTTP stack.
package rr
