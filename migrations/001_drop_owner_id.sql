-- Migration: drop owner_id column (remove multi-tenancy)
-- Run this AFTER deploying code that no longer reads/writes owner_id (tasks 1-2).
-- Requires SQLite 3.35.0+ for DROP COLUMN. Back up your database before running.

-- Accounting tables
ALTER TABLE db_account_providers DROP COLUMN owner_id;
ALTER TABLE db_accounts DROP COLUMN owner_id;
ALTER TABLE db_transactions DROP COLUMN owner_id;
ALTER TABLE db_entries DROP COLUMN owner_id;

-- Marketdata instruments
ALTER TABLE db_instruments DROP COLUMN owner_id;

-- If closure-tree (categories) uses a tenant/owner column in its table, add here after inspecting the schema:
-- ALTER TABLE <closure_tree_table> DROP COLUMN <tenant_column>;
