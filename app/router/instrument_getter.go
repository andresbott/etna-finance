package router

import (
	"context"
	"errors"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/marketdata"
)

// NewInstrumentGetter returns an accounting.InstrumentGetter backed by the marketdata store.
// Used when initializing the app (e.g. in runServer) so the same store can be shared.
func NewInstrumentGetter(store *marketdata.Store) accounting.InstrumentGetter {
	return &marketDataInstrumentGetter{store: store}
}

// marketDataInstrumentGetter adapts marketdata.Store to accounting.InstrumentGetter.
type marketDataInstrumentGetter struct {
	store *marketdata.Store
}

func (g *marketDataInstrumentGetter) GetInstrument(ctx context.Context, id uint) (accounting.InstrumentInfo, error) {
	inst, err := g.store.GetInstrument(ctx, id)
	if err != nil {
		if errors.Is(err, marketdata.ErrInstrumentNotFound) {
			return accounting.InstrumentInfo{}, accounting.ErrInstrumentNotFound
		}
		return accounting.InstrumentInfo{}, err
	}
	return accounting.InstrumentInfo{ID: inst.ID, Currency: inst.Currency}, nil
}
