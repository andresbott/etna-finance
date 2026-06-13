package backup

import (
	"encoding/json"
	"testing"
)

func TestPriceRecordV1HasOHLCV(t *testing.T) {
	t.Run("OHLCV candle is recognized", func(t *testing.T) {
		var rec priceRecordV1
		raw := `{"symbol":"VOO","time":"2025-03-10T00:00:00Z","open":1,"high":4,"low":0.5,"close":3,"volume":100}`
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !rec.hasOHLCV() {
			t.Error("expected hasOHLCV true for full candle")
		}
	})

	t.Run("legacy price-only record has no OHLCV", func(t *testing.T) {
		var rec priceRecordV1
		// Pre-OHLCV backups carry a single "price" field and no OHLCV keys.
		raw := `{"symbol":"VOO","time":"2025-03-10T00:00:00Z","price":515.51}`
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if rec.hasOHLCV() {
			t.Error("expected hasOHLCV false for legacy price-only record")
		}
		if rec.Price == nil || *rec.Price != 515.51 {
			t.Errorf("expected legacy Price 515.51, got %v", rec.Price)
		}
	})

	t.Run("a single non-zero OHLCV leg counts as a candle", func(t *testing.T) {
		var rec priceRecordV1
		raw := `{"symbol":"VOO","time":"2025-03-10T00:00:00Z","close":11}`
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !rec.hasOHLCV() {
			t.Error("expected hasOHLCV true when any OHLCV leg is non-zero")
		}
	})
}
