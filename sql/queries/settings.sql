-- name: GetSetting :one
SELECT value FROM settings WHERE key = $1;

-- name: UpdateSetting :exec
INSERT INTO settings (key, value, updated_at)
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (key)
DO UPDATE SET value = $2, updated_at = CURRENT_TIMESTAMP;

-- name: DeleteSetting :exec
DELETE FROM settings WHERE key = $1;

-- name: GetAllSettings :many
SELECT key, value, updated_at FROM settings;