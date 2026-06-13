package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecideFeatureSettings(t *testing.T) {
	tcs := []struct {
		name             string
		rsu, instruments bool
		facts            dbFeatureFacts
		wantRsu          bool
		wantInstruments  bool
		wantAutoEnabled  []featureAutoEnable
	}{
		{
			name:            "both configured on: nothing auto-enabled",
			rsu:             true,
			instruments:     true,
			wantRsu:         true,
			wantInstruments: true,
			wantAutoEnabled: nil,
		},
		{
			name:            "both off, no db facts: stay off",
			wantRsu:         false,
			wantInstruments: false,
			wantAutoEnabled: nil,
		},
		{
			name:            "off + restricted-stock account: auto-enables RSU, which cascades to instruments",
			facts:           dbFeatureFacts{hasRestrictedStock: true, hasInvestment: true},
			wantRsu:         true,
			wantInstruments: true,
			wantAutoEnabled: []featureAutoEnable{
				{"rsu",
					"config discrepancy: Settings.Rsu is false but database contains restricted-stock accounts; enabling Rsu",
					"db_has_restricted_stock_accounts"},
				{"investmentInstruments",
					"config discrepancy: Settings.Rsu is true but Settings.InvestmentInstruments is false; enabling InvestmentInstruments (required by RSU)",
					"rsu_requires_instruments"},
			},
		},
		{
			name:            "off + vest/forfeit transactions: auto-enables RSU, cascades to instruments",
			facts:           dbFeatureFacts{hasVestForfeit: true},
			wantRsu:         true,
			wantInstruments: true,
			wantAutoEnabled: []featureAutoEnable{
				{"rsu",
					"config discrepancy: Settings.Rsu is false but database contains vest/forfeit transactions; enabling Rsu",
					"db_has_vest_forfeit_transactions"},
				{"investmentInstruments",
					"config discrepancy: Settings.Rsu is true but Settings.InvestmentInstruments is false; enabling InvestmentInstruments (required by RSU)",
					"rsu_requires_instruments"},
			},
		},
		{
			name:            "off + investment account only: auto-enables instruments, not RSU",
			facts:           dbFeatureFacts{hasInvestment: true},
			wantRsu:         false,
			wantInstruments: true,
			wantAutoEnabled: []featureAutoEnable{
				{"investmentInstruments",
					"config discrepancy: Settings.InvestmentInstruments is false but database contains investment or restricted-stock accounts; enabling InvestmentInstruments",
					"db_has_investment_accounts"},
			},
		},
		{
			name:            "RSU configured on, instruments off: only instruments auto-enabled (RSU is from config)",
			rsu:             true,
			wantRsu:         true,
			wantInstruments: true,
			wantAutoEnabled: []featureAutoEnable{
				{"investmentInstruments",
					"config discrepancy: Settings.Rsu is true but Settings.InvestmentInstruments is false; enabling InvestmentInstruments (required by RSU)",
					"rsu_requires_instruments"},
			},
		},
		{
			name:            "instruments configured on, RSU off + restricted stock: only RSU auto-enabled",
			instruments:     true,
			facts:           dbFeatureFacts{hasRestrictedStock: true, hasInvestment: true},
			wantRsu:         true,
			wantInstruments: true,
			wantAutoEnabled: []featureAutoEnable{
				{"rsu",
					"config discrepancy: Settings.Rsu is false but database contains restricted-stock accounts; enabling Rsu",
					"db_has_restricted_stock_accounts"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			gotRsu, gotInstruments, gotAuto := decideFeatureSettings(tc.rsu, tc.instruments, tc.facts)
			if gotRsu != tc.wantRsu {
				t.Errorf("effective RSU: got %v, want %v", gotRsu, tc.wantRsu)
			}
			if gotInstruments != tc.wantInstruments {
				t.Errorf("effective InvestmentInstruments: got %v, want %v", gotInstruments, tc.wantInstruments)
			}
			if diff := cmp.Diff(tc.wantAutoEnabled, gotAuto, cmp.AllowUnexported(featureAutoEnable{})); diff != "" {
				t.Errorf("autoEnabled mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
