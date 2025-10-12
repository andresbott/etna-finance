package main

var Accounts = []Provider{
	{
		Name:        "Cash Store",
		Description: "Stored in Cash",
		Accounts: []Account{
			{

				Name:        "Wallet",
				Description: "Physical wallet",
				Currency:    "CHF",
				Type:        "cash",
			},
		},
	},
	{
		Name:        "Main Bank",
		Description: "Sample provider",
		Accounts: []Account{
			{

				Name:        "Main Account",
				Description: "Primary checking",
				Currency:    "CHF",
				Type:        "checkin",
			},
			{
				Name:        "Eur expenses",
				Description: "",
				Currency:    "EUR",
				Type:        "checkin",
			},
		},
	},
	{
		Name:        "Secondary Bank",
		Description: "",
		Accounts: []Account{
			{
				Name:        "Another Bank",
				Description: "secondary checking",
				Currency:    "CHF",
				Type:        "checkin",
			},
		},
	},
}

var ExpenseCategories = []Category{
	{
		Name:        "Office Expenses",
		Description: "All office related expenses",
		Children: []Category{
			{
				Name:        "Stationery",
				Description: "Pens, papers, etc.",
			},
			{
				Name:        "Software",
				Description: "SaaS subscriptions",
			},
		},
	},
}

var IncomeCategories = []Category{
	{
		Name:        "Sales",
		Description: "Revenue from sales",
		Children: []Category{
			{
				Name:        "Online Sales",
				Description: "Revenue from online store",
			},
			{
				Name:        "Retail Sales",
				Description: "Revenue from physical store",
			},
		},
	},
}

var Entries = []EntryDefinition{
	// Expense entries
	{
		Description:  "Office supplies purchase",
		DaysDelta:    -1, // yesterday
		Type:         "expense",
		Amount:       150.50,
		ProviderName: "Banana",
		AccountName:  "Checking Account",
		CategoryName: "Stationery",
	},
	{
		Description:  "Monthly software subscription",
		DaysDelta:    -5, // 5 days ago
		Type:         "expense",
		Amount:       29.99,
		ProviderName: "Banana",
		AccountName:  "Checking Account",
		CategoryName: "Software",
	},
	// Income entries
	{
		Description:  "Online store sales",
		DaysDelta:    -2, // 2 days ago
		Type:         "income",
		Amount:       750.00,
		ProviderName: "Banana",
		AccountName:  "Checking Account",
		CategoryName: "Online Sales",
	},
	{
		Description:  "Physical store sales",
		DaysDelta:    -3, // 3 days ago
		Type:         "income",
		Amount:       425.75,
		ProviderName: "Banana",
		AccountName:  "Savings Account",
		CategoryName: "Retail Sales",
	},
	// Transfer entry
	{
		Description:    "Transfer from checking to savings",
		DaysDelta:      -4, // 4 days ago
		Type:           "transfer",
		OriginAmount:   200.00,
		OriginProvider: "Banana",
		OriginAccount:  "Checking Account",
		TargetAmount:   200.00,
		TargetProvider: "Banana",
		TargetAccount:  "Savings Account",
	},
}
