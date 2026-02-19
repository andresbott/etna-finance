package router

import (
	"context"
	"errors"

	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/marketdata"
)

// marketDataInstrumentGetter adapts marketdata.Store to accounting.InstrumentGetter.
type marketDataInstrumentGetter struct {
	store *marketdata.Store
}

func (g *marketDataInstrumentGetter) GetInstrument(ctx context.Context, id uint, tenant string) (accounting.InstrumentInfo, error) {
	inst, err := g.store.GetInstrument(ctx, id, tenant)
	if err != nil {
		if errors.Is(err, marketdata.ErrInstrumentNotFound) {
			return accounting.InstrumentInfo{}, accounting.ErrInstrumentNotFound
		}
		return accounting.InstrumentInfo{}, err
	}
	return accounting.InstrumentInfo{ID: inst.ID, Currency: inst.Currency}, nil
}
