package finance

import (
	"context"
	"time"
)

type EntryType int

const (
	Income EntryType = iota
	Expense
	Transfer
)

type Transaction struct {
	Id          uint
	Date        time.Time
	Description string
}

func (store *Store) CreateIncome(ctx context.Context, accountID uint, amount float64, quantity int, description string) error {

	// get the account type and check if it a cash type
	//if quantity >0 its considerd a stock account

	tx := dbTransaction{
		Description: description,
		Date:        time.Now(),
		Entries: []dbEntry{
			{AccountID: accountID, Amount: amount, entryTyp: Income},
		},
	}
	if err := store.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return err
	}
	return nil
}

//// CreateExpense adds an expense entry
//func CreateExpense(accountID uint, amount float64, description string) (*Transaction, error) {
//	tx := Transaction{
//		Description: description,
//		Date:        time.Now(),
//		Entries: []Entry{
//			{AccountID: accountID, Amount: -amount, entryTyoe expense }, // Credit expense account
//		},
//	}
//	if err := db.Create(&tx).Error; err != nil {
//		return nil, err
//	}
//	return &tx, nil
//}
//
//// CreateTransfer creates a transfer between two accounts
//func CreateTransfer(fromAccountID, toAccountID uint, amount float64, description string) (*Transaction, error) {
//	tx := Transaction{
//		Description: description,
//		Date:        time.Now(),
//		Entries: []Entry{
//			{AccountID: fromAccountID, Amount: -amount, entryTyoe transferOut}, // Credit from account
//			{AccountID: toAccountID, Amount: amount entryTyoe transferIN },    // Debit to account
//		},
//	}
//	if err := db.Create(&tx).Error; err != nil {
//		return nil, err
//	}
//	return &tx, nil
//}

//func BuyStock(fromAccountID, toAccountID uint, amount float64, description string) (*Transaction, error) {
//	tx := Transaction{
//		Description: description,
//		Date:        time.Now(),
//		Entries: []Entry{
//			{AccountID: SourceAccount, Amount: -amount, entryTyoe transferOutStock  }, // Credit from account
//			{AccountID: StockAccount, Amount: amount, stock ammount, entryTyoe transferinStock },    // Debit to account
//		},
//	}
//	if err := db.Create(&tx).Error; err != nil {
//		return nil, err
//	}
//	return &tx, nil
//}
//
//func sellStock(fromAccountID, toAccountID uint, amount float64, description string) (*Transaction, error) {
//	tx := Transaction{
//		Description: description,
//		Date:        time.Now(),
//		Entries: []Entry{
//			{AccountID: cash.ID, Amount: 1000  entryTyoe transferInStock},       // debit cash
//			{AccountID: realizedGain.ID, Amount: 200, entryTyoe income },// credit gain income
//			{AccountID: stockXYZ.ID, Amount: -1000,  entryTyoe transferoutStock},  // credit stock asset
//		},
//	}
//	if err := db.Create(&tx).Error; err != nil {
//		return nil, err
//	}
//	return &tx, nil
//}
