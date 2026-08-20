package handler

import (
	"net/http"

	"apigateway/pkg/httpx"
)

func (s *Server) registerStatsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/stats", s.getStats)
}

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.svc.GetGatewayStats()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, stats)
}
