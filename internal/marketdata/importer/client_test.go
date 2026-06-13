package importer

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClient_FetchDailyPrices_yieldsResult(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
	points := []PricePoint{
		{Time: start, Open: 98.0, High: 101.0, Low: 97.0, Close: 100.0, Volume: 1000},
		{Time: start.AddDate(0, 0, 1), Open: 100.0, High: 102.0, Low: 99.0, Close: 101.0, Volume: 1100},
		{Time: end, Open: 101.0, High: 103.0, Low: 100.0, Close: 102.0, Volume: 1200},
	}

	client := &mockClient{points: points}
	got, err := client.FetchDailyPrices(ctx, "AAPL", start, end)
	if err != nil {
		t.Fatalf("FetchDailyPrices: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d points, want 3", len(got))
	}
	for i, p := range got {
		want := points[i]
		if !p.Time.Equal(want.Time) || p.Open != want.Open || p.High != want.High || p.Low != want.Low || p.Close != want.Close || p.Volume != want.Volume {
			t.Errorf("point[%d]: got Time=%v Open=%v High=%v Low=%v Close=%v Volume=%v, want Time=%v Open=%v High=%v Low=%v Close=%v Volume=%v",
				i, p.Time, p.Open, p.High, p.Low, p.Close, p.Volume,
				want.Time, want.Open, want.High, want.Low, want.Close, want.Volume)
		}
	}
}

func TestClient_FetchDailyPrices_emptyResult(t *testing.T) {
	ctx := context.Background()
	client := &mockClient{points: nil}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)

	got, err := client.FetchDailyPrices(ctx, "XYZ", start, end)
	if err != nil {
		t.Fatalf("FetchDailyPrices: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice, got len %d", len(got))
	}
}

func TestClient_FetchDailyPrices_propagatesError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("api error")
	client := &mockClient{err: wantErr}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)

	_, err := client.FetchDailyPrices(ctx, "AAPL", start, end)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error should be %v: %v", wantErr, err)
	}
}

// mockClient implements Client for tests.
type mockClient struct {
	points []PricePoint
	err    error
}

func (m *mockClient) FetchDailyPrices(_ context.Context, _ string, _, _ time.Time) ([]PricePoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.points, nil
}
