package rr

import (
	"encoding/json"
	"net/http"
)

// Response is the unified API response envelope.
//
// Example success:
//
//	{"success":true,"msg":"ok","data":{...},"meta":{...}}
//
// Example error:
//
//	{"success":false,"error":{"code":400,"message":"invalid id"}}
type Response[T any] struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg,omitempty"`
	Data    T      `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

// Error holds application-level error details inside the envelope.
// Code is free-form (HTTP status, business code, or both — your convention).
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Meta holds optional response metadata such as pagination.
type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	TotalCount int64 `json:"total_count,omitempty"`
	// TotalPages is computed when both PerPage and TotalCount are set via NewMeta.
	TotalPages int `json:"total_pages,omitempty"`
}

// NewMeta builds pagination metadata and fills TotalPages when possible.
func NewMeta(page, perPage int, totalCount int64) *Meta {
	m := &Meta{
		Page:       page,
		PerPage:    perPage,
		TotalCount: totalCount,
	}
	if perPage > 0 {
		m.TotalPages = int((totalCount + int64(perPage) - 1) / int64(perPage))
	}
	return m
}

// ---------------------------------------------------------------------------
// Pure builders (no I/O) — use these when you need the value, not to write HTTP.
// ---------------------------------------------------------------------------

// OK builds a successful response with data.
func OK[T any](data T) Response[T] {
	return Response[T]{
		Success: true,
		Msg:     "ok",
		Data:    data,
	}
}

// OKMsg builds a successful response with a custom message.
func OKMsg[T any](data T, msg string) Response[T] {
	return Response[T]{
		Success: true,
		Msg:     msg,
		Data:    data,
	}
}

// OKMeta builds a successful response with data and metadata.
func OKMeta[T any](data T, meta *Meta) Response[T] {
	return Response[T]{
		Success: true,
		Msg:     "ok",
		Data:    data,
		Meta:    meta,
	}
}

// Fail builds a failed response with an error payload.
func Fail(code int, message string) Response[any] {
	return Response[any]{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// FailData builds a failed response that also carries partial data.
func FailData[T any](code int, message string, data T) Response[T] {
	return Response[T]{
		Success: false,
		Data:    data,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// ---------------------------------------------------------------------------
// HTTP writers — encode JSON to any http.ResponseWriter.
// ---------------------------------------------------------------------------

// Write encodes resp as JSON with the given HTTP status code.
func Write[T any](w http.ResponseWriter, status int, resp Response[T]) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// WriteOK writes a 200 success response.
func WriteOK[T any](w http.ResponseWriter, data T) {
	Write(w, http.StatusOK, OK(data))
}

// WriteOKMsg writes a 200 success response with a custom message.
func WriteOKMsg[T any](w http.ResponseWriter, data T, msg string) {
	Write(w, http.StatusOK, OKMsg(data, msg))
}

// WriteOKMeta writes a 200 success response with metadata.
func WriteOKMeta[T any](w http.ResponseWriter, data T, meta *Meta) {
	Write(w, http.StatusOK, OKMeta(data, meta))
}

// WriteError writes a failed response with the given HTTP status and error body.
// status is the HTTP status code; code is the application error code in the body.
func WriteError(w http.ResponseWriter, status, code int, message string) {
	Write(w, status, Fail(code, message))
}

// ---------------------------------------------------------------------------
// Common HTTP error shortcuts (status == body code by default).
// ---------------------------------------------------------------------------

// BadRequest writes 400.
func BadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, http.StatusBadRequest, message)
}

// Unauthorized writes 401.
func Unauthorized(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnauthorized, http.StatusUnauthorized, message)
}

// Forbidden writes 403.
func Forbidden(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, http.StatusForbidden, message)
}

// NotFound writes 404.
func NotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, http.StatusNotFound, message)
}

// Conflict writes 409.
func Conflict(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, http.StatusConflict, message)
}

// UnprocessableEntity writes 422.
func UnprocessableEntity(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, message)
}

// TooManyRequests writes 429.
func TooManyRequests(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusTooManyRequests, http.StatusTooManyRequests, message)
}

// InternalError writes 500.
func InternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, http.StatusInternalServerError, message)
}

// ---------------------------------------------------------------------------
// JSON helper for frameworks that only need the encoded bytes.
// ---------------------------------------------------------------------------

// JSON returns the response as JSON bytes (useful for tests or custom writers).
func JSON[T any](resp Response[T]) ([]byte, error) {
	return json.Marshal(resp)
}
