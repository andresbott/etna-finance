# Brain dump
* Accounts
  * make sub-accounts:
    * bank: post finance
      * main account CHF
      * saving EUR
    * stocks IBKR
      * savings USD
      * VOO
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
  * define csv structure column name to field mapping,
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