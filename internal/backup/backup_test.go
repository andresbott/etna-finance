package backup

import (
	"github.com/andresbott/etna/internal/accounting"
	"golang.org/x/text/currency"
	"sort"
	"testing"
)

func TestBackupRestoreV1(t *testing.T) {

}

const tenant1 = "tenant1"

func sampleData(t *testing.T, store *accounting.Store, data map[int]accounting.Transaction) {

	// =========================================
	// create accounts providers
	// =========================================

	accProviderId, err := store.CreateAccountProvider(t.Context(), accounting.AccountProvider{Name: "p1"}, tenant1)
	if err != nil {
		t.Fatalf("error creating provider 1: %v", err)
	}
	// =========================================
	// create accounts
	// =========================================
	Accs := []accounting.Account{
		{AccountProviderID: accProviderId, Name: "acc1", Currency: currency.EUR, Type: accounting.CashAccountType},
		{AccountProviderID: accProviderId, Name: "acc2", Currency: currency.USD, Type: accounting.CashAccountType},
		{AccountProviderID: accProviderId, Name: "acc3", Currency: currency.CHF, Type: accounting.CashAccountType},
	}
	for _, acc := range Accs {
		_, err = store.CreateAccount(t.Context(), acc, tenant1)
		if err != nil {
			t.Fatalf("error creating account 1: %v", err)
		}
	}
	// =========================================
	// Create Transactions
	// =========================================

	// transform the map into a sorted array to have predictable test results
	var dataKeys []int
	for k := range data {
		dataKeys = append(dataKeys, k)
	}
	sort.Ints(dataKeys)

	var dataAr = make([]accounting.Transaction, len(data))
	for i, k := range dataKeys {
		dataAr[i] = data[k]
	}

	for _, tx := range dataAr {
		_, err = store.CreateTransaction(t.Context(), tx, tenant1)
		if err != nil {
			t.Fatalf("error creating account: %v", err)
		}
	}
}
