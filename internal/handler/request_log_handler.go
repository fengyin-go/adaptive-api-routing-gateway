package handler

import (
	"net/http"
	"strconv"

	"apigateway/internal/model"
	"apigateway/pkg/httpx"
)

func (s *Server) registerRequestLogRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/logs", s.listRequestLogs)
	mux.HandleFunc("GET /api/logs/{id}", s.getRequestLog)
}

func (s *Server) listRequestLogs(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	statusCode := 0
	if v := r.URL.Query().Get("status_code"); v != "" {
		statusCode, _ = strconv.Atoi(v)
	}
	filter := model.RequestLogFilter{
		RouteID:    r.URL.Query().Get("route_id"),
		AppID:      r.URL.Query().Get("app_id"),
		StatusCode: statusCode,
	}
	items, total, err := s.svc.ListRequestLogs(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getRequestLog(w http.ResponseWriter, r *http.Request) {
	l, err := s.svc.GetRequestLog(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, l)
}
