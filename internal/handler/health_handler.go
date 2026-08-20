package handler

import (
	"net/http"

	"apigateway/pkg/httpx"
)

func (s *Server) registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.listHealthChecks)
	mux.HandleFunc("POST /api/health/{serviceId}/probe", s.probeService)
	mux.HandleFunc("GET /api/health/{serviceId}", s.getHealthCheck)
}

func (s *Server) listHealthChecks(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	items, total, err := s.svc.ListHealthChecks(pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getHealthCheck(w http.ResponseWriter, r *http.Request) {
	hc, err := s.svc.GetHealthCheck(r.PathValue("serviceId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, hc)
}

func (s *Server) probeService(w http.ResponseWriter, r *http.Request) {
	hc, err := s.svc.ProbeService(r.PathValue("serviceId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, hc)
}
