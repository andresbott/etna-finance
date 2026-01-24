
# flow

define rule for a category, regex (?) map to category

* upload vsc into memory / location
* parse entries
  * identify the profile to use to get the fields
  * get first and last entry
  * get all entries between first and last entry ( add extra days as buffer)
  * every entry check for same amount on the same date => duplicate
  * if enty does not exist
    * create a new temp entry
    * use profiles to map colum to type, need to guess/identify the profile to use
    * guess type
    * guess description
    * guess category
      * use rule for category
    * add to temp list
  * return temp transaction play
  * allow user to review and adapt mass import transaction
  * submit and apply
    * endpoint that allows to submit a list of transactions, instead of only one
