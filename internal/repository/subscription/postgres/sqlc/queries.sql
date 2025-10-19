-- name: CreateSubscription :one
INSERT INTO subscriptions (user_id, service_name, cost, start_date, end_date)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(service_name),
    sqlc.arg(cost),
    sqlc.arg(start_date),
    sqlc.narg(end_date)
)
RETURNING id, user_id, service_name, cost, start_date, end_date;

-- name: UpdateSubscription :execrows
UPDATE subscriptions
SET
    user_id = sqlc.arg(user_id),
    service_name = sqlc.arg(service_name),
    cost = sqlc.arg(cost),
    start_date = sqlc.arg(start_date),
    end_date = sqlc.narg(end_date)
WHERE id = sqlc.arg(id);

-- name: DeleteSubscription :execrows
DELETE FROM subscriptions
WHERE id = sqlc.arg(id);

-- name: GetSubscription :one
SELECT id, user_id, service_name, cost, start_date, end_date
FROM subscriptions
WHERE id = sqlc.arg(id);

-- name: ListSubscriptions :many
SELECT id, user_id, service_name, cost, start_date, end_date
FROM subscriptions
WHERE
    (sqlc.narg(user_id) IS NULL OR user_id = sqlc.narg(user_id))
    AND (sqlc.narg(service_name) IS NULL OR service_name = sqlc.narg(service_name))
    AND (
        sqlc.narg(period_from) IS NULL
        OR (
            (end_date IS NULL OR end_date >= sqlc.narg(period_from))
            AND (sqlc.narg(period_to) IS NULL OR start_date <= sqlc.narg(period_to))
        )
    )
ORDER BY start_date, service_name, id
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: SumSubscriptionCost :one
WITH params AS (
    SELECT
        sqlc.arg(period_from) AS from_date,
        sqlc.arg(period_to) AS to_date,
        sqlc.narg(user_id) AS user_id,
        sqlc.narg(service_name) AS service_name
),
filtered AS (
    SELECT s.*
    FROM subscriptions s
    CROSS JOIN params p
    WHERE s.start_date <= p.to_date
      AND (s.end_date IS NULL OR s.end_date >= p.from_date)
      AND (p.user_id IS NULL OR s.user_id = p.user_id)
      AND (p.service_name IS NULL OR s.service_name = p.service_name)
),
expanded AS (
    SELECT f.cost
    FROM filtered f
    CROSS JOIN params p
    CROSS JOIN LATERAL generate_series(
        GREATEST(f.start_date, p.from_date),
        LEAST(COALESCE(f.end_date, p.to_date), p.to_date),
        interval '1 month'
    ) AS month_start
)
SELECT COALESCE(SUM(cost), 0)::bigint AS total_cost
FROM expanded;
