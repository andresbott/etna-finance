package importer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"
)

func sf(filing, end, period, year string, basic, diluted float64) models.StockFinancial {
	return models.StockFinancial{
		FilingDate:   filing,
		EndDate:      end,
		FiscalPeriod: period,
		FiscalYear:   year,
		Financials: map[string]models.Financial{
			"income_statement": {
				"basic_earnings_per_share":   {Value: basic},
				"diluted_earnings_per_share": {Value: diluted},
			},
		},
	}
}

func TestEpsFromStockFinancial(t *testing.T) {
	t.Run("extracts basic and diluted from filing date", func(t *testing.T) {
		got, ok := epsFromStockFinancial("AAPL", sf("2025-02-01", "2024-12-28", "Q1", "2025", 6.42, 6.38))
		if !ok {
			t.Fatal("expected ok=true")
		}
		want := EPSPoint{
			Symbol: "AAPL", FiscalPeriod: "Q1", FiscalYear: "2025",
			Time:    time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			Basic:   6.42, Diluted: 6.38,
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("falls back to end date when filing date missing", func(t *testing.T) {
		got, ok := epsFromStockFinancial("AAPL", sf("", "2024-12-28", "Q1", "2025", 1.0, 0.9))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if !got.Time.Equal(time.Date(2024, 12, 28, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("expected fallback to end date, got %v", got.Time)
		}
	})

	t.Run("skips filing with no parseable date", func(t *testing.T) {
		if _, ok := epsFromStockFinancial("AAPL", sf("", "", "Q1", "2025", 1.0, 0.9)); ok {
			t.Error("expected ok=false when no date present")
		}
	})

	t.Run("missing income statement yields zero EPS but is kept", func(t *testing.T) {
		bare := models.StockFinancial{FilingDate: "2025-02-01", FiscalPeriod: "Q1", FiscalYear: "2025"}
		got, ok := epsFromStockFinancial("AAPL", bare)
		if !ok {
			t.Fatal("expected ok=true (date present)")
		}
		if got.Basic != 0 || got.Diluted != 0 {
			t.Errorf("expected zero EPS, got %+v", got)
		}
	})
}

// --- pool tests (mirror the reference) ---

type mockFundamentalsClient struct {
	result []EPSPoint
	err    error
}

func (m *mockFundamentalsClient) FetchEPS(_ context.Context, _ string) ([]EPSPoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type countingFundamentalsClient struct {
	mu    sync.Mutex
	calls int
}

func (c *countingFundamentalsClient) FetchEPS(_ context.Context, _ string) ([]EPSPoint, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return nil, nil
}

func TestNewMassiveFundamentalsPool_validation(t *testing.T) {
	if _, err := NewMassiveFundamentalsPool(nil); err == nil {
		t.Error("expected error when no keys")
	}
	if _, err := NewMassiveFundamentalsPool([]string{""}); err == nil {
		t.Error("expected error when key is empty")
	}
	if _, err := NewMassiveFundamentalsPool([]string{"key1"}); err != nil {
		t.Errorf("unexpected error for single valid key: %v", err)
	}
}

func TestFundamentalsPool_spreadsLoad(t *testing.T) {
	ctx := context.Background()
	a, b := &countingFundamentalsClient{}, &countingFundamentalsClient{}
	pool, err := NewFundamentalsPool(a, b)
	if err != nil {
		t.Fatalf("NewFundamentalsPool: %v", err)
	}
	for i := 0; i < 4; i++ {
		_, _ = pool.FetchEPS(ctx, "AAPL")
	}
	if a.calls != 2 || b.calls != 2 {
		t.Errorf("spread load: a=%d b=%d, want 2 each", a.calls, b.calls)
	}
}

func TestFundamentalsPool_allClientsFail(t *testing.T) {
	ctx := context.Background()
	errA := errors.New("key A error")
	pool, _ := NewFundamentalsPool(&mockFundamentalsClient{err: errA}, &mockFundamentalsClient{err: errA})
	if _, err := pool.FetchEPS(ctx, "AAPL"); err == nil {
		t.Fatal("expected error when all clients fail")
	}
}

func TestFundamentalsPool_returnsData(t *testing.T) {
	ctx := context.Background()
	expected := []EPSPoint{{Symbol: "AAPL", FiscalPeriod: "Q3", FiscalYear: "2025", Basic: 6.42, Diluted: 6.38}}
	pool, _ := NewFundamentalsPool(&mockFundamentalsClient{result: expected})
	got, err := pool.FetchEPS(ctx, "AAPL")
	if err != nil {
		t.Fatalf("FetchEPS: %v", err)
	}
	if len(got) != 1 || got[0].Basic != 6.42 {
		t.Errorf("got %+v, want %+v", got, expected)
	}
}

type failingThenOKFundamentalsClient struct {
	err      error
	result   []EPSPoint
	failNext bool
}

func (c *failingThenOKFundamentalsClient) FetchEPS(_ context.Context, _ string) ([]EPSPoint, error) {
	if c.failNext {
		c.failNext = false
		return nil, c.err
	}
	return c.result, nil
}

func TestFundamentalsPool_failoverRecovers(t *testing.T) {
	ctx := context.Background()
	want := []EPSPoint{{Symbol: "AAPL", Basic: 1.23}}
	failing := &failingThenOKFundamentalsClient{err: errors.New("key A down"), failNext: true}
	ok := &mockFundamentalsClient{result: want}
	pool, err := NewFundamentalsPool(failing, ok)
	if err != nil {
		t.Fatalf("NewFundamentalsPool: %v", err)
	}
	got, err := pool.FetchEPS(ctx, "AAPL")
	if err != nil {
		t.Fatalf("expected failover to succeed, got error: %v", err)
	}
	if len(got) != 1 || got[0].Basic != 1.23 {
		t.Errorf("expected recovered data %+v, got %+v", want, got)
	}
}
