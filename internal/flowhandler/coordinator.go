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
	return queued, c.service.CachedBatch(queued.Tenant)
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
	first := flowmodel.Attempt{Key: key, Version: 1, State: "running"}
	c.service.FinishAttempt(first)
	retry := first.Next()
	retry.State = "done"
	c.service.FinishAttempt(retry)
	late := first
	late.State = "running"
	c.service.FinishAttempt(late)
	return c.service.Attempt(key), 1
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan string, len(values))
	errs := make(chan error, 1)
	var wg sync.WaitGroup

	// wg.Add must happen before the goroutine starts so the supervisor's
	// wg.Wait() cannot observe an empty WaitGroup and call cancel() early.
	wg.Add(len(values))
	for i, value := range values {
		go func(i int, value string) {
			defer wg.Done()
			if i == failAt {
				// Report the error then signal shutdown. errs is buffered
				// (cap 1) so this send never blocks even if no one is
				// reading yet, and the first failure wins.
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
	// Once every worker has finished, release the cancel sentinel so the
	// aggregator drains remaining results and exits.
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	var out []string
	for {
		select {
		case <-done:
			// All workers finished. Drain any buffered results before
			// deciding; an error reported earlier may still be pending.
			select {
			case err := <-errs:
				return out, err
			default:
			}
			for {
				select {
				case value := <-results:
					out = append(out, value)
				default:
					return out, nil
				}
			}
		case value := <-results:
			out = append(out, value)
		case err := <-errs:
			// A worker failed: cancel the rest, but still wait for them
			// to wind down so no goroutine lingers writing stale results.
			cancel()
			<-done
			return out, err
		}
	}
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
	for version := 1; version <= 2; version++ {
		if err := publish(version); err != nil {
			continue
		}
		c.service.SaveEvent(flowmodel.Event{Key: key, Version: version, Status: "committed"})
		break
	}
	c.service.SaveEvent(flowmodel.Event{Key: key, Version: 1, Status: "pending"})
	return c.service.Event(key), 1
}
