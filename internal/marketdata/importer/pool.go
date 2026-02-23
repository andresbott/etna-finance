package importer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Pool is a Client that rotates over multiple underlying clients (e.g. multiple MassiveClient
// instances with different API keys). It advances to the next client after every call, on both
// success and failure—so load is spread across keys. On error it also retries with the next
// client until one succeeds or all have been tried.
type Pool struct {
	clients []Client
	mu      sync.Mutex
	current int
}

// NewPool returns a Client that rotates to the next client after every call (success or error).
// At least one client is required.
func NewPool(clients ...Client) (*Pool, error) {
	if len(clients) == 0 {
		return nil, fmt.Errorf("at least one client is required")
	}
	for _, c := range clients {
		if c == nil {
			return nil, fmt.Errorf("client cannot be nil")
		}
	}
	return &Pool{clients: clients}, nil
}

// NewMassivePool builds a Pool of MassiveClients from the given API keys.
// This is a convenience for the common case of rotating over multiple Massive keys.
func NewMassivePool(apiKeys []string) (*Pool, error) {
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("at least one API key is required")
	}
	clients := make([]Client, len(apiKeys))
	for i, key := range apiKeys {
		if key == "" {
			return nil, fmt.Errorf("API key at index %d is empty", i)
		}
		clients[i] = NewMassiveClient(key)
	}
	return NewPool(clients...)
}

// FetchDailyPrices implements Client. It calls the current client; on success it rotates to
// the next client and returns. On error it rotates to the next client and retries, until one
// succeeds or all have been tried. The pool always advances after each call (success or failure).
func (p *Pool) FetchDailyPrices(ctx context.Context, symbol string, start, end time.Time) ([]PricePoint, error) {
	p.mu.Lock()
	n := len(p.clients)
	startAt := p.current
	p.mu.Unlock()

	var lastErr error
	for i := 0; i < n; i++ {
		idx := (startAt + i) % n
		client := p.clients[idx]
		points, err := client.FetchDailyPrices(ctx, symbol, start, end)
		if err == nil {
			p.mu.Lock()
			p.current = (idx + 1) % n
			p.mu.Unlock()
			return points, nil
		}
		lastErr = err
		p.mu.Lock()
		p.current = (idx + 1) % n
		p.mu.Unlock()
	}
	return nil, lastErr
}

// FXPool is an FXClient that rotates over multiple FXClients (e.g. multiple MassiveClient instances).
type FXPool struct {
	clients []FXClient
	mu      sync.Mutex
	current int
}

// NewFXPool returns an FXClient that rotates to the next client after every call.
func NewFXPool(clients ...FXClient) (*FXPool, error) {
	if len(clients) == 0 {
		return nil, fmt.Errorf("at least one FX client is required")
	}
	for _, c := range clients {
		if c == nil {
			return nil, fmt.Errorf("FX client cannot be nil")
		}
	}
	return &FXPool{clients: clients}, nil
}

// NewMassiveFXPool builds an FXPool of MassiveClients from the given API keys.
func NewMassiveFXPool(apiKeys []string) (*FXPool, error) {
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("at least one API key is required")
	}
	clients := make([]FXClient, len(apiKeys))
	for i, key := range apiKeys {
		if key == "" {
			return nil, fmt.Errorf("API key at index %d is empty", i)
		}
		clients[i] = NewMassiveClient(key)
	}
	return NewFXPool(clients...)
}

// FetchDailyRates implements FXClient. Rotates to the next client on each call; on error retries with next client.
func (p *FXPool) FetchDailyRates(ctx context.Context, main, secondary string, start, end time.Time) ([]RatePoint, error) {
	p.mu.Lock()
	n := len(p.clients)
	startAt := p.current
	p.mu.Unlock()

	var lastErr error
	for i := 0; i < n; i++ {
		idx := (startAt + i) % n
		client := p.clients[idx]
		points, err := client.FetchDailyRates(ctx, main, secondary, start, end)
		if err == nil {
			p.mu.Lock()
			p.current = (idx + 1) % n
			p.mu.Unlock()
			return points, nil
		}
		lastErr = err
		p.mu.Lock()
		p.current = (idx + 1) % n
		p.mu.Unlock()
	}
	return nil, lastErr
}
