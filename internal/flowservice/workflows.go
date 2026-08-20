package flowservice

import (
	"context"
	"fmt"
	"sync"

	"apigateway/internal/flowmodel"
	"apigateway/internal/flowstore"
)

type Service struct {
	store *flowstore.Store
	pool  sync.Pool
}

func New(store *flowstore.Store) *Service {
	s := &Service{store: store}
	s.pool.New = func() any { return &flowmodel.PooledRequest{} }
	return s
}

func (s *Service) QueueBatch(batch flowmodel.Batch) flowmodel.Batch {
	snapshot := batch.Snapshot()
	s.store.SaveBatch(snapshot)
	return snapshot
}

func (s *Service) CachedBatch(tenant string) flowmodel.Batch {
	return s.store.Batch(tenant)
}

func (s *Service) CallWithScope(scope flowmodel.RequestScope, call func(context.Context) error) error {
	if err := scope.Ctx.Err(); err != nil {
		return err
	}
	return call(scope.Ctx)
}

func (s *Service) ExecuteWithRetry(key string, call func(int) error) (int, error) {
	for attempt := 1; attempt <= 2; attempt++ {
		err := call(attempt)
		if err == nil {
			s.store.SaveAttempt(flowmodel.Attempt{Key: key, Version: attempt, State: "done", Committed: true})
			return attempt, nil
		}
		if err != flowmodel.ErrTemporary {
			return attempt, err
		}
	}
	return 2, flowmodel.ErrTemporary
}

func (s *Service) BuildSafely(key string, build func(*flowmodel.BuildResult)) (result flowmodel.BuildResult, err error) {
	working := flowmodel.BuildResult{Key: key, Fields: make(map[string]string)}
	defer func() {
		if recovered := recover(); recovered != nil {
			result = flowmodel.BuildResult{}
			err = fmt.Errorf("build %s: %v", key, recovered)
		}
	}()
	build(&working)
	working.Ready = true
	s.store.PublishBuild(working)
	return working.Snapshot(), nil
}

func (s *Service) FinishAttempt(attempt flowmodel.Attempt) bool {
	return s.store.SaveAttempt(attempt)
}

func (s *Service) Attempt(key string) flowmodel.Attempt {
	return s.store.Attempt(key)
}

func (s *Service) AcquireRequest(tenant string, headers []string) *flowmodel.PooledRequest {
	p := s.pool.Get().(*flowmodel.PooledRequest)
	p.Reset()
	p.Tenant = tenant
	p.Headers = append(p.Headers, headers...)
	return p
}

func (s *Service) ReleaseRequest(p *flowmodel.PooledRequest) {
	s.pool.Put(p)
}

func (s *Service) SaveEvent(event flowmodel.Event) bool { return s.store.SaveEvent(event) }
func (s *Service) Event(key string) flowmodel.Event     { return s.store.Event(key) }
