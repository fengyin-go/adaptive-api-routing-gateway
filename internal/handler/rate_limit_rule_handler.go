package handler

import (
	"net/http"

	"apigateway/internal/model"
	"apigateway/pkg/httpx"
)

func (s *Server) registerRateLimitRuleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/rate-limit-rules", s.createRateLimitRule)
	mux.HandleFunc("GET /api/rate-limit-rules", s.listRateLimitRules)
	mux.HandleFunc("GET /api/rate-limit-rules/{id}", s.getRateLimitRule)
	mux.HandleFunc("PUT /api/rate-limit-rules/{id}", s.updateRateLimitRule)
	mux.HandleFunc("DELETE /api/rate-limit-rules/{id}", s.deleteRateLimitRule)
}

type rateLimitRuleRequest struct {
	Name   string `json:"name"`
	Limit  int    `json:"limit"`
	Window int    `json:"window"`
}

func (s *Server) createRateLimitRule(w http.ResponseWriter, r *http.Request) {
	var req rateLimitRuleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	v, err := s.svc.CreateRateLimitRule(model.RateLimitRule{Name: req.Name, Limit: req.Limit, Window: req.Window})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, v)
}

func (s *Server) listRateLimitRules(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	items, total, err := s.svc.ListRateLimitRules(pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getRateLimitRule(w http.ResponseWriter, r *http.Request) {
	v, err := s.svc.GetRateLimitRule(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, v)
}

func (s *Server) updateRateLimitRule(w http.ResponseWriter, r *http.Request) {
	var req rateLimitRuleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	v, err := s.svc.UpdateRateLimitRule(r.PathValue("id"), model.RateLimitRule{Name: req.Name, Limit: req.Limit, Window: req.Window})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, v)
}

func (s *Server) deleteRateLimitRule(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteRateLimitRule(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
