package handler

import (
	"net/http"

	"apigateway/internal/model"
	"apigateway/pkg/httpx"
)

func (s *Server) registerRouteRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/routes", s.createRoute)
	mux.HandleFunc("GET /api/routes", s.listRoutes)
	mux.HandleFunc("GET /api/routes/{id}", s.getRoute)
	mux.HandleFunc("PUT /api/routes/{id}", s.updateRoute)
	mux.HandleFunc("DELETE /api/routes/{id}", s.deleteRoute)
}

type routeRequest struct {
	Path            string `json:"path"`
	Method          string `json:"method"`
	ServiceID       string `json:"service_id"`
	StripPrefix     bool   `json:"strip_prefix"`
	RequiresAuth    bool   `json:"requires_auth"`
	RateLimitRuleID string `json:"rate_limit_rule_id"`
}

func (s *Server) createRoute(w http.ResponseWriter, r *http.Request) {
	var req routeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	v, err := s.svc.CreateRoute(model.Route{
		Path: req.Path, Method: req.Method, ServiceID: req.ServiceID,
		StripPrefix: req.StripPrefix, RequiresAuth: req.RequiresAuth, RateLimitRuleID: req.RateLimitRuleID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, v)
}

func (s *Server) listRoutes(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.RouteFilter{
		ServiceID: r.URL.Query().Get("service_id"),
		Method:    r.URL.Query().Get("method"),
	}
	items, total, err := s.svc.ListRoutes(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getRoute(w http.ResponseWriter, r *http.Request) {
	v, err := s.svc.GetRoute(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, v)
}

func (s *Server) updateRoute(w http.ResponseWriter, r *http.Request) {
	var req routeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	v, err := s.svc.UpdateRoute(r.PathValue("id"), model.Route{
		Path: req.Path, Method: req.Method, ServiceID: req.ServiceID,
		StripPrefix: req.StripPrefix, RequiresAuth: req.RequiresAuth, RateLimitRuleID: req.RateLimitRuleID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, v)
}

func (s *Server) deleteRoute(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteRoute(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
