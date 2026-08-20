// Package httpx 提供 HTTP 响应与请求解析的通用工具。
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// Response 统一 API 响应结构。
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, status, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Code: code, Message: message, Data: data})
}

func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, 0, "ok", data)
}

func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, 0, "ok", data)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, status, code int, message string) {
	JSON(w, status, code, message, nil)
}

func BadRequest(w http.ResponseWriter, message string)    { Error(w, http.StatusBadRequest, 400, message) }
func Unauthorized(w http.ResponseWriter, message string)  { Error(w, http.StatusUnauthorized, 401, message) }
func Forbidden(w http.ResponseWriter, message string)     { Error(w, http.StatusForbidden, 403, message) }
func NotFound(w http.ResponseWriter, message string)      { Error(w, http.StatusNotFound, 404, message) }
func Conflict(w http.ResponseWriter, message string)      { Error(w, http.StatusConflict, 409, message) }
func InternalError(w http.ResponseWriter, message string) { Error(w, http.StatusInternalServerError, 500, message) }

// Decode 解析 JSON 请求体，限制 1MB，且只允许单个 JSON 对象。
func Decode(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("请求体只能包含单个 JSON 对象")
	}
	return nil
}

// Pagination 分页信息。
type Pagination struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

// PageResult 分页查询结果。
type PageResult struct {
	Items      interface{} `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

// PageParams 分页入参。
type PageParams struct {
	Page int
	Size int
}

func ParsePagination(r *http.Request, defaultSize, maxSize int) PageParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}
	return PageParams{Page: page, Size: size}
}
