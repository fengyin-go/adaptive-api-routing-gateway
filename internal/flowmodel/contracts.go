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

// CanReplace decides whether a new attempt may overwrite the current one.
// A late first-round callback must never clobber a later, already-committed
// result, so an older version (next.Version < current.Version) is always
// rejected. Equal versions are only allowed when the next attempt is not
// reverting an already-committed result back to a non-committed state — this
// keeps forward progress (running -> done) while blocking regressions.
func (a Attempt) CanReplace(current Attempt) bool {
	if a.Key != current.Key {
		return false
	}
	if a.Version < current.Version {
		return false
	}
	if a.Version == current.Version && current.Committed && !a.Committed {
		return false
	}
	return true
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

// CanReplace enforces monotonic publish progression for an event so that a
// committed result is final and cannot be reverted by a late, stale callback.
//
//   - A strictly higher version always wins (a new publish round supersedes
//     the previous one, e.g. version 2 over version 1).
//   - The same version may advance status forward (pending -> committed) but
//     never regress a committed status back to pending — once committed, the
//     result is final for that version.
//   - A lower version is rejected outright (a late first-round callback
//     arriving after a later round has already committed).
func (e Event) CanReplace(current Event) bool {
	if e.Key != current.Key {
		return false
	}
	if e.Version > current.Version {
		return true
	}
	if e.Version < current.Version {
		return false
	}
	// Same version: only forward transitions are allowed. A committed status
	// must never be reverted to a non-committed one (e.g. pending).
	return !(current.Status == "committed" && e.Status != "committed")
}

type ResourceResult struct {
	Committed bool
	Audit     string
	Err       error
}
