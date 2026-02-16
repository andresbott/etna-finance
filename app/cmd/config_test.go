package cmd

import (
	"testing"
)

func TestValidateSettings(t *testing.T) {
	valid := AppSettings{
		DateFormat:   "YYYY-MM-DD",
		MainCurrency: "CHF",
		Currencies:   []string{"CHF"},
		Instruments:  false,
	}

	tests := []struct {
		name    string
		modify  func(AppSettings) AppSettings
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(s AppSettings) AppSettings { return s },
			wantErr: false,
		},
		// Date format
		{
			name:    "valid date DD/MM/YYYY",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "DD/MM/YYYY"; return s },
			wantErr: false,
		},
		{
			name:    "valid date MM.DD.YYYY",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "MM.DD.YYYY"; return s },
			wantErr: false,
		},
		{
			name:    "valid date DD-MM-YY",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "DD-MM-YY"; return s },
			wantErr: false,
		},
		{
			name:    "valid date YY/MM/DD",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "YY/MM/DD"; return s },
			wantErr: false,
		},
		{
			name:    "invalid date format empty",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = ""; return s },
			wantErr: true,
		},
		{
			name:    "invalid date format go style",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "2006-01-02"; return s },
			wantErr: true,
		},
		{
			name:    "invalid date format lowercase",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "yyyy-mm-dd"; return s },
			wantErr: true,
		},
		{
			name:    "invalid date format missing day",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "YYYY-MM"; return s },
			wantErr: true,
		},
		{
			name:    "invalid date format missing month",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "YYYY-DD-DD"; return s },
			wantErr: true,
		},
		{
			name:    "invalid date format missing year",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "DD-MM-DD"; return s },
			wantErr: true,
		},
		{
			name:    "invalid date format bad separator",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "YYYY_MM_DD"; return s },
			wantErr: true,
		},
		{
			name:    "invalid date format random string",
			modify:  func(s AppSettings) AppSettings { s.DateFormat = "hello"; return s },
			wantErr: true,
		},
		// Currencies
		{
			name:    "valid multiple currencies",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"CHF", "USD", "EUR"}; return s },
			wantErr: false,
		},
		{
			name:    "invalid currency lowercase",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"chf"}; s.MainCurrency = "chf"; return s },
			wantErr: true,
		},
		{
			name:    "invalid currency too short",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"CH"}; s.MainCurrency = "CH"; return s },
			wantErr: true,
		},
		{
			name:    "invalid currency too long",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"CHFF"}; s.MainCurrency = "CHFF"; return s },
			wantErr: true,
		},
		{
			name:    "invalid currency with numbers",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"CH1"}; s.MainCurrency = "CH1"; return s },
			wantErr: true,
		},
		{
			name:    "invalid currency empty string",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{""}; s.MainCurrency = ""; return s },
			wantErr: true,
		},
		{
			name:    "invalid empty currencies list",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{}; return s },
			wantErr: true,
		},
		{
			name: "invalid main currency not in list",
			modify: func(s AppSettings) AppSettings {
				s.MainCurrency = "USD"
				s.Currencies = []string{"CHF", "EUR"}
				return s
			},
			wantErr: true,
		},
		{
			name:    "invalid one bad currency in list",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"CHF", "bad", "EUR"}; return s },
			wantErr: true,
		},
		{
			name:    "invalid main currency format",
			modify:  func(s AppSettings) AppSettings { s.MainCurrency = "usd"; return s },
			wantErr: true,
		},
		// ISO 4217 specific: valid format but not a real currency
		{
			name:    "invalid currency not in ISO 4217",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"ZZZ"}; s.MainCurrency = "ZZZ"; return s },
			wantErr: true,
		},
		{
			name:    "valid historical currency BAM",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"CHF", "BAM"}; return s },
			wantErr: false,
		},
		{
			name:    "valid currency JPY",
			modify:  func(s AppSettings) AppSettings { s.Currencies = []string{"CHF", "JPY", "GBP"}; return s },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.modify(valid)
			err := validateSettings(s)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
