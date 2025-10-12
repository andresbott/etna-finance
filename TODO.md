# TODO

## CSV import
* new dialog to upload CSV
  * select account
* new backend that processes CSV and generates  entries list
  * apply text to category 
  * search for duplicates
  * apply / return errors
  
* new View that allows to define import rules per account
  * rules are like, N column => field mapping
  * column name => field name
  * list of description regex to category mapping 
  * mandatory fields
    * amount
    * date
    * description
## Currency
* expose available currencies in the config file, use fallback as default

## CI
add goreleaser to build and create the releases

## Backup restore
* backup/restore
  * use a background scheduled anacron job to generate backups
  * backup is a json dump of the tables + dump version
  * to import we keep importers in place for every version, like migration
  * the startup process should check for DB changes and do a backup/import if there is a change

# Brain dump
* maintenance
  * add maintenance job that will check if there are entries assigned to a non-existing category and assigng them to the root category
    * add a log message to the job
  * make sure that you can't delete a category that has entries

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
      * when you sell a stock 2 operations need to happen: 
        * 1 a transfer back in the amount of stoicks slold at buy price
        * 2 a win/loss income/expense for the difference between buy and sell
    
  * Stock Value (extra Table) > auto import in the future
    * add an extra entry that updates the value of a stock
      * this is a time progressing entry to also see the evolution
* account listing: sort accounts by amount of movemnts, putting the one with most on the top
  * alternative add sort
* stock details
  * Tikker 
  * Peso % en portfolio
  * Nº acciones,
  * valor de compra 
  * total invertido en $$$ 
  * Cotización => valor actual
  * Diferencia => dif compra / actual por accion
  * Diferencia total => dif compra / actual por todo invertido  en $$
  * Diferencia en % => dif compra / actual por todo invertido  en %
  * stop => stop value to sell
  

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