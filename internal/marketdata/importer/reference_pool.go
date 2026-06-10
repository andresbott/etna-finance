package importer

import (
	"context"
	"fmt"
	"sync"
)

// ReferencePool is a ReferenceClient that rotates over multiple underlying clients (e.g.
// multiple MassiveClient instances with different API keys). It advances to the next client
// after every call; on error it retries with the next client until one succeeds or all fail.
type ReferencePool struct {
	clients []ReferenceClient
	mu      sync.Mutex
	current int
}

// NewReferencePool returns a ReferenceClient that rotates over the given clients. At least one
// client is required.
func NewReferencePool(clients ...ReferenceClient) (*ReferencePool, error) {
	if len(clients) == 0 {
		return nil, fmt.Errorf("at least one reference client is required")
	}
	for _, c := range clients {
		if c == nil {
			return nil, fmt.Errorf("reference client cannot be nil")
		}
	}
	return &ReferencePool{clients: clients}, nil
}

// NewMassiveReferencePool builds a ReferencePool of MassiveClients from the given API keys.
func NewMassiveReferencePool(apiKeys []string) (*ReferencePool, error) {
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("at least one API key is required")
	}
	clients := make([]ReferenceClient, len(apiKeys))
	for i, key := range apiKeys {
		if key == "" {
			return nil, fmt.Errorf("API key at index %d is empty", i)
		}
		clients[i] = NewMassiveClient(key)
	}
	return NewReferencePool(clients...)
}

// GetTickerDetails implements ReferenceClient. It calls the current client; on success it
// rotates to the next client and returns. On error it rotates and retries until one succeeds
// or all have been tried.
func (p *ReferencePool) GetTickerDetails(ctx context.Context, symbol string) (TickerDetails, error) {
	p.mu.Lock()
	n := len(p.clients)
	startAt := p.current
	p.mu.Unlock()

	var lastErr error
	for i := 0; i < n; i++ {
		idx := (startAt + i) % n
		client := p.clients[idx]
		details, err := client.GetTickerDetails(ctx, symbol)
		p.mu.Lock()
		p.current = (idx + 1) % n
		p.mu.Unlock()
		if err == nil {
			return details, nil
		}
		lastErr = err
	}
	return TickerDetails{}, lastErr
}
