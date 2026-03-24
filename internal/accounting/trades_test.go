package accounting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/go-bumbu/testdbs"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// GetTrade
// ---------------------------------------------------------------------------

func TestGetTrade(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("tradeGet"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			buyID, err := store.CreateStockBuy(ctx, StockBuy{
				Description:         "Buy 10",
				Date:                getDate("2025-01-10"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1000,
				StockAmount:         1000,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}

			// Retrieve the trade created under that transaction.
			trades, err := store.ListTrades(ctx, ListTradesOpts{AccountID: invID})
			if err != nil {
				t.Fatalf("ListTrades: %v", err)
			}
			if len(trades) != 1 {
				t.Fatalf("expected 1 trade, got %d", len(trades))
			}

			got, err := store.GetTrade(ctx, trades[0].Id)
			if err != nil {
				t.Fatalf("GetTrade: %v", err)
			}

			if got.TradeType != BuyTrade {
				t.Errorf("TradeType got %v, want BuyTrade", got.TradeType)
			}
			if got.Quantity != 10 {
				t.Errorf("Quantity got %v, want 10", got.Quantity)
			}
			if got.TotalAmount != 1000 {
				t.Errorf("TotalAmount got %v, want 1000", got.TotalAmount)
			}
			if got.AccountID != invID {
				t.Errorf("AccountID got %v, want %v", got.AccountID, invID)
			}
			if got.InstrumentID != instID {
				t.Errorf("InstrumentID got %v, want %v", got.InstrumentID, instID)
			}
			if got.TransactionID != buyID {
				t.Errorf("TransactionID got %v, want %v", got.TransactionID, buyID)
			}
		})
	}
}

func TestGetTrade_notFound(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, _ := newAccountingStoreWithMarketData(t, db.ConnDbName("tradeGetMissing"))

			_, err := store.GetTrade(ctx, 99999)
			if err == nil {
				t.Fatal("expected error for non-existent trade, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ListTrades
// ---------------------------------------------------------------------------

func TestListTrades(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("tradeList"))

			// Two investment accounts, two instruments.
			provID, _ := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			inv1, _ := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "inv1", Currency: currency.USD, Type: InvestmentAccountType})
			inv2, _ := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "inv2", Currency: currency.USD, Type: InvestmentAccountType})
			cash, _ := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "cash", Currency: currency.USD, Type: CashAccountType})
			inst1, _ := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "AAA", Name: "AAA Inc", Currency: currency.USD})
			inst2, _ := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "BBB", Name: "BBB Inc", Currency: currency.USD})

			buy := func(invID, instID uint, qty, amount float64, date string) {
				t.Helper()
				_, err := store.CreateStockBuy(ctx, StockBuy{
					Description: "buy", Date: getDate(date),
					InvestmentAccountID: invID, CashAccountID: cash,
					InstrumentID: instID, Quantity: qty, TotalAmount: amount, StockAmount: amount,
				})
				if err != nil {
					t.Fatalf("CreateStockBuy: %v", err)
				}
			}

			// inv1: buy 10 inst1 Jan, buy 5 inst2 Feb; inv2: buy 8 inst1 Mar
			buy(inv1, inst1, 10, 1000, "2025-01-10")
			buy(inv1, inst2, 5, 500, "2025-02-10")
			buy(inv2, inst1, 8, 800, "2025-03-10")

			// Also create a sell from inv1/inst1 in April.
			_, err := store.CreateStockSell(ctx, StockSell{
				Description: "sell", Date: getDate("2025-04-01"),
				InvestmentAccountID: inv1, CashAccountID: cash,
				InstrumentID: inst1, Quantity: 3, TotalAmount: 330,
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			tcs := []struct {
				name      string
				opts      ListTradesOpts
				wantCount int
			}{
				{"no filter returns all", ListTradesOpts{}, 4},
				{"filter by inv1", ListTradesOpts{AccountID: inv1}, 3},
				{"filter by inv2", ListTradesOpts{AccountID: inv2}, 1},
				{"filter by inst1", ListTradesOpts{InstrumentID: inst1}, 3},
				{"filter by inst2", ListTradesOpts{InstrumentID: inst2}, 1},
				{"filter by date range Jan-Feb", ListTradesOpts{StartDate: getDate("2025-01-01"), EndDate: getDate("2025-02-28")}, 2},
				{"filter by date range Mar only", ListTradesOpts{StartDate: getDate("2025-03-01"), EndDate: getDate("2025-03-31")}, 1},
				{"filter inv1 + inst1", ListTradesOpts{AccountID: inv1, InstrumentID: inst1}, 2},
			}
			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					trades, err := store.ListTrades(ctx, tc.opts)
					if err != nil {
						t.Fatalf("ListTrades: %v", err)
					}
					if len(trades) != tc.wantCount {
						t.Errorf("got %d trades, want %d", len(trades), tc.wantCount)
					}
				})
			}

			// Results are ordered by date ascending.
			t.Run("results ordered by date asc", func(t *testing.T) {
				trades, err := store.ListTrades(ctx, ListTradesOpts{})
				if err != nil {
					t.Fatalf("ListTrades: %v", err)
				}
				for i := 1; i < len(trades); i++ {
					if trades[i].Date.Before(trades[i-1].Date) {
						t.Errorf("trade %d date %v is before trade %d date %v — not sorted asc",
							i, trades[i].Date, i-1, trades[i-1].Date)
					}
				}
			})
		})
	}
}

// ---------------------------------------------------------------------------
// Lot state transitions
// ---------------------------------------------------------------------------

func TestLotStates(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("lotStates"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			// Buy 10 → one lot, Open.
			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description: "Buy 10", Date: getDate("2025-01-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 10, TotalAmount: 1000, StockAmount: 1000,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}

			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: invID})
			if len(lots) != 1 {
				t.Fatalf("after buy: expected 1 lot, got %d", len(lots))
			}
			if lots[0].Status != LotOpen {
				t.Errorf("after buy: lot status got %v, want LotOpen", lots[0].Status)
			}
			if lots[0].Quantity != 10 {
				t.Errorf("after buy: lot qty got %v, want 10", lots[0].Quantity)
			}
			if lots[0].OriginalQty != 10 {
				t.Errorf("after buy: lot original qty got %v, want 10", lots[0].OriginalQty)
			}
			if lots[0].CostPerShare != 100 {
				t.Errorf("after buy: cost per share got %v, want 100", lots[0].CostPerShare)
			}

			// Sell 4 → lot becomes Partial with 6 remaining.
			_, err = store.CreateStockSell(ctx, StockSell{
				Description: "Sell 4", Date: getDate("2025-01-02"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 4, TotalAmount: 400,
			})
			if err != nil {
				t.Fatalf("CreateStockSell (partial): %v", err)
			}

			lots, _ = store.ListLots(ctx, ListLotsOpts{AccountID: invID})
			if len(lots) != 1 {
				t.Fatalf("after partial sell: expected 1 lot, got %d", len(lots))
			}
			if lots[0].Status != LotPartial {
				t.Errorf("after partial sell: lot status got %v, want LotPartial", lots[0].Status)
			}
			if lots[0].Quantity != 6 {
				t.Errorf("after partial sell: lot qty got %v, want 6", lots[0].Quantity)
			}

			// Sell remaining 6 → lot becomes Closed.
			_, err = store.CreateStockSell(ctx, StockSell{
				Description: "Sell 6", Date: getDate("2025-01-03"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 6, TotalAmount: 600,
			})
			if err != nil {
				t.Fatalf("CreateStockSell (full): %v", err)
			}

			lots, _ = store.ListLots(ctx, ListLotsOpts{AccountID: invID})
			if len(lots) != 1 {
				t.Fatalf("after full sell: expected 1 lot, got %d", len(lots))
			}
			if lots[0].Status != LotClosed {
				t.Errorf("after full sell: lot status got %v, want LotClosed", lots[0].Status)
			}
			if lots[0].Quantity != 0 {
				t.Errorf("after full sell: lot qty got %v, want 0", lots[0].Quantity)
			}
			if lots[0].ClosedDate == nil {
				t.Error("after full sell: lot ClosedDate should be set")
			}
		})
	}
}

func TestListLots_Filters(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("lotFilters"))

			provID, _ := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			inv1, _ := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "inv1", Currency: currency.USD, Type: InvestmentAccountType})
			inv2, _ := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "inv2", Currency: currency.USD, Type: InvestmentAccountType})
			cash, _ := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "cash", Currency: currency.USD, Type: CashAccountType})
			inst1, _ := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "AAA", Name: "AAA Inc", Currency: currency.USD})
			inst2, _ := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "BBB", Name: "BBB Inc", Currency: currency.USD})

			// inv1: 10 inst1 (open), 5 inst2 (open); inv2: 8 inst1, sold 4 → partial.
			_, err := store.CreateStockBuy(ctx, StockBuy{Description: "b", Date: getDate("2025-01-01"), InvestmentAccountID: inv1, CashAccountID: cash, InstrumentID: inst1, Quantity: 10, TotalAmount: 1000, StockAmount: 1000})
			if err != nil {
				t.Fatalf("buy1: %v", err)
			}
			_, err = store.CreateStockBuy(ctx, StockBuy{Description: "b", Date: getDate("2025-01-02"), InvestmentAccountID: inv1, CashAccountID: cash, InstrumentID: inst2, Quantity: 5, TotalAmount: 500, StockAmount: 500})
			if err != nil {
				t.Fatalf("buy2: %v", err)
			}
			_, err = store.CreateStockBuy(ctx, StockBuy{Description: "b", Date: getDate("2025-01-03"), InvestmentAccountID: inv2, CashAccountID: cash, InstrumentID: inst1, Quantity: 8, TotalAmount: 800, StockAmount: 800})
			if err != nil {
				t.Fatalf("buy3: %v", err)
			}
			_, err = store.CreateStockSell(ctx, StockSell{Description: "s", Date: getDate("2025-01-04"), InvestmentAccountID: inv2, CashAccountID: cash, InstrumentID: inst1, Quantity: 4, TotalAmount: 400})
			if err != nil {
				t.Fatalf("sell: %v", err)
			}

			open := LotOpen
			partial := LotPartial

			tcs := []struct {
				name      string
				opts      ListLotsOpts
				wantCount int
			}{
				{"no filter", ListLotsOpts{}, 3},
				{"filter inv1", ListLotsOpts{AccountID: inv1}, 2},
				{"filter inv2", ListLotsOpts{AccountID: inv2}, 1},
				{"filter inst1", ListLotsOpts{InstrumentID: inst1}, 2},
				{"filter inst2", ListLotsOpts{InstrumentID: inst2}, 1},
				{"filter status open", ListLotsOpts{Status: &open}, 2},
				{"filter status partial", ListLotsOpts{Status: &partial}, 1},
			}
			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					lots, err := store.ListLots(ctx, tc.opts)
					if err != nil {
						t.Fatalf("ListLots: %v", err)
					}
					if len(lots) != tc.wantCount {
						t.Errorf("got %d lots, want %d", len(lots), tc.wantCount)
					}
				})
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FIFO lot selection
// ---------------------------------------------------------------------------

// TestFIFO_LotSelection verifies that selling consumes the oldest lot first.
func TestFIFO_LotSelection(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("fifoLot"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			// Lot A: 5 shares @ $100 = $500 cost, opened first.
			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description: "Lot A", Date: getDate("2025-01-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 5, TotalAmount: 500, StockAmount: 500,
			})
			if err != nil {
				t.Fatalf("buy lot A: %v", err)
			}

			// Lot B: 5 shares @ $200 = $1000 cost, opened second.
			_, err = store.CreateStockBuy(ctx, StockBuy{
				Description: "Lot B", Date: getDate("2025-01-15"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 5, TotalAmount: 1000, StockAmount: 1000,
			})
			if err != nil {
				t.Fatalf("buy lot B: %v", err)
			}

			// Sell 5 — FIFO must consume lot A entirely.
			sellID, err := store.CreateStockSell(ctx, StockSell{
				Description: "Sell 5", Date: getDate("2025-02-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 5, TotalAmount: 600,
			})
			if err != nil {
				t.Fatalf("sell: %v", err)
			}

			// Cost basis should come entirely from lot A ($500), not lot B ($1000).
			got, err := store.GetTransaction(ctx, sellID)
			if err != nil {
				t.Fatalf("GetTransaction: %v", err)
			}
			s := got.(StockSell)
			if s.CostBasis != 500 {
				t.Errorf("FIFO cost basis got %v, want 500 (lot A)", s.CostBasis)
			}
			if s.RealizedGainLoss != 100 {
				t.Errorf("FIFO realized gain got %v, want 100", s.RealizedGainLoss)
			}

			// Lot A must be Closed, lot B must remain Open.
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: invID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 2 {
				t.Fatalf("expected 2 lots, got %d", len(lots))
			}
			// Lots are ordered open_date ASC → lots[0]=A, lots[1]=B.
			if lots[0].Status != LotClosed {
				t.Errorf("lot A status got %v, want LotClosed", lots[0].Status)
			}
			if lots[1].Status != LotOpen {
				t.Errorf("lot B status got %v, want LotOpen", lots[1].Status)
			}
			if lots[1].Quantity != 5 {
				t.Errorf("lot B remaining qty got %v, want 5", lots[1].Quantity)
			}
		})
	}
}

// TestFIFO_PartialOldestLot verifies FIFO partially consumes the oldest lot before touching the next.
func TestFIFO_PartialOldestLot(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("fifoPartial"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			// Lot A: 10 @ $100.
			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description: "Lot A", Date: getDate("2025-01-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 10, TotalAmount: 1000, StockAmount: 1000,
			})
			if err != nil {
				t.Fatalf("buy lot A: %v", err)
			}
			// Lot B: 10 @ $200.
			_, err = store.CreateStockBuy(ctx, StockBuy{
				Description: "Lot B", Date: getDate("2025-01-15"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 10, TotalAmount: 2000, StockAmount: 2000,
			})
			if err != nil {
				t.Fatalf("buy lot B: %v", err)
			}

			// Sell 15 — exhausts lot A (10) and takes 5 from lot B.
			sellID, err := store.CreateStockSell(ctx, StockSell{
				Description: "Sell 15", Date: getDate("2025-02-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 15, TotalAmount: 1800,
			})
			if err != nil {
				t.Fatalf("sell: %v", err)
			}

			// Cost basis: 10*100 + 5*200 = 2000.
			got, _ := store.GetTransaction(ctx, sellID)
			s := got.(StockSell)
			if s.CostBasis != 2000 {
				t.Errorf("cost basis got %v, want 2000 (10*100 + 5*200)", s.CostBasis)
			}

			lots, _ := store.ListLots(ctx, ListLotsOpts{AccountID: invID})
			if len(lots) != 2 {
				t.Fatalf("expected 2 lots, got %d", len(lots))
			}
			// lots[0]=A (closed), lots[1]=B (partial, 5 remaining).
			if lots[0].Status != LotClosed {
				t.Errorf("lot A: status got %v, want LotClosed", lots[0].Status)
			}
			if lots[1].Status != LotPartial {
				t.Errorf("lot B: status got %v, want LotPartial", lots[1].Status)
			}
			if lots[1].Quantity != 5 {
				t.Errorf("lot B: remaining qty got %v, want 5", lots[1].Quantity)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Position
// ---------------------------------------------------------------------------

func TestPosition_AfterBuy(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("posAfterBuy"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description: "Buy 10 @ 100", Date: getDate("2025-01-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 10, TotalAmount: 1000, StockAmount: 1000,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}

			pos, err := store.GetPosition(ctx, invID, instID)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos.Quantity != 10 {
				t.Errorf("Quantity got %v, want 10", pos.Quantity)
			}
			if pos.CostBasis != 1000 {
				t.Errorf("CostBasis got %v, want 1000", pos.CostBasis)
			}
			if pos.AvgCost != 100 {
				t.Errorf("AvgCost got %v, want 100", pos.AvgCost)
			}
		})
	}
}

func TestPosition_MultipleBuys(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("posMultiBuy"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			// Buy 10 @ $100.
			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description: "Buy 10", Date: getDate("2025-01-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 10, TotalAmount: 1000, StockAmount: 1000,
			})
			if err != nil {
				t.Fatalf("first buy: %v", err)
			}

			// Buy 10 more @ $200.
			_, err = store.CreateStockBuy(ctx, StockBuy{
				Description: "Buy 10 more", Date: getDate("2025-01-15"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 10, TotalAmount: 2000, StockAmount: 2000,
			})
			if err != nil {
				t.Fatalf("second buy: %v", err)
			}

			pos, err := store.GetPosition(ctx, invID, instID)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos.Quantity != 20 {
				t.Errorf("Quantity got %v, want 20", pos.Quantity)
			}
			if pos.CostBasis != 3000 {
				t.Errorf("CostBasis got %v, want 3000", pos.CostBasis)
			}
			// avgCost = 3000 / 20 = 150
			if pos.AvgCost != 150 {
				t.Errorf("AvgCost got %v, want 150", pos.AvgCost)
			}
		})
	}
}

func TestPosition_AfterPartialSell(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("posPartialSell"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description: "Buy 10 @ 100", Date: getDate("2025-01-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 10, TotalAmount: 1000, StockAmount: 1000,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy: %v", err)
			}
			_, err = store.CreateStockSell(ctx, StockSell{
				Description: "Sell 3", Date: getDate("2025-01-02"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 3, TotalAmount: 330,
			})
			if err != nil {
				t.Fatalf("CreateStockSell: %v", err)
			}

			pos, err := store.GetPosition(ctx, invID, instID)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos.Quantity != 7 {
				t.Errorf("Quantity got %v, want 7", pos.Quantity)
			}
			// Remaining cost: 7 shares @ $100 = $700.
			if pos.CostBasis != 700 {
				t.Errorf("CostBasis got %v, want 700", pos.CostBasis)
			}
			if pos.AvgCost != 100 {
				t.Errorf("AvgCost got %v, want 100", pos.AvgCost)
			}

			// Position must appear in ListPositions (qty > 0).
			positions, err := store.ListPositions(ctx, ListPositionsOpts{AccountID: invID})
			if err != nil {
				t.Fatalf("ListPositions: %v", err)
			}
			if len(positions) != 1 {
				t.Errorf("ListPositions got %d, want 1", len(positions))
			}
		})
	}
}

func TestPosition_ZeroAfterFullSell(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("posZero"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description: "Buy 5", Date: getDate("2025-01-01"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 5, TotalAmount: 500, StockAmount: 500,
			})
			if err != nil {
				t.Fatalf("buy: %v", err)
			}
			_, err = store.CreateStockSell(ctx, StockSell{
				Description: "Sell all 5", Date: getDate("2025-01-02"),
				InvestmentAccountID: invID, CashAccountID: cashID,
				InstrumentID: instID, Quantity: 5, TotalAmount: 600,
			})
			if err != nil {
				t.Fatalf("sell: %v", err)
			}

			// Position record still exists but with zero quantity.
			pos, err := store.GetPosition(ctx, invID, instID)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos.Quantity != 0 {
				t.Errorf("Quantity got %v, want 0 after selling all", pos.Quantity)
			}

			// ListPositions excludes zero-quantity positions.
			positions, err := store.ListPositions(ctx, ListPositionsOpts{AccountID: invID})
			if err != nil {
				t.Fatalf("ListPositions: %v", err)
			}
			if len(positions) != 0 {
				t.Errorf("ListPositions got %d, want 0 (zero-qty positions excluded)", len(positions))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Position after grant and transfer
// ---------------------------------------------------------------------------

func TestPosition_AfterGrant(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("posGrant"))
			grantAccID, _, instID := setupStockGrantTransferTest(t, ctx, store, mktStore)

			_, err := store.CreateStockGrant(ctx, StockGrant{
				Description:    "RSU vest",
				Date:           getDate("2025-03-01"),
				AccountID:      grantAccID,
				InstrumentID:   instID,
				Quantity:       100,
				FairMarketValue: 50,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			pos, err := store.GetPosition(ctx, grantAccID, instID)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos.Quantity != 100 {
				t.Errorf("Quantity got %v, want 100", pos.Quantity)
			}
			// FMV $50 × 100 shares = $5000 cost basis.
			if pos.CostBasis != 5000 {
				t.Errorf("CostBasis got %v, want 5000", pos.CostBasis)
			}
			if pos.AvgCost != 50 {
				t.Errorf("AvgCost got %v, want 50", pos.AvgCost)
			}

			// Lot must be Open.
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: grantAccID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 1 {
				t.Fatalf("expected 1 lot, got %d", len(lots))
			}
			if lots[0].Status != LotOpen {
				t.Errorf("lot status got %v, want LotOpen", lots[0].Status)
			}
			if lots[0].CostPerShare != 50 {
				t.Errorf("lot CostPerShare got %v, want 50", lots[0].CostPerShare)
			}
		})
	}
}

func TestPosition_AfterTransfer(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("posTransfer"))

			// Create two investment accounts and an instrument.
			provID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			invA, err := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "Investment A", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			invB, err := store.CreateAccount(ctx, Account{AccountProviderID: provID, Name: "Investment B", Currency: currency.USD, Type: InvestmentAccountType})
			if err != nil {
				t.Fatal(err)
			}
			instID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{Symbol: "RSU", Name: "Company RSU", Currency: currency.USD})
			if err != nil {
				t.Fatal(err)
			}

			// Grant 100 shares @ FMV $50 into investment A.
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:    "RSU grant",
				Date:           getDate("2025-03-01"),
				AccountID:      invA,
				InstrumentID:   instID,
				Quantity:       100,
				FairMarketValue: 50,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// Transfer 40 shares from investment A to investment B.
			_, err = store.CreateStockTransfer(ctx, StockTransfer{
				Description:     "transfer",
				Date:            getDate("2025-03-15"),
				SourceAccountID: invA,
				TargetAccountID: invB,
				InstrumentID:    instID,
				Quantity:        40,
			})
			if err != nil {
				t.Fatalf("CreateStockTransfer: %v", err)
			}

			// Source: 60 shares remaining, cost = 60 * $50 = $3000.
			srcPos, err := store.GetPosition(ctx, invA, instID)
			if err != nil {
				t.Fatalf("GetPosition (source): %v", err)
			}
			if srcPos.Quantity != 60 {
				t.Errorf("source Quantity got %v, want 60", srcPos.Quantity)
			}
			if srcPos.CostBasis != 3000 {
				t.Errorf("source CostBasis got %v, want 3000", srcPos.CostBasis)
			}

			// Target: 40 shares, cost basis preserved at $50/share = $2000.
			tgtPos, err := store.GetPosition(ctx, invB, instID)
			if err != nil {
				t.Fatalf("GetPosition (target): %v", err)
			}
			if tgtPos.Quantity != 40 {
				t.Errorf("target Quantity got %v, want 40", tgtPos.Quantity)
			}
			if tgtPos.CostBasis != 2000 {
				t.Errorf("target CostBasis got %v, want 2000 (cost basis preserved)", tgtPos.CostBasis)
			}
			if tgtPos.AvgCost != 50 {
				t.Errorf("target AvgCost got %v, want 50 (same cost per share)", tgtPos.AvgCost)
			}

			// ListAllPositions returns both accounts.
			all, err := store.ListAllPositions(ctx)
			if err != nil {
				t.Fatalf("ListAllPositions: %v", err)
			}
			if len(all) != 2 {
				t.Errorf("ListAllPositions got %d positions, want 2", len(all))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Manual lot allocation
// ---------------------------------------------------------------------------

func TestManualLotAllocation(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("manualLotAlloc"))
			invID, cashID, instID := setupStockBuySellTest(t, ctx, store, mktStore)

			// Buy lot 1: 10 shares at $100 each = $1000 cost basis
			_, err := store.CreateStockBuy(ctx, StockBuy{
				Description:         "Buy lot 1",
				Date:                getDate("2025-01-10"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         1000,
				StockAmount:         1000,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy lot1: %v", err)
			}

			// Buy lot 2: 10 shares at $200 each = $2000 cost basis
			_, err = store.CreateStockBuy(ctx, StockBuy{
				Description:         "Buy lot 2",
				Date:                getDate("2025-02-10"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            10,
				TotalAmount:         2000,
				StockAmount:         2000,
			})
			if err != nil {
				t.Fatalf("CreateStockBuy lot2: %v", err)
			}

			// Get the two lots so we can pick lot 2 explicitly.
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: invID, InstrumentID: instID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 2 {
				t.Fatalf("expected 2 lots, got %d", len(lots))
			}
			// Lots are ordered by open_date ASC: lots[0] = lot1 ($100/share), lots[1] = lot2 ($200/share).
			lot2ID := lots[1].Id

			// Sell 5 shares explicitly from lot 2 (cost basis = 5 * $200 = $1000).
			_, err = store.CreateStockSell(ctx, StockSell{
				Description:         "Sell from lot 2",
				Date:                getDate("2025-03-01"),
				InvestmentAccountID: invID,
				CashAccountID:       cashID,
				InstrumentID:        instID,
				Quantity:            5,
				TotalAmount:         1500, // proceeds
				LotSelections: []LotSelection{
					{LotID: lot2ID, Quantity: 5},
				},
			})
			if err != nil {
				t.Fatalf("CreateStockSell with manual lot: %v", err)
			}

			// Verify lot 2 is now partial with 5 shares remaining.
			updatedLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: invID, InstrumentID: instID})
			if err != nil {
				t.Fatalf("ListLots after sell: %v", err)
			}
			var lot2After *Lot
			for i := range updatedLots {
				if updatedLots[i].Id == lot2ID {
					lot2After = &updatedLots[i]
					break
				}
			}
			if lot2After == nil {
				t.Fatal("lot2 not found after sell")
			}
			if lot2After.Status != LotPartial {
				t.Errorf("lot2 status got %v, want LotPartial", lot2After.Status)
			}
			if lot2After.Quantity != 5 {
				t.Errorf("lot2 remaining quantity got %v, want 5", lot2After.Quantity)
			}

			// Lot 1 must be untouched (FIFO would have consumed lot 1 first).
			var lot1After *Lot
			for i := range updatedLots {
				if updatedLots[i].Id == lots[0].Id {
					lot1After = &updatedLots[i]
					break
				}
			}
			if lot1After == nil {
				t.Fatal("lot1 not found after sell")
			}
			if lot1After.Status != LotOpen {
				t.Errorf("lot1 status got %v, want LotOpen (should be untouched)", lot1After.Status)
			}
			if lot1After.Quantity != 10 {
				t.Errorf("lot1 quantity got %v, want 10 (should be untouched)", lot1After.Quantity)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// vestLots
// ---------------------------------------------------------------------------

// setupVestLotsTest creates accounts, instrument, grants shares, and vests 60 of them.
// Returns the unvested account ID, investment account ID, instrument ID, and vest date.
func setupVestLotsTest(t *testing.T, ctx context.Context, store *Store, mktStore *marketdata.Store, dbName string) (
	unvestedID, investID, instID uint, vestDate time.Time,
) {
	t.Helper()
	provID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
	if err != nil {
		t.Fatal(err)
	}
	unvestedID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: provID, Name: "RSU Unvested",
		Currency: currency.USD, Type: RestrictedStockAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}
	investID, err = store.CreateAccount(ctx, Account{
		AccountProviderID: provID, Name: "Broker Vested",
		Currency: currency.USD, Type: InvestmentAccountType,
	})
	if err != nil {
		t.Fatal(err)
	}
	instID, err = mktStore.CreateInstrument(ctx, marketdata.Instrument{
		Symbol: "VEST", Name: "Vest Corp", Currency: currency.USD,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.CreateStockGrant(ctx, StockGrant{
		Description: "RSU grant", Date: getDate("2025-01-15"),
		AccountID: unvestedID, InstrumentID: instID,
		Quantity: 100, FairMarketValue: 50,
	})
	if err != nil {
		t.Fatalf("CreateStockGrant: %v", err)
	}

	lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instID})
	if err != nil {
		t.Fatalf("ListLots: %v", err)
	}
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	sourceLotID := lots[0].Id

	vestDate = getDate("2025-06-01")
	vestingPrice := 75.0

	err = store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		tx := dbTransaction{Description: "vest event", Date: vestDate, Type: StockVestTransaction}
		if err := dbTx.Create(&tx).Error; err != nil {
			return err
		}
		trade := dbTrade{
			TransactionID: tx.Id, AccountID: investID, InstrumentID: instID,
			TradeType: TransferInTrade, Quantity: 60, PricePerShare: vestingPrice,
			TotalAmount: 60 * vestingPrice, Currency: "USD", Date: vestDate,
		}
		if err := dbTx.Create(&trade).Error; err != nil {
			return err
		}
		return store.vestLots(ctx, dbTx, []LotSelection{
			{LotID: sourceLotID, Quantity: 60},
		}, vestingPrice, vestDate, trade.Id, unvestedID, investID, instID)
	})
	if err != nil {
		t.Fatalf("vestLots transaction: %v", err)
	}
	return unvestedID, investID, instID, vestDate
}

func verifyVestLotsSource(t *testing.T, ctx context.Context, store *Store, unvestedID, instID uint) {
	t.Helper()
	srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instID})
	if err != nil {
		t.Fatalf("ListLots (source): %v", err)
	}
	if len(srcLots) != 1 {
		t.Fatalf("expected 1 source lot, got %d", len(srcLots))
	}
	src := srcLots[0]
	if src.Quantity != 40 {
		t.Errorf("source lot quantity got %v, want 40", src.Quantity)
	}
	if src.CostPerShare != 50 {
		t.Errorf("source lot CostPerShare got %v, want 50", src.CostPerShare)
	}
	if src.CostBasis != 2000 {
		t.Errorf("source lot CostBasis got %v, want 2000", src.CostBasis)
	}
	if src.Status != LotPartial {
		t.Errorf("source lot status got %v, want LotPartial", src.Status)
	}
}

func verifyVestLotsTarget(t *testing.T, ctx context.Context, store *Store, investID, instID uint, vestDate time.Time) {
	t.Helper()
	tgtLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: investID, InstrumentID: instID})
	if err != nil {
		t.Fatalf("ListLots (target): %v", err)
	}
	if len(tgtLots) != 1 {
		t.Fatalf("expected 1 target lot, got %d", len(tgtLots))
	}
	tgt := tgtLots[0]
	if tgt.Quantity != 60 {
		t.Errorf("target lot quantity got %v, want 60", tgt.Quantity)
	}
	if tgt.CostPerShare != 75 {
		t.Errorf("target lot CostPerShare got %v, want 75", tgt.CostPerShare)
	}
	if tgt.CostBasis != 4500 {
		t.Errorf("target lot CostBasis got %v, want 4500", tgt.CostBasis)
	}
	if tgt.Status != LotOpen {
		t.Errorf("target lot status got %v, want LotOpen", tgt.Status)
	}
	if !tgt.OpenDate.Equal(vestDate) {
		t.Errorf("target lot OpenDate got %v, want %v", tgt.OpenDate, vestDate)
	}
}

func TestVestLots(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("vestLots"))
			unvestedID, investID, instID, vestDate := setupVestLotsTest(t, ctx, store, mktStore, "vestLots")

			t.Run("verify source lot", func(t *testing.T) {
				verifyVestLotsSource(t, ctx, store, unvestedID, instID)
			})
			t.Run("verify target lot", func(t *testing.T) {
				verifyVestLotsTarget(t, ctx, store, investID, instID, vestDate)
			})
		})
	}
}

func TestForfeitLots(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("forfeitLots"))

			// Create unvested account.
			provID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			unvestedID, err := store.CreateAccount(ctx, Account{
				AccountProviderID: provID,
				Name:              "RSU Unvested",
				Currency:          currency.USD,
				Type:              RestrictedStockAccountType,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Create instrument.
			instID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{
				Symbol:   "FORF",
				Name:     "Forfeit Corp",
				Currency: currency.USD,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Grant 100 shares at FMV $50 into unvested account.
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "RSU grant",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedID,
				InstrumentID:    instID,
				Quantity:        100,
				FairMarketValue: 50,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// List lots to get the lot ID.
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 1 {
				t.Fatalf("expected 1 lot, got %d", len(lots))
			}
			sourceLotID := lots[0].Id

			// Create a dbTransaction + dbTrade for FK safety.
			forfeitDate := getDate("2025-06-01")

			err = store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
				tx := dbTransaction{
					Description: "forfeit event",
					Date:        forfeitDate,
					Type:        StockForfeitTransaction,
				}
				if err := dbTx.Create(&tx).Error; err != nil {
					return err
				}
				trade := dbTrade{
					TransactionID: tx.Id,
					AccountID:     unvestedID,
					InstrumentID:  instID,
					TradeType:     ForfeitTrade,
					Quantity:      40,
					Date:          forfeitDate,
				}
				if err := dbTx.Create(&trade).Error; err != nil {
					return err
				}

				// Call forfeitLots: forfeit 40 of the 100 shares.
				return store.forfeitLots(ctx, dbTx, []LotSelection{
					{LotID: sourceLotID, Quantity: 40},
				}, forfeitDate, trade.Id, unvestedID, instID)
			})
			if err != nil {
				t.Fatalf("forfeitLots transaction: %v", err)
			}

			// ---- Verify source lot: 60 remaining at original $50/share ----
			srcLots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedID, InstrumentID: instID})
			if err != nil {
				t.Fatalf("ListLots (source): %v", err)
			}
			if len(srcLots) != 1 {
				t.Fatalf("expected 1 source lot, got %d", len(srcLots))
			}
			src := srcLots[0]
			if src.Quantity != 60 {
				t.Errorf("source lot quantity got %v, want 60", src.Quantity)
			}
			if src.CostPerShare != 50 {
				t.Errorf("source lot CostPerShare got %v, want 50", src.CostPerShare)
			}
			if src.CostBasis != 3000 {
				t.Errorf("source lot CostBasis got %v, want 3000", src.CostBasis)
			}
			if src.Status != LotPartial {
				t.Errorf("source lot status got %v, want LotPartial", src.Status)
			}

			// ---- Verify no new lots were created (forfeit doesn't create target lots) ----
			// No target account to check — forfeited shares just disappear.
		})
	}
}

func TestVestLots_RejectsCrossAccountLot(t *testing.T) {
	for _, db := range testdbs.DBs() {
		t.Run(db.DbType(), func(t *testing.T) {
			ctx := t.Context()
			store, mktStore := newAccountingStoreWithMarketData(t, db.ConnDbName("vestLotsCrossAcct"))

			// Create two unvested accounts (A and B) and one investment account.
			provID, err := store.CreateAccountProvider(ctx, AccountProvider{Name: "Broker"})
			if err != nil {
				t.Fatal(err)
			}
			unvestedA, err := store.CreateAccount(ctx, Account{
				AccountProviderID: provID,
				Name:              "Unvested A",
				Currency:          currency.USD,
				Type:              RestrictedStockAccountType,
			})
			if err != nil {
				t.Fatal(err)
			}
			unvestedB, err := store.CreateAccount(ctx, Account{
				AccountProviderID: provID,
				Name:              "Unvested B",
				Currency:          currency.USD,
				Type:              RestrictedStockAccountType,
			})
			if err != nil {
				t.Fatal(err)
			}
			investID, err := store.CreateAccount(ctx, Account{
				AccountProviderID: provID,
				Name:              "Broker Vested",
				Currency:          currency.USD,
				Type:              InvestmentAccountType,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Create instrument.
			instID, err := mktStore.CreateInstrument(ctx, marketdata.Instrument{
				Symbol:   "XACCT",
				Name:     "Cross Account Corp",
				Currency: currency.USD,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Grant 100 shares into account A.
			_, err = store.CreateStockGrant(ctx, StockGrant{
				Description:     "Grant A",
				Date:            getDate("2025-01-15"),
				AccountID:       unvestedA,
				InstrumentID:    instID,
				Quantity:        100,
				FairMarketValue: 50,
			})
			if err != nil {
				t.Fatalf("CreateStockGrant: %v", err)
			}

			// Get the lot ID from account A.
			lots, err := store.ListLots(ctx, ListLotsOpts{AccountID: unvestedA, InstrumentID: instID})
			if err != nil {
				t.Fatalf("ListLots: %v", err)
			}
			if len(lots) != 1 {
				t.Fatalf("expected 1 lot, got %d", len(lots))
			}
			lotFromA := lots[0].Id

			// Try to vest the lot from account A but pass account B as the source.
			vestDate := getDate("2025-06-01")
			vestingPrice := 75.0

			err = store.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
				tx := dbTransaction{
					Description: "cross-account vest",
					Date:        vestDate,
					Type:        StockVestTransaction,
				}
				if err := dbTx.Create(&tx).Error; err != nil {
					return err
				}
				trade := dbTrade{
					TransactionID: tx.Id,
					AccountID:     investID,
					InstrumentID:  instID,
					TradeType:     TransferInTrade,
					Quantity:      50,
					PricePerShare: vestingPrice,
					Date:          vestDate,
				}
				if err := dbTx.Create(&trade).Error; err != nil {
					return err
				}

				// Pass unvestedB as sourceAccountID, but lotFromA belongs to unvestedA.
				return store.vestLots(ctx, dbTx, []LotSelection{
					{LotID: lotFromA, Quantity: 50},
				}, vestingPrice, vestDate, trade.Id, unvestedB, investID, instID)
			})

			if err == nil {
				t.Fatal("expected validation error for cross-account lot, got nil")
			}
			var validationErr ErrValidation
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ErrValidation, got %T: %v", err, err)
			}
			if !strings.Contains(string(validationErr), "does not belong to the source account") {
				t.Errorf("unexpected error message: %q", validationErr)
			}
		})
	}
}
