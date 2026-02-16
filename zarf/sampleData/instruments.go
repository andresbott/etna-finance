package main

import (
	"fmt"
)

type instrumentResponse struct {
	ID       int    `json:"id"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

// createInstrument sends a POST request to create an instrument and returns the generated id.
func createInstrument(baseURL string, inst Instrument) (int, error) {
	url := fmt.Sprintf("%s/api/v0/fin/instrument", baseURL)
	var resp instrumentResponse
	err := postJSON(url, inst, &resp)
	if err != nil {
		return 0, err
	}
	return resp.ID, nil
}

// findInstrumentID returns the instrument id by symbol (must have been created and stored in Instruments).
func findInstrumentID(symbol string) (int, error) {
	for _, inst := range Instruments {
		if inst.Symbol == symbol {
			if inst.ID == 0 {
				return 0, fmt.Errorf("instrument %q has no id (not yet created)", symbol)
			}
			return inst.ID, nil
		}
	}
	return 0, fmt.Errorf("instrument %q not found", symbol)
}
