package importer

import (
	"context"
	"fmt"
	"strings"
	"time"

	massive "github.com/massive-com/client-go/v2/rest"
	"github.com/massive-com/client-go/v2/rest/models"
)

// MassiveClient fetches daily stock aggregates and forex rates from the Massive (formerly Polygon) API
// using a single API key. For rate limiting, use a Pool of MassiveClients with multiple keys.
type MassiveClient struct {
	rest *massive.Client
}

// NewMassiveClient creates a client that uses the given API key for all requests.
func NewMassiveClient(apiKey string) *MassiveClient {
	return &MassiveClient{rest: massive.New(apiKey)}
}

// FetchDailyPrices implements Client by calling the Massive aggregates API with 1-day bars,
// adjusted for splits, and returns close price for each day.
func (c *MassiveClient) FetchDailyPrices(ctx context.Context, symbol string, start, end time.Time) ([]PricePoint, error) {
	params := models.ListAggsParams{
		Ticker:     symbol,
		Multiplier: 1,
		Timespan:   models.Day,
		From:       models.Millis(start),
		To:         models.Millis(end),
	}.WithOrder(models.Asc).WithAdjusted(true)

	iter := c.rest.ListAggs(ctx, params)
	var points []PricePoint
	for iter.Next() {
		agg := iter.Item()
		t := time.Time(agg.Timestamp)
		if t.IsZero() {
			continue
		}
		points = append(points, PricePoint{
			Time:  t,
			Price: agg.Close,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// forexTicker returns the Massive/Polygon forex ticker for the pair (main/secondary).
// Format is C:MAINSEC e.g. C:CHFUSD for 1 CHF = X USD.
func forexTicker(main, secondary string) string {
	return "C:" + strings.ToUpper(main) + strings.ToUpper(secondary)
}

// FetchDailyRates implements FXClient by calling the Massive aggregates API for forex ticker C:MAINSEC,
// 1-day bars, and returns close as the exchange rate for each day.
func (c *MassiveClient) FetchDailyRates(ctx context.Context, main, secondary string, start, end time.Time) ([]RatePoint, error) {
	if main == "" || secondary == "" {
		return nil, fmt.Errorf("main and secondary currency are required")
	}
	ticker := forexTicker(main, secondary)
	params := models.ListAggsParams{
		Ticker:     ticker,
		Multiplier: 1,
		Timespan:   models.Day,
		From:       models.Millis(start),
		To:         models.Millis(end),
	}.WithOrder(models.Asc).WithAdjusted(false)

	iter := c.rest.ListAggs(ctx, params)
	var points []RatePoint
	for iter.Next() {
		agg := iter.Item()
		t := time.Time(agg.Timestamp)
		if t.IsZero() {
			continue
		}
		points = append(points, RatePoint{
			Time: t,
			Rate: agg.Close,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return points, nil
}
