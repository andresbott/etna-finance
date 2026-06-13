package importer

import (
	"context"
	"errors"
	"testing"
)

type fakeRef struct {
	details TickerDetails
	err     error
	calls   int
}

func (f *fakeRef) GetTickerDetails(_ context.Context, _ string) (TickerDetails, error) {
	f.calls++
	return f.details, f.err
}

func TestReferencePool_RotatesAndRetries(t *testing.T) {
	bad := &fakeRef{err: errors.New("boom")}
	good := &fakeRef{details: TickerDetails{Name: "Apple Inc.", Found: true}}

	pool, err := NewReferencePool(bad, good)
	if err != nil {
		t.Fatalf("NewReferencePool: %v", err)
	}

	got, err := pool.GetTickerDetails(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Apple Inc." {
		t.Errorf("name: got %q want %q", got.Name, "Apple Inc.")
	}
	if bad.calls != 1 || good.calls != 1 {
		t.Errorf("calls: bad=%d good=%d, want 1 and 1", bad.calls, good.calls)
	}
}

func TestReferencePool_AllFailReturnsError(t *testing.T) {
	a := &fakeRef{err: errors.New("boom a")}
	b := &fakeRef{err: errors.New("boom b")}

	pool, err := NewReferencePool(a, b)
	if err != nil {
		t.Fatalf("NewReferencePool: %v", err)
	}

	if _, err := pool.GetTickerDetails(context.Background(), "AAPL"); err == nil {
		t.Fatal("expected error when all clients fail")
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("calls: a=%d b=%d, want each tried exactly once", a.calls, b.calls)
	}
}

func TestNewReferencePool_RequiresClient(t *testing.T) {
	if _, err := NewReferencePool(); err == nil {
		t.Error("expected error with no clients")
	}
}

func TestNewMassiveReferencePool_RequiresKey(t *testing.T) {
	if _, err := NewMassiveReferencePool(nil); err == nil {
		t.Error("expected error with no keys")
	}
}
