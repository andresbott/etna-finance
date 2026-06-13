package importer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
// adjusted for splits, and returns full OHLCV candles for each day.
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
			Time:   t,
			Open:   agg.Open,
			High:   agg.High,
			Low:    agg.Low,
			Close:  agg.Close,
			Volume: agg.Volume,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// GetTickerDetails implements ReferenceClient by calling the Massive reference API
// (/v3/reference/tickers/{ticker}). A symbol the provider does not know returns
// TickerDetails{Found: false} with a nil error.
func (c *MassiveClient) GetTickerDetails(ctx context.Context, symbol string) (TickerDetails, error) {
	if symbol == "" {
		return TickerDetails{}, fmt.Errorf("symbol is required")
	}
	params := &models.GetTickerDetailsParams{Ticker: strings.ToUpper(symbol)}
	res, err := c.rest.GetTickerDetails(ctx, params)
	if err != nil {
		// An unknown ticker comes back as HTTP 404; report that as a clean not-found so the
		// caller can degrade gracefully. Any other failure (transport error, auth, 5xx, rate
		// limit) is a real problem and must surface as an error per the ReferenceClient contract.
		var apiErr *models.ErrorResponse
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return TickerDetails{Found: false}, nil
		}
		return TickerDetails{}, err
	}
	t := res.Results
	notes := strings.TrimSpace(t.Description)
	if t.PrimaryExchange != "" {
		if notes != "" {
			notes = t.PrimaryExchange + " — " + notes
		} else {
			notes = t.PrimaryExchange
		}
	}
	return TickerDetails{
		Name:     t.Name,
		Currency: strings.ToUpper(t.CurrencyName),
		Type:     t.Type,
		Exchange: t.PrimaryExchange,
		Notes:    notes,
		Found:    t.Name != "" || t.Type != "" || t.PrimaryExchange != "",
	}, nil
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
