package handler

import (
	"net/http"

	"apigateway/internal/model"
	"apigateway/pkg/httpx"
)

func (s *Server) registerExportRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/export", s.exportConfig)
	mux.HandleFunc("POST /api/import/services", s.importServices)
	mux.HandleFunc("POST /api/import/rate-limit-rules", s.importRateLimitRules)
}

func (s *Server) exportConfig(w http.ResponseWriter, r *http.Request) {
	snapshot := s.svc.ExportConfig()
	httpx.OK(w, snapshot)
}

type importServicesRequest struct {
	Services []model.Service `json:"services"`
}

func (s *Server) importServices(w http.ResponseWriter, r *http.Request) {
	var req importServicesRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	created, errs := s.svc.ImportServices(req.Services)
	httpx.OK(w, map[string]interface{}{
		"created": created,
		"errors":  errs,
	})
}

type importRulesRequest struct {
	Rules []model.RateLimitRule `json:"rules"`
}

func (s *Server) importRateLimitRules(w http.ResponseWriter, r *http.Request) {
	var req importRulesRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	created, errs := s.svc.ImportRateLimitRules(req.Rules)
	httpx.OK(w, map[string]interface{}{
		"created": created,
		"errors":  errs,
	})
}
