-- name: CreateRecurring :one
INSERT INTO recurring_transactions (
  description,
  type,
  amount,
  start_date,
  "interval",
  day_of_week,
  day_of_month,
  end_date,
  active
) VALUES (
  sqlc.arg(description),
  sqlc.arg(type),
  sqlc.arg(amount),
  sqlc.arg(start_date),
  sqlc.arg(interval),
  sqlc.arg(day_of_week),
  sqlc.arg(day_of_month),
  sqlc.arg(end_date),
  sqlc.arg(active)
)
RETURNING *;

-- name: GetRecurringByID :one
SELECT * FROM recurring_transactions WHERE id = sqlc.arg(id);

-- name: ListRecurring :many
SELECT * FROM recurring_transactions ORDER BY id;

-- name: DeleteRecurring :exec
DELETE FROM recurring_transactions WHERE id = sqlc.arg(id);

-- name: SetRecurringActive :exec
UPDATE recurring_transactions
SET active = sqlc.arg(active)
WHERE id = sqlc.arg(id);

-- name: UpdateRecurring :one
UPDATE recurring_transactions
SET
  description  = sqlc.arg(description),
  type         = sqlc.arg(type),
  amount       = sqlc.arg(amount),
  start_date   = sqlc.arg(start_date),
  "interval"   = sqlc.arg(interval),
  day_of_week  = sqlc.arg(day_of_week),
  day_of_month = sqlc.arg(day_of_month),
  end_date     = sqlc.arg(end_date),
  active       = sqlc.arg(active)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListActiveRecurring :many
SELECT * FROM recurring_transactions WHERE active = TRUE;
