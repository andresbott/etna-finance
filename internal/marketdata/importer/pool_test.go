package importer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewPool_validation(t *testing.T) {
	_, err := NewPool()
	if err == nil {
		t.Error("expected error when no clients")
	}
	_, err = NewPool(nil)
	if err == nil {
		t.Error("expected error when client is nil")
	}
	c := NewMassiveClient("key")
	_, err = NewPool(c, nil)
	if err == nil {
		t.Error("expected error when one client is nil")
	}

	_, err = NewMassivePool(nil)
	if err == nil {
		t.Error("expected error when no keys")
	}
	_, err = NewMassivePool([]string{})
	if err == nil {
		t.Error("expected error when empty keys")
	}
	_, err = NewMassivePool([]string{""})
	if err == nil {
		t.Error("expected error when key is empty")
	}
}

func TestNewPool_rotatesAfterEveryCall(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	errFirst := errors.New("rate limit")
	a := &failingThenOKClient{err: errFirst} // first call fails, then OK
	b := &countingClient{}

	pool, err := NewPool(a, b)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	// First call: A fails, pool rotates to B and retries; B succeeds. Pool then rotates (next call will use A).
	_, err = pool.FetchDailyPrices(ctx, "T", start, end)
	if err != nil {
		t.Fatalf("first FetchDailyPrices: %v", err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("after first call: a.calls=%d b.calls=%d, want 1 each", a.calls, b.calls)
	}

	// Second call: current rotated to A; A succeeds this time; pool rotates again.
	_, err = pool.FetchDailyPrices(ctx, "T", start, end)
	if err != nil {
		t.Fatalf("second FetchDailyPrices: %v", err)
	}
	if a.calls != 2 || b.calls != 1 {
		t.Errorf("after second call: a.calls=%d b.calls=%d, want 2 and 1 (rotate after every call)", a.calls, b.calls)
	}
}

func TestNewPool_spreadsLoadAcrossClients(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	a := &countingClient{}
	b := &countingClient{}
	c := &countingClient{}
	pool, err := NewPool(a, b, c)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	for i := 0; i < 6; i++ {
		_, _ = pool.FetchDailyPrices(ctx, "T", start, end)
	}
	if a.calls != 2 || b.calls != 2 || c.calls != 2 {
		t.Errorf("rotate after every call: A=%d B=%d C=%d, want 2 each", a.calls, b.calls, c.calls)
	}
}

func TestNewPool_allClientsFail(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	errA := errors.New("key A rate limit")
	errB := errors.New("key B rate limit")
	a := &mockClient{err: errA}
	b := &mockClient{err: errB}

	pool, _ := NewPool(a, b)
	_, err := pool.FetchDailyPrices(ctx, "T", start, end)
	if err == nil {
		t.Fatal("expected error when all clients fail")
	}
	if !errors.Is(err, errA) && !errors.Is(err, errB) {
		t.Errorf("error should be from one of the clients: %v", err)
	}
}

func TestNewMassivePool_singleKey(t *testing.T) {
	pool, err := NewMassivePool([]string{"one"})
	if err != nil {
		t.Fatalf("NewMassivePool: %v", err)
	}
	if pool == nil {
		t.Fatal("pool is nil")
	}
}

type countingClient struct {
	mu    sync.Mutex
	calls int
}

func (c *countingClient) FetchDailyPrices(_ context.Context, _ string, _, _ time.Time) ([]PricePoint, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return nil, nil
}

// failingThenOKClient fails on the first call, then returns success.
type failingThenOKClient struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (c *failingThenOKClient) FetchDailyPrices(_ context.Context, _ string, _, _ time.Time) ([]PricePoint, error) {
	c.mu.Lock()
	c.calls++
	first := c.calls == 1
	c.mu.Unlock()
	if first {
		return nil, c.err
	}
	return nil, nil
}
