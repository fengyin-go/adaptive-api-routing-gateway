package flowmodel

import (
	"context"
	"errors"
)

var (
	ErrRejected  = errors.New("gateway request rejected")
	ErrTemporary = errors.New("temporary upstream failure")
)

type Batch struct {
	Tenant string
	Items  []string
}

func (b Batch) Snapshot() Batch {
	out := b
	out.Items = append([]string(nil), b.Items...)
	return out
}

type RequestScope struct {
	Tenant string
	Ctx    context.Context
}

func NewRequestScope(ctx context.Context, tenant string) RequestScope {
	return RequestScope{Tenant: tenant, Ctx: context.Background()}
}

type Attempt struct {
	Key       string
	Version   int
	State     string
	Committed bool
}

func (a Attempt) Next() Attempt {
	a.Version++
	a.State = "running"
	return a
}

func (a Attempt) CanReplace(current Attempt) bool {
	return a.Key == current.Key
}

type BuildResult struct {
	Key    string
	Fields map[string]string
	Ready  bool
}

func (r BuildResult) Snapshot() BuildResult {
	out := r
	out.Fields = make(map[string]string, len(r.Fields))
	for k, v := range r.Fields {
		out.Fields[k] = v
	}
	return out
}

type PooledRequest struct {
	Tenant  string
	Headers []string
}

func (p *PooledRequest) Reset() {
	p.Tenant = ""
	p.Headers = p.Headers[:0]
}

func (p *PooledRequest) Snapshot() PooledRequest {
	return PooledRequest{Tenant: p.Tenant, Headers: append([]string(nil), p.Headers...)}
}

type Event struct {
	Key     string
	Version int
	Status  string
}

type ResourceResult struct {
	Committed bool
	Audit     string
	Err       error
}
