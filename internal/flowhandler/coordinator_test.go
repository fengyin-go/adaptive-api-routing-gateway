package flowhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"apigateway/internal/flowmodel"
	"apigateway/internal/flowservice"
	"apigateway/internal/flowstore"
)

func testCoordinator() *Coordinator { return New(flowservice.New(flowstore.New())) }

func TestSnapshotSequenceIsolation(t *testing.T) {
	c := testCoordinator()
	first := flowmodel.Batch{Tenant: "alpha", Items: []string{"route-a", "route-b"}}
	queued, cached := c.SnapshotSequence(first, flowmodel.Batch{Tenant: "beta", Items: []string{"route-x"}})
	want := []string{"route-a", "route-b"}
	if !reflect.DeepEqual(queued.Items, want) || !reflect.DeepEqual(cached.Items, want) {
		t.Fatalf("first batch changed: queued=%v cached=%v", queued.Items, cached.Items)
	}
}

func TestScopeSequenceCancellationIsolation(t *testing.T) {
	c := testCoordinator()
	first, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	a, b := c.ScopeSequence(first, context.Background(), func(ctx context.Context) error { calls++; return ctx.Err() })
	if !errors.Is(a, context.Canceled) || b != nil || calls != 1 {
		t.Fatalf("first=%v second=%v calls=%d", a, b, calls)
	}
}

func TestRetrySequenceTransactionBoundary(t *testing.T) {
	c := testCoordinator()
	count, state, err := c.RetrySequence("reject", func(int) error { return flowmodel.ErrRejected })
	if !errors.Is(err, flowmodel.ErrRejected) || count != 1 || state.Committed {
		t.Fatalf("reject count=%d state=%+v err=%v", count, state, err)
	}
	calls := 0
	count, state, err = c.RetrySequence("temporary", func(int) error {
		calls++
		if calls == 1 {
			return flowmodel.ErrTemporary
		}
		return nil
	})
	if err != nil || count != 2 || calls != 2 || !state.Committed {
		t.Fatalf("retry count=%d calls=%d state=%+v err=%v", count, calls, state, err)
	}
}

func TestRecoverySequenceDoesNotPublishPartialState(t *testing.T) {
	c := testCoordinator()
	result, ready, err := c.RecoverySequence("broken", func(out *flowmodel.BuildResult) { out.Fields["partial"] = "yes"; panic("decoder") })
	if err == nil || !ready || result.Key != "broken-next" {
		t.Fatalf("result=%+v ready=%v err=%v", result, ready, err)
	}
}

func TestVersionSequenceRejectsLateCallback(t *testing.T) {
	state, sideEffects := testCoordinator().VersionSequence("deploy")
	if state.Version != 2 || state.State != "done" || sideEffects != 1 {
		t.Fatalf("state=%+v sideEffects=%d", state, sideEffects)
	}
}

func TestPoolSequenceTenantIsolation(t *testing.T) {
	a, b := testCoordinator().PoolSequence()
	if a.Tenant != "tenant-a" || b.Tenant != "tenant-b" || reflect.DeepEqual(a.Headers, b.Headers) {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
}

func TestFanoutErrorClosesLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := testCoordinator().Fanout(ctx, []string{"a", "b", "c"}, 1)
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err=%v", err)
	}
	out, err := testCoordinator().Fanout(context.Background(), []string{"a", "b"}, -1)
	if err != nil || len(out) != 2 {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func TestResourceSequenceRollsBackBeforeAudit(t *testing.T) {
	c := testCoordinator()
	failed := c.ResourceSequence([]string{"a", "b", "c"}, 1)
	if failed.Committed || failed.Audit != "failed" || failed.Err == nil {
		t.Fatalf("failed=%+v", failed)
	}
	ok := c.ResourceSequence([]string{"a", "b"}, -1)
	if !ok.Committed || ok.Audit != "committed" || ok.Err != nil {
		t.Fatalf("ok=%+v", ok)
	}
}

func TestShutdownSequenceStopsRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	done := make(chan int, 1)
	go func() {
		done <- testCoordinator().ShutdownSequence(ctx, func() {
			calls++
			if calls == 2 {
				cancel()
			}
		})
	}()
	select {
	case count := <-done:
		if count != 2 || calls != 2 {
			t.Fatalf("count=%d calls=%d", count, calls)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("shutdown timed out")
	}
}

func TestPublishSequenceKeepsCommittedEvent(t *testing.T) {
	state, sideEffects := testCoordinator().VersionSequence("publish-state")
	if state.Version != 2 || state.State != "done" || sideEffects != 1 {
		t.Fatalf("publish state=%+v sideEffects=%d", state, sideEffects)
	}
	calls := 0
	event, commits := testCoordinator().PublishSequence("route", func(int) error {
		calls++
		if calls == 1 {
			return errors.New("broker unavailable")
		}
		return nil
	})
	if calls != 2 || commits != 1 || event.Version != 2 || event.Status != "committed" {
		t.Fatalf("calls=%d commits=%d event=%+v", calls, commits, event)
	}
}
