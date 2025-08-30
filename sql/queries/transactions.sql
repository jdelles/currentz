-- name: CreateTransaction :exec
INSERT INTO transactions (date, amount, description, type)
VALUES ($1, $2, $3, $4);

-- name: GetAllTransactions :many
SELECT id, date, amount, description, type, created_at
FROM transactions
ORDER BY date ASC;

-- name: GetTransactionsByDateRange :many
SELECT id, date, amount, description, type, created_at
FROM transactions
WHERE date BETWEEN $1 AND $2
ORDER BY date ASC;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE id = $1;

-- name: GetTransactionByID :one
SELECT id, date, amount, description, type, created_at
FROM transactions
WHERE id = $1;

-- name: GetTransactionsByType :many
SELECT id, date, amount, description, type, created_at
FROM transactions
WHERE type = $1
ORDER BY date ASC;