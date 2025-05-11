# TODO
* backend to filter entries with account IDs
* edit dialogs to work
* categories
  * crud on categories both backend and frondend
  * modify dialogs to support slection of categorie4s

# Brain dump

* movements
  * income uses:  target account, amount, currency and category
    * can be to a stock account -> in that case also capture amount of stocks
  * expense uses: target account, amount, currency and category
    * can be to a stock account ( not common) -> in that case also capture amount of stocks
  * transfer uses: origin/target account, origin and target amount, currency, NO CATEGORY
    * don't allow to move between stock money accounts
  * Buy stocks / Sell stocks:
    * 2 separate dialogs
    * Dialog similar to transfer, but contains the amount of stocks
      * here tha backend calculates the stock value based on money and amount
  * Stock Value (extra Table) > auto import in the future
    * add an extra entry that updates the value of a stock
      * this is a time progressing entry to also see the evolution
* account listing: sort accounts by amount of movemnts, putting the one with most on the top
  * alternative add sort
  

* dashbaord
  * in the financial overview => time graph use main category aggregated
  * in the accound distribution use aggregated category
  * in the accounts status view makea  tree representation
  * create another view of income to exprense ratio
    * extrapolate savings
* currency excahnge
  * we need to keep track of currency exchange 
  * user setting with main desired currency
* category
  * tree
  * name
  * icons
* CRUD Account entry, 
  * proper date handling, using unix timestamp
  * proper currency handling (using account currency (?))
  * classification => tree  
  * add filter for account
* Report
  * Generate tree balance
  * View with current status
  * view with progression table
* real estate prediction
* option to adjust for inflation


## future ideas
* allow to upload files, e.g. invoices and link them to expenses
  * needs a blob store
* add feature to import csv  
  * define csv structure column name to field mapping, ( For now hardcoded in the backend) 
    * frontend to make a mapping between required and optional fields and the column name of a csv
    * optionally upload a csv to the front end to pre-populate sources
    * store in the backend a template + a map of source=>target fields
  * define concept to classification mapping
  * allow preview:
    * step 1 upload, the backend parses ( iterates over the csv templates to find the best match) 
      the content and retuns a json
    * step 2 the fronend gets a digested view of what the changes will be
    * step 3 the frondend can send the same json or slightly modify it to make a bulk create
* import/export as backup option