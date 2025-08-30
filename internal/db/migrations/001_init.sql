-- +goose Up
-- Enable UUID generator
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Updated-at trigger function (reused by all tables)
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  IF row(NEW.*) IS DISTINCT FROM row(OLD.*) THEN
    NEW.updated_at = now();
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd

-- Domain enums
CREATE TYPE account_type AS ENUM ('checking','savings','credit','loan','investment');

-- USERS
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  full_name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'America/New_York',
  base_currency CHAR(3) NOT NULL DEFAULT 'USD',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER set_timestamp_users
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- INSTITUTIONS (optional metadata)
CREATE TABLE institutions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  logo_url TEXT,
  country CHAR(2),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER set_timestamp_institutions
  BEFORE UPDATE ON institutions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ACCOUNTS
CREATE TABLE accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  institution_id UUID REFERENCES institutions(id),
  name TEXT NOT NULL,
  type account_type NOT NULL,
  currency CHAR(3) NOT NULL DEFAULT 'USD',
  is_liability BOOLEAN NOT NULL DEFAULT FALSE,
  current_balance_cents BIGINT NOT NULL DEFAULT 0,
  as_of DATE NOT NULL DEFAULT CURRENT_DATE,
  display_order INT,
  last_four TEXT,
  external_ref TEXT,
  is_closed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uniq_account_name_open
  ON accounts(user_id, name)
  WHERE is_closed = false;
CREATE TRIGGER set_timestamp_accounts
  BEFORE UPDATE ON accounts
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- CATEGORIES
CREATE TABLE categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  parent_id UUID REFERENCES categories(id),
  is_income BOOLEAN NOT NULL DEFAULT FALSE,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uniq_category_name_active
  ON categories(user_id, name)
  WHERE archived_at IS NULL;
CREATE TRIGGER set_timestamp_categories
  BEFORE UPDATE ON categories
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- TRANSACTIONS
CREATE TABLE transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  account_id UUID NOT NULL REFERENCES accounts(id),
  posted_at TIMESTAMPTZ NOT NULL,
  authorized_at TIMESTAMPTZ,
  amount_cents BIGINT NOT NULL,               -- negative = outflow
  currency CHAR(3) NOT NULL DEFAULT 'USD',
  fx_amount_cents BIGINT,
  fx_currency CHAR(3),
  fitid TEXT,                                  -- bank's unique id (for dedupe)
  payee TEXT,
  normalized_payee TEXT,
  memo TEXT,
  raw JSONB,
  category_id UUID REFERENCES categories(id),
  import_batch_id UUID,
  statement_id UUID,
  is_transfer BOOLEAN NOT NULL DEFAULT FALSE,
  is_pending  BOOLEAN NOT NULL DEFAULT FALSE,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (amount_cents <> 0)
);
CREATE UNIQUE INDEX uniq_tx_fitid
  ON transactions(user_id, account_id, fitid)
  WHERE fitid IS NOT NULL;
CREATE INDEX idx_tx_user_account_posted
  ON transactions(user_id, account_id, posted_at DESC);
CREATE INDEX idx_tx_user_normpayee
  ON transactions(user_id, normalized_payee);
CREATE TRIGGER set_timestamp_transactions
  BEFORE UPDATE ON transactions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- TRANSACTION SPLITS
CREATE TABLE transaction_splits (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  category_id UUID REFERENCES categories(id),
  amount_cents BIGINT NOT NULL,
  memo TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_splits_tx ON transaction_splits(transaction_id);
CREATE TRIGGER set_timestamp_transaction_splits
  BEFORE UPDATE ON transaction_splits
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- TRANSFERS (link legs across accounts)
CREATE TABLE transfers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  from_tx_id UUID UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
  to_tx_id   UUID UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
  amount_cents BIGINT NOT NULL,
  matched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uniq_transfer_pair ON transfers(from_tx_id, to_tx_id);
CREATE TRIGGER set_timestamp_transfers
  BEFORE UPDATE ON transfers
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- RECURRING SERIES (for projections)
CREATE TABLE recurring_series (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  amount_cents BIGINT NOT NULL,
  frequency TEXT NOT NULL,        -- 'weekly','biweekly','monthly','custom'
  start_date DATE NOT NULL,
  end_date DATE,
  category_id UUID REFERENCES categories(id),
  account_id UUID REFERENCES accounts(id),
  payee TEXT,
  anchor_day SMALLINT,
  variability_pct SMALLINT NOT NULL DEFAULT 0,
  timezone TEXT NOT NULL DEFAULT 'America/New_York',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_recurring_user ON recurring_series(user_id);
CREATE TRIGGER set_timestamp_recurring_series
  BEFORE UPDATE ON recurring_series
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- PROJECTIONS (optional cache)
CREATE TABLE projections (
  user_id UUID NOT NULL REFERENCES users(id),
  account_id UUID REFERENCES accounts(id),
  date DATE NOT NULL,
  projected_balance_cents BIGINT NOT NULL,
  method TEXT NOT NULL DEFAULT 'deterministic',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, account_id, date)
);
CREATE TRIGGER set_timestamp_projections
  BEFORE UPDATE ON projections
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- RULES (auto-categorization / transfers / rename)
CREATE TABLE rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  pattern TEXT NOT NULL,          -- regex on payee/memo
  field TEXT NOT NULL,            -- 'payee' | 'memo'
  action TEXT NOT NULL,           -- 'set_category' | 'mark_transfer' | 'rename_payee'
  category_id UUID REFERENCES categories(id),
  transfer_account_id UUID REFERENCES accounts(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_rules_user ON rules(user_id);
CREATE TRIGGER set_timestamp_rules
  BEFORE UPDATE ON rules
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- STATEMENTS (reconciliation)
CREATE TABLE statements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id UUID NOT NULL REFERENCES accounts(id),
  period_start DATE NOT NULL,
  period_end DATE NOT NULL,
  closing_balance_cents BIGINT NOT NULL,
  statement_ref TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER set_timestamp_statements
  BEFORE UPDATE ON statements
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ACCOUNT BALANCES (daily snapshots / imports)
CREATE TABLE account_balances (
  account_id UUID NOT NULL REFERENCES accounts(id),
  as_of DATE NOT NULL,
  balance_cents BIGINT NOT NULL,
  source TEXT NOT NULL,           -- 'import' | 'manual' | 'statement'
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (account_id, as_of)
);
CREATE TRIGGER set_timestamp_account_balances
  BEFORE UPDATE ON account_balances
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- IMPORT BATCHES (audit)
CREATE TABLE import_batches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  source TEXT NOT NULL,           -- 'csv:chase' | 'ofx' | 'plaid'
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  raw_manifest JSONB
);
CREATE INDEX idx_import_batches_user ON import_batches(user_id);
CREATE TRIGGER set_timestamp_import_batches
  BEFORE UPDATE ON import_batches
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- TAGS (optional)
CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uniq_tag_name ON tags(user_id, name);
CREATE TRIGGER set_timestamp_tags
  BEFORE UPDATE ON tags
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE transaction_tags (
  transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (transaction_id, tag_id)
);
CREATE TRIGGER set_timestamp_transaction_tags
  BEFORE UPDATE ON transaction_tags
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS transaction_tags;
DROP TABLE IF EXISTS tags;
DROP INDEX IF EXISTS idx_import_batches_user;
DROP TABLE IF EXISTS import_batches;
DROP TABLE IF EXISTS account_balances;
DROP TABLE IF EXISTS statements;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS projections;
DROP INDEX IF EXISTS idx_recurring_user;
DROP TABLE IF EXISTS recurring_series;
DROP INDEX IF EXISTS uniq_transfer_pair;
DROP TABLE IF EXISTS transfers;
DROP INDEX IF EXISTS idx_splits_tx;
DROP TABLE IF EXISTS transaction_splits;
DROP INDEX IF EXISTS idx_tx_user_normpayee;
DROP INDEX IF EXISTS idx_tx_user_account_posted;
DROP INDEX IF EXISTS uniq_tx_fitid;
DROP TABLE IF EXISTS transactions;
DROP INDEX IF EXISTS uniq_category_name_active;
DROP TABLE IF EXISTS categories;
DROP INDEX IF EXISTS uniq_account_name_open;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS institutions;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS account_type;
-- +goose StatementBegin
DROP FUNCTION IF EXISTS set_updated_at();
-- +goose StatementEnd
