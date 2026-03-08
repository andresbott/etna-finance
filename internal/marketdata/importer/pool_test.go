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

// --- forexTicker tests ---

func TestForexTicker(t *testing.T) {
	tests := []struct {
		main, secondary, want string
	}{
		{"chf", "usd", "C:CHFUSD"},
		{"EUR", "GBP", "C:EURGBP"},
		{"usd", "JPY", "C:USDJPY"},
	}
	for _, tt := range tests {
		got := forexTicker(tt.main, tt.secondary)
		if got != tt.want {
			t.Errorf("forexTicker(%q, %q) = %q, want %q", tt.main, tt.secondary, got, tt.want)
		}
	}
}

// --- FXPool tests ---

func TestNewFXPool_validation(t *testing.T) {
	_, err := NewFXPool()
	if err == nil {
		t.Error("expected error when no FX clients")
	}
	_, err = NewFXPool(nil)
	if err == nil {
		t.Error("expected error when FX client is nil")
	}
	c := &mockFXClient{}
	_, err = NewFXPool(c, nil)
	if err == nil {
		t.Error("expected error when one FX client is nil")
	}
}

func TestNewMassiveFXPool_validation(t *testing.T) {
	_, err := NewMassiveFXPool(nil)
	if err == nil {
		t.Error("expected error when no keys")
	}
	_, err = NewMassiveFXPool([]string{})
	if err == nil {
		t.Error("expected error when empty keys")
	}
	_, err = NewMassiveFXPool([]string{""})
	if err == nil {
		t.Error("expected error when key is empty")
	}
}

func TestNewMassiveFXPool_singleKey(t *testing.T) {
	pool, err := NewMassiveFXPool([]string{"key1"})
	if err != nil {
		t.Fatalf("NewMassiveFXPool: %v", err)
	}
	if pool == nil {
		t.Fatal("pool is nil")
	}
}

func TestFXPool_rotatesAfterEveryCall(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	errFirst := errors.New("rate limit")
	a := &failingThenOKFXClient{err: errFirst}
	b := &countingFXClient{}

	pool, err := NewFXPool(a, b)
	if err != nil {
		t.Fatalf("NewFXPool: %v", err)
	}

	// First call: A fails, pool rotates to B; B succeeds.
	_, err = pool.FetchDailyRates(ctx, "USD", "EUR", start, end)
	if err != nil {
		t.Fatalf("first FetchDailyRates: %v", err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("after first call: a.calls=%d b.calls=%d, want 1 each", a.calls, b.calls)
	}

	// Second call: A succeeds now.
	_, err = pool.FetchDailyRates(ctx, "USD", "EUR", start, end)
	if err != nil {
		t.Fatalf("second FetchDailyRates: %v", err)
	}
	if a.calls != 2 || b.calls != 1 {
		t.Errorf("after second call: a.calls=%d b.calls=%d, want 2 and 1", a.calls, b.calls)
	}
}

func TestFXPool_spreadsLoad(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	a := &countingFXClient{}
	b := &countingFXClient{}
	c := &countingFXClient{}
	pool, err := NewFXPool(a, b, c)
	if err != nil {
		t.Fatalf("NewFXPool: %v", err)
	}

	for i := 0; i < 6; i++ {
		_, _ = pool.FetchDailyRates(ctx, "USD", "EUR", start, end)
	}
	if a.calls != 2 || b.calls != 2 || c.calls != 2 {
		t.Errorf("spread load: A=%d B=%d C=%d, want 2 each", a.calls, b.calls, c.calls)
	}
}

func TestFXPool_allClientsFail(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	errA := errors.New("key A rate limit")
	errB := errors.New("key B rate limit")
	a := &mockFXClient{err: errA}
	b := &mockFXClient{err: errB}

	pool, _ := NewFXPool(a, b)
	_, err := pool.FetchDailyRates(ctx, "USD", "EUR", start, end)
	if err == nil {
		t.Fatal("expected error when all FX clients fail")
	}
	if !errors.Is(err, errA) && !errors.Is(err, errB) {
		t.Errorf("error should be from one of the clients: %v", err)
	}
}

func TestFXPool_returnsData(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	expected := []RatePoint{
		{Time: start, Rate: 1.05},
		{Time: end, Rate: 1.06},
	}
	a := &mockFXClient{points: expected}
	pool, err := NewFXPool(a)
	if err != nil {
		t.Fatalf("NewFXPool: %v", err)
	}
	got, err := pool.FetchDailyRates(ctx, "EUR", "USD", start, end)
	if err != nil {
		t.Fatalf("FetchDailyRates: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("got %d points, want %d", len(got), len(expected))
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("point[%d] = %+v, want %+v", i, got[i], expected[i])
		}
	}
}

// --- FX mock helpers ---

type mockFXClient struct {
	points []RatePoint
	err    error
}

func (m *mockFXClient) FetchDailyRates(_ context.Context, _, _ string, _, _ time.Time) ([]RatePoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.points, nil
}

type countingFXClient struct {
	mu    sync.Mutex
	calls int
}

func (c *countingFXClient) FetchDailyRates(_ context.Context, _, _ string, _, _ time.Time) ([]RatePoint, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return nil, nil
}

type failingThenOKFXClient struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (c *failingThenOKFXClient) FetchDailyRates(_ context.Context, _, _ string, _, _ time.Time) ([]RatePoint, error) {
	c.mu.Lock()
	c.calls++
	first := c.calls == 1
	c.mu.Unlock()
	if first {
		return nil, c.err
	}
	return nil, nil
}

// --- existing mock helpers ---

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
