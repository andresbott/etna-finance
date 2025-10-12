# Timeseries

This is a library that allows to store float values as time series into a Gorm compatible database.

And it contains methods 

The library allows to define a different policies that define how downsampling and retention is handled.

# Status

The implementation is not complete and many planed features are not yet there.


## TODO

* downsampling
  * aggregate records for a given policy to the specified precision value
* retention:
  * auto clean records for a given policy for the specified retention period
* Retrieval
  * the retrieval functions should take in considered the different policies and return the most accurate
  value for a given series
  * different retrieval methods should be implemented to allow e.g. linear progression in time between 2 values
  * function to retrieve a time series, not only a single value.

