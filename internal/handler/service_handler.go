package handler

import (
	"net/http"

	"apigateway/internal/model"
	"apigateway/pkg/httpx"
)

func (s *Server) registerServiceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/services", s.createService)
	mux.HandleFunc("GET /api/services", s.listServices)
	mux.HandleFunc("GET /api/services/{id}", s.getService)
	mux.HandleFunc("PUT /api/services/{id}", s.updateService)
	mux.HandleFunc("DELETE /api/services/{id}", s.deleteService)
}

type serviceRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Timeout int    `json:"timeout"`
	Retries int    `json:"retries"`
	Status  string `json:"status"`
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	v, err := s.svc.CreateService(model.Service{
		Name: req.Name, BaseURL: req.BaseURL, Timeout: req.Timeout, Retries: req.Retries, Status: req.Status,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, v)
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.ServiceFilter{
		Status:  r.URL.Query().Get("status"),
		Keyword: r.URL.Query().Get("keyword"),
	}
	items, total, err := s.svc.ListServices(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	v, err := s.svc.GetService(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, v)
}

func (s *Server) updateService(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	v, err := s.svc.UpdateService(r.PathValue("id"), model.Service{
		Name: req.Name, BaseURL: req.BaseURL, Timeout: req.Timeout, Retries: req.Retries, Status: req.Status,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, v)
}

func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteService(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
