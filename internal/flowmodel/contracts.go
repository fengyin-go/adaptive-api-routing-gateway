package flowmodel

import (
	"context"
	"errors"
)

var (
	// ErrRejected is a non-retryable failure: the gateway explicitly refused the
	// write, so retrying would only repeat the rejection.
	ErrRejected = errors.New("gateway request rejected")
	// ErrTemporary is a retryable failure: the upstream or resource was transiently
	// unavailable, so the caller may attempt the write again.
	ErrTemporary = errors.New("gateway temporary failure")
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
	return RequestScope{Tenant: tenant, Ctx: ctx}
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

// CanReplace reports whether attempt a may overwrite the current attempt for
// the same key. A strictly newer version always wins. An equal version is also
// permitted so that a failed terminal state can replace a prior partial state
// recorded for the same attempt; only older (stale) versions are rejected.
func (a Attempt) CanReplace(current Attempt) bool {
	return a.Key == current.Key && a.Version >= current.Version
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

// Finalize resolves a resource sequence result into its terminal transaction
// state. On failure the write must roll back: nothing is committed and the audit
// records the failure honestly rather than masking it as a success.
func (r ResourceResult) Finalize() ResourceResult {
	if r.Err != nil {
		r.Committed = false
		r.Audit = "failed"
	}
	return r
}
