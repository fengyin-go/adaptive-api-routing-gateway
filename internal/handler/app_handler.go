package handler

import (
	"net/http"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/httpx"
)

func (s *Server) registerAppRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/apps", s.createApp)
	mux.HandleFunc("GET /api/apps", s.listApps)
	mux.HandleFunc("GET /api/apps/{id}", s.getApp)
	mux.HandleFunc("PUT /api/apps/{id}", s.updateApp)
	mux.HandleFunc("DELETE /api/apps/{id}", s.deleteApp)
}

type appRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (s *Server) createApp(w http.ResponseWriter, r *http.Request) {
	var req appRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	a, err := s.svc.CreateApp(model.App{Name: req.Name, Status: req.Status})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, a)
}

func (s *Server) listApps(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.AppFilter{Status: r.URL.Query().Get("status")}
	items, total, err := s.svc.ListApps(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getApp(w http.ResponseWriter, r *http.Request) {
	a, err := s.svc.GetApp(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, a)
}

func (s *Server) updateApp(w http.ResponseWriter, r *http.Request) {
	var req appRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	a, err := s.svc.UpdateApp(r.PathValue("id"), model.App{Name: req.Name, Status: req.Status})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, a)
}

func (s *Server) deleteApp(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteApp(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (s *Server) registerAPIKeyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/api-keys", s.createAPIKey)
	mux.HandleFunc("GET /api/api-keys", s.listAPIKeys)
	mux.HandleFunc("GET /api/api-keys/{id}", s.getAPIKey)
	mux.HandleFunc("PATCH /api/api-keys/{id}/status", s.updateAPIKeyStatus)
	mux.HandleFunc("DELETE /api/api-keys/{id}", s.deleteAPIKey)
}

type apiKeyRequest struct {
	AppID      string `json:"app_id"`
	Secret     string `json:"secret"`
	ExpiresAt  string `json:"expires_at"`
}

func (s *Server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req apiKeyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	var expiresAt time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			httpx.BadRequest(w, "expires_at 格式应为 RFC3339")
			return
		}
		expiresAt = t
	}
	k, err := s.svc.CreateAPIKey(req.AppID, req.Secret, expiresAt)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, k)
}

func (s *Server) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	items, total, err := s.svc.ListAPIKeys(r.URL.Query().Get("app_id"), pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getAPIKey(w http.ResponseWriter, r *http.Request) {
	k, err := s.svc.GetAPIKey(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, k)
}

type apiKeyStatusRequest struct {
	Status string `json:"status"`
}

func (s *Server) updateAPIKeyStatus(w http.ResponseWriter, r *http.Request) {
	var req apiKeyStatusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	k, err := s.svc.UpdateAPIKeyStatus(r.PathValue("id"), req.Status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, k)
}

func (s *Server) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteAPIKey(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
