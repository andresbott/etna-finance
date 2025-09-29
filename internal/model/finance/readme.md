

# big refactoring

separate entries in:
* income / expenses / transfers / stock operation
* values are stored as uint with 2 digits displaced instead of floats
* categories are stored in the same tree, and at high level we check uppon creation or update if it's a valid parent
* transfer generates 2 entries ?
  * one income one expense between accounts


https://chatgpt.com/share/68d98555-3194-8005-842d-defdb65fa1b8

# methods
* Create entry
* Get entry
* Delete Entry
* Update Entry
* List entries 
  * Joint of all types
* sum by  category
  * lists incomes and expenses by categories, individual or aggregated
* account balance
  * needs to add and substract all operations to calculate the balance


I have account providers e.g. robinhood for stocks or bank of america for different bank accounts
every account has a currency 

create incomes assigned to an account and a category e.g salary
create expenses assigned to an account and a category
create a transfer e.g. move some amount from one account to anothers
stock operations where i buy and sell stocks, gains/loses are captured

categories are only for gains and expenses
categories for income and expenses are separated



# Problems to solve
* get the current status of an account
* get the incomes/expenses for a certain period of time
* get the current value of stoks account
  * 