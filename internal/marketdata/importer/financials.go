package importer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"
)

// ErrNoFinancials indicates the provider returned no financial filings for the symbol.
var ErrNoFinancials = errors.New("no financials found")

// epsFilingsLimit is the number of recent quarterly filings to request per symbol.
const epsFilingsLimit = 8

// EPSPoint is one quarterly EPS observation extracted from a company's SEC filing.
type EPSPoint struct {
	Symbol       string
	FiscalPeriod string // Q1, Q2, Q3, Q4, FY
	FiscalYear   string
	Time         time.Time // FilingDate; falls back to EndDate; parsed "2006-01-02"
	Basic        float64   // income_statement.basic_earnings_per_share
	Diluted      float64   // income_statement.diluted_earnings_per_share
}

// FundamentalsClient fetches quarterly EPS for a stock ticker. Implementations may wrap a single
// API key or a pool of clients for key rotation. The client only yields data; the caller persists it.
type FundamentalsClient interface {
	FetchEPS(ctx context.Context, symbol string) ([]EPSPoint, error)
}

// epsFromStockFinancial extracts an EPSPoint from a single Massive StockFinancial. It returns
// ok=false when neither FilingDate nor EndDate parses (the point cannot be timestamped). A filing
// with no income_statement yields zero EPS but is still returned (ok=true) when a date exists.
func epsFromStockFinancial(symbol string, sf models.StockFinancial) (EPSPoint, bool) {
	var t time.Time
	for _, dateStr := range []string{sf.FilingDate, sf.EndDate} {
		if dateStr != "" {
			if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
				t = parsed.UTC()
				break
			}
		}
	}
	if t.IsZero() {
		return EPSPoint{}, false
	}

	var basic, diluted float64
	if income, ok := sf.Financials["income_statement"]; ok {
		if eps, ok := income["basic_earnings_per_share"]; ok {
			basic = eps.Value
		}
		if eps, ok := income["diluted_earnings_per_share"]; ok {
			diluted = eps.Value
		}
	}

	return EPSPoint{
		Symbol:       symbol,
		FiscalPeriod: sf.FiscalPeriod,
		FiscalYear:   sf.FiscalYear,
		Time:         t,
		Basic:        basic,
		Diluted:      diluted,
	}, true
}

// FetchEPS implements FundamentalsClient using the Massive VX Stock Financials API. It requests the
// most recent quarterly filings and extracts basic and diluted EPS from each.
func (c *MassiveClient) FetchEPS(ctx context.Context, symbol string) ([]EPSPoint, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	tf := models.TFQuarterly
	order := models.Desc
	limit := epsFilingsLimit
	params := &models.ListStockFinancialsParams{
		Ticker:    &symbol,
		Timeframe: &tf,
		Order:     &order,
		Limit:     &limit,
	}

	iter := c.rest.VX.ListStockFinancials(ctx, params)
	var results []EPSPoint
	for iter.Next() {
		if pt, ok := epsFromStockFinancial(symbol, iter.Item()); ok {
			results = append(results, pt)
			// We only need the most recent epsFilingsLimit quarters (Order=Desc). Stop once we have
			// them so the iterator does not request the next (older) page — every extra page is
			// another API call that can hit the rate limit for data we would discard anyway.
			if len(results) >= epsFilingsLimit {
				break
			}
		}
	}
	if err := iter.Err(); err != nil {
		// ListStockFinancials paginates. With Order=Desc + Limit, the first page already holds the
		// most recent filings — exactly what TTM needs. The iterator hits the API again to fetch the
		// next (older) page, and on a rate-limited / low-quota plan that follow-up request is what
		// 429s. Discarding the filings we already collected would mean we can never make progress
		// (every retry re-fetches page 1, burning more quota). So return what we have; only fail when
		// we got nothing at all.
		if len(results) > 0 {
			return results, nil
		}
		return nil, fmt.Errorf("list financials for %s: %w", symbol, err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("%w for %s", ErrNoFinancials, symbol)
	}
	return results, nil
}

// FundamentalsPool rotates over multiple FundamentalsClients for API key rotation. It advances to
// the next client after every call (success or failure) and, on error, retries with the next client
// until one succeeds or all have been tried. Mirrors Pool / FXPool.
type FundamentalsPool struct {
	clients []FundamentalsClient
	mu      sync.Mutex
	current int
}

// NewFundamentalsPool returns a FundamentalsClient that rotates over the given clients.
func NewFundamentalsPool(clients ...FundamentalsClient) (*FundamentalsPool, error) {
	if len(clients) == 0 {
		return nil, fmt.Errorf("at least one client is required")
	}
	for i, c := range clients {
		if c == nil {
			return nil, fmt.Errorf("client at index %d is nil", i)
		}
	}
	return &FundamentalsPool{clients: clients}, nil
}

// NewMassiveFundamentalsPool builds a FundamentalsPool of MassiveClients from the given API keys.
func NewMassiveFundamentalsPool(apiKeys []string) (*FundamentalsPool, error) {
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("at least one API key is required")
	}
	clients := make([]FundamentalsClient, len(apiKeys))
	for i, key := range apiKeys {
		if key == "" {
			return nil, fmt.Errorf("API key at index %d is empty", i)
		}
		clients[i] = NewMassiveClient(key)
	}
	return NewFundamentalsPool(clients...)
}

// FetchEPS implements FundamentalsClient. Rotates to the next client on each call; on error retries
// with the next client until one succeeds or all have been tried.
func (p *FundamentalsPool) FetchEPS(ctx context.Context, symbol string) ([]EPSPoint, error) {
	p.mu.Lock()
	n := len(p.clients)
	startAt := p.current
	p.mu.Unlock()

	var lastErr error
	for i := 0; i < n; i++ {
		idx := (startAt + i) % n
		result, err := p.clients[idx].FetchEPS(ctx, symbol)
		p.mu.Lock()
		p.current = (idx + 1) % n
		p.mu.Unlock()
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
