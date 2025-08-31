-- +goose Up
CREATE TYPE recurrence_interval AS ENUM ('weekly', 'biweekly', 'monthly', 'yearly');

CREATE TABLE recurring_transactions (
  id              SERIAL PRIMARY KEY,
  description     TEXT NOT NULL,
  type            TEXT NOT NULL CHECK (type IN ('income','expense')),
  amount          NUMERIC(12,2) NOT NULL,               -- positive number; expense will be negated in display
  start_date      DATE NOT NULL,
  interval        recurrence_interval NOT NULL,
  day_of_week     INT CHECK (day_of_week BETWEEN 0 AND 6), -- 0=Sunday ... 6=Saturday (for weekly/biweekly)
  day_of_month    INT CHECK (day_of_month BETWEEN 1 AND 31),-- for monthly/yearly
  end_date        DATE,                                  -- optional
  active          BOOLEAN NOT NULL DEFAULT TRUE
);

-- Note:
-- weekly/biweekly use start_date as the phase anchor. Optional day_of_week lets you pin a weekday.
-- monthly/yearly use start_date and/or day_of_month. If day_of_month is NULL, we'll use start_date's day.

-- +goose Down
DROP TABLE IF EXISTS recurring_transactions;
DROP TYPE IF EXISTS recurrence_interval;
