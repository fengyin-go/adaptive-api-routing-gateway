package flowhandler

import (
	"context"
	"errors"
	"sync"
	"time"

	"apigateway/internal/flowmodel"
	"apigateway/internal/flowservice"
)

type Coordinator struct{ service *flowservice.Service }

func New(service *flowservice.Service) *Coordinator { return &Coordinator{service: service} }

func (c *Coordinator) SnapshotSequence(first, second flowmodel.Batch) (flowmodel.Batch, flowmodel.Batch) {
	queued := c.service.QueueBatch(first)
	first.Items = append(first.Items[:0], second.Items...)
	c.service.QueueBatch(second)
	return queued.Snapshot(), c.service.CachedBatch(queued.Tenant)
}

func (c *Coordinator) ScopeSequence(first context.Context, second context.Context, call func(context.Context) error) (error, error) {
	firstErr := c.service.CallWithScope(flowmodel.NewRequestScope(first, "first"), call)
	secondErr := c.service.CallWithScope(flowmodel.NewRequestScope(second, "second"), call)
	return firstErr, secondErr
}

func (c *Coordinator) RetrySequence(key string, call func(int) error) (int, flowmodel.Attempt, error) {
	count, err := c.service.ExecuteWithRetry(key, call)
	return count, c.service.Attempt(key), err
}

func (c *Coordinator) RecoverySequence(key string, build func(*flowmodel.BuildResult)) (flowmodel.BuildResult, bool, error) {
	_, err := c.service.BuildSafely(key, build)
	result, _ := c.service.BuildSafely(key+"-next", func(out *flowmodel.BuildResult) { out.Fields["status"] = "ready" })
	return result, okReady(result), err
}

func okReady(result flowmodel.BuildResult) bool {
	return result.Ready && result.Fields["status"] == "ready"
}

func (c *Coordinator) VersionSequence(key string) (flowmodel.Attempt, int) {
	// First publish round starts running, then a second round succeeds and
	// commits. The late first-round callback arriving afterwards must be
	// dropped so the committed result stays final — the store's CanReplace
	// guard rejects the older version. A single publish yields a single
	// effective operation, so only the committed (terminal) write counts; the
	// transient "running" start is not an effective operation.
	first := flowmodel.Attempt{Key: key, Version: 1, State: "running"}
	c.service.FinishAttempt(first)
	retry := first.Next()
	retry.State = "done"
	retry.Committed = true
	sideEffects := 0
	if c.service.FinishAttempt(retry) {
		sideEffects++
	}
	// Late first-round callback: older version (1) after a committed v2 —
	// rejected by the store, so it produces no side effect.
	late := first
	late.State = "running"
	c.service.FinishAttempt(late)
	return c.service.Attempt(key), sideEffects
}

func (c *Coordinator) PoolSequence() (flowmodel.PooledRequest, flowmodel.PooledRequest) {
	first := c.service.AcquireRequest("tenant-a", []string{"a"})
	firstSnapshot := first.Snapshot()
	c.service.ReleaseRequest(first)
	second := c.service.AcquireRequest("tenant-b", []string{"b"})
	secondSnapshot := second.Snapshot()
	c.service.ReleaseRequest(second)
	return firstSnapshot, secondSnapshot
}

func (c *Coordinator) Fanout(ctx context.Context, values []string, failAt int) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan string, len(values))
	errs := make(chan error, 1)
	var wg sync.WaitGroup
	for i, value := range values {
		wg.Add(1)
		go func(i int, value string) {
			defer wg.Done()
			if i == failAt {
				select {
				case errs <- errors.New("upstream rejected"):
				default:
				}
				cancel()
				return
			}
			select {
			case results <- value:
			case <-ctx.Done():
			}
		}(i, value)
	}
	go func() { wg.Wait(); close(results) }()
	var out []string
	for results != nil {
		select {
		case value, ok := <-results:
			if !ok {
				results = nil
				continue
			}
			out = append(out, value)
		case err := <-errs:
			return out, err
		case <-ctx.Done():
			select {
			case err := <-errs:
				return out, err
			default:
				return out, ctx.Err()
			}
		}
	}
	return out, nil
}

func (c *Coordinator) ResourceSequence(values []string, failAt int) flowmodel.ResourceResult {
	open := 0
	for i := range values {
		open++
		if open > 1 {
			return flowmodel.ResourceResult{Err: errors.New("resource limit exceeded")}
		}
		if i == failAt {
			open--
			return flowmodel.ResourceResult{Audit: "failed", Err: errors.New("write rejected")}
		}
		open--
	}
	return flowmodel.ResourceResult{Committed: true, Audit: "committed"}
}

func (c *Coordinator) ShutdownSequence(ctx context.Context, call func()) int {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	count := 0
	for {
		select {
		case <-ctx.Done():
			return count
		case <-ticker.C:
			call()
			count++
		}
	}
}

func (c *Coordinator) PublishSequence(key string, publish func(int) error) (flowmodel.Event, int) {
	// A publish may need more than one round: the first attempt can fail and
	// the second succeeds and commits. The committed event is final — a late
	// first-round callback (version 1 pending arriving after version 2
	// committed) must not revert it. Only genuinely committed writes count.
	commits := 0
	for version := 1; version <= 2; version++ {
		c.service.SaveEvent(flowmodel.Event{Key: key, Version: version, Status: "pending"})
		if err := publish(version); err != nil {
			continue
		}
		if c.service.SaveEvent(flowmodel.Event{Key: key, Version: version, Status: "committed"}) {
			commits++
		}
		break
	}
	// Late first-round callback: version 1 pending after version 2 has
	// committed — rejected by the store's monotonic guard, no commit recorded.
	c.service.SaveEvent(flowmodel.Event{Key: key, Version: 1, Status: "pending"})
	return c.service.Event(key), commits
}
