package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"subs_tracker/internal/entity"
	"subs_tracker/internal/usecase"
	"time"
)

type SubRepository struct {
	pool *pgxpool.Pool
}

func NewSubRepository(pool *pgxpool.Pool) *SubRepository {
	return &SubRepository{
		pool: pool,
	}
}

func (r *SubRepository) SaveSub(ctx context.Context, sub *entity.Subscription) (*entity.Subscription, error) {
	var out entity.Subscription

	if sub.DateTo != nil {
		const q = `
          INSERT INTO subscriptions (user_id, service_name, cost, start_date, end_date)
          VALUES ($1, $2, $3, $4, $5)
          RETURNING id, user_id, service_name, cost, start_date, end_date;
        `
		row := r.pool.QueryRow(ctx, q, sub.UserID, sub.ServiceName, sub.Cost, sub.DateFrom, *sub.DateTo)
		if err := row.Scan(&out.ID, &out.UserID, &out.ServiceName, &out.Cost, &out.DateFrom, &out.DateTo); err != nil {
			return nil, fmt.Errorf("save sub: scan: %w", err)
		}
		return &out, nil
	}

	const q = `
      INSERT INTO subscriptions (user_id, service_name, cost, start_date)
      VALUES ($1, $2, $3, $4)
      RETURNING id, user_id, service_name, cost, start_date;
    `
	row := r.pool.QueryRow(ctx, q, sub.UserID, sub.ServiceName, sub.Cost, sub.DateFrom)
	if err := row.Scan(&out.ID, &out.UserID, &out.ServiceName, &out.Cost, &out.DateFrom); err != nil {
		return nil, fmt.Errorf("save sub: scan: %w", err)
	}
	return &out, nil
}

func (r *SubRepository) UpdateSub(ctx context.Context, sub *entity.Subscription) error {
	var (
		sb   strings.Builder
		args []any
		i    = 1
	)

	sb.WriteString("UPDATE subscriptions")

	sets := make([]string, 0)

	if sub.ServiceName != "" {
		sets = append(sets, fmt.Sprintf("service_name=$%d", i))
		args = append(args, sub.ServiceName)
		i++
	}

	if sub.Cost != 0 {
		sets = append(sets, fmt.Sprintf("cost=$%d", i))
		args = append(args, sub.Cost)
		i++
	}

	if !sub.DateFrom.IsZero() {
		sets = append(sets, fmt.Sprintf("start_date=$%d", i))
		args = append(args, sub.DateFrom)
		i++
	}
	if sub.DateTo != nil {
		if !sub.DateTo.IsZero() {
			sets = append(sets, fmt.Sprintf("end_date=$%d", i))
			args = append(args, *sub.DateTo)
			i++
		} else {
			sets = append(sets, "end_date=NULL")
		}
	}

	if len(sets) == 0 {
		return fmt.Errorf("update sub: no fields to update")
	}

	sb.WriteString("\nSET ")
	sb.WriteString(strings.Join(sets, ", "))

	sb.WriteString(fmt.Sprintf("\nWHERE id=$%d", i))
	args = append(args, sub.ID)

	ct, err := r.pool.Exec(ctx, sb.String(), args...)
	if err != nil {
		return fmt.Errorf("update sub: exec: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return usecase.ErrSubscriptionNotFound
	}

	return nil
}

func (r *SubRepository) DeleteSub(ctx context.Context, id int64) error {
	const deleteSub = `
	DELETE FROM subscriptions
	WHERE id=$1
	`
	ct, err := r.pool.Exec(ctx, deleteSub, id)

	if err != nil {
		return fmt.Errorf("delete sub: exec: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return usecase.ErrSubscriptionNotFound
	}

	return nil
}

func (r *SubRepository) GetSubByID(ctx context.Context, id int64) (*entity.Subscription, error) {
	const q = `
	SELECT id, user_id, service_name, cost, start_date, end_date
	FROM subscriptions
	WHERE id = $1;
	`
	var s entity.Subscription
	row := r.pool.QueryRow(ctx, q, id)

	if err := row.Scan(&s.ID, &s.UserID, &s.ServiceName, &s.Cost, &s.DateFrom, &s.DateTo); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, usecase.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get sub by id=%d: scan: %w", id, err)
	}
	return &s, nil
}

func (r *SubRepository) ListSubsByFilter(ctx context.Context, f usecase.SubFilter) ([]*entity.Subscription, error) {
	var (
		sb   strings.Builder
		args []any
		i    = 1
	)

	sb.WriteString(`
	SELECT id, user_id, service_name, cost, start_date, end_date
	FROM subscriptions
	WHERE true
	`)

	if f.Period != nil {
		hasFrom := !f.Period.From.IsZero()
		hasTo := !f.Period.To.IsZero()

		if hasFrom && hasTo {
			sb.WriteString(fmt.Sprintf("  AND start_date <= $%d AND (end_date IS NULL OR end_date >= $%d) ", i, i+1))
			args = append(args, f.Period.To, f.Period.From)
			i += 2
		} else if hasFrom && !hasTo {
			sb.WriteString(fmt.Sprintf("  AND (end_date IS NULL OR end_date >= $%d)  ", i))
			args = append(args, f.Period.From)
			i += 1
		}

	}
	if f.UserID != "" {
		sb.WriteString(fmt.Sprintf("  AND user_id = $%d ", i))
		args = append(args, f.UserID)
		i++
	}
	if f.ServiceName != nil {
		sb.WriteString(fmt.Sprintf("  AND service_name = $%d ", i))
		args = append(args, *f.ServiceName)
		i++
	}

	sb.WriteString("ORDER BY start_date, service_name, id")

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list subs by fiter: query: %w", err)
	}
	defer rows.Close()

	var out []*entity.Subscription
	for rows.Next() {
		var s entity.Subscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.ServiceName, &s.Cost, &s.DateFrom, &s.DateTo); err != nil {
			return nil, fmt.Errorf("list subs by fiter: scan: %w", err)
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *SubRepository) CostSubsByFilter(ctx context.Context, f usecase.SubFilter) (int64, error) {
	var (
		sb   strings.Builder
		args []any
		i    = 1
		cost = int64(0)
	)

	sb.WriteString(`
	SELECT cost, start_date, end_date
	FROM subscriptions
	WHERE true
	`)

	if f.Period != nil {
		sb.WriteString(fmt.Sprintf("  AND start_date <= $%d AND (end_date IS NULL OR end_date >= $%d) ", i, i+1))
		args = append(args, f.Period.To, f.Period.From)
		i += 2
	}
	var nilUUID strfmt.UUID

	if f.UserID != nilUUID {
		sb.WriteString(fmt.Sprintf("  AND user_id = $%d ", i))
		args = append(args, f.UserID)
		i++
	}
	if f.ServiceName != nil {
		sb.WriteString(fmt.Sprintf("  AND service_name = $%d ", i))
		args = append(args, *f.ServiceName)
		i++
	}

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("cost subs by filter: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rowCost int64
		var startDate, endDate *time.Time
		if err := rows.Scan(&rowCost, &startDate, &endDate); err != nil {
			return 0, fmt.Errorf("cost subs by filter: scan: %w", err)
		}

		if f.Period != nil {
			from := f.Period.From
			to := f.Period.To

			if startDate != nil && startDate.After(from) {
				from = *startDate
			}
			if endDate != nil && endDate.Before(to) {
				to = *endDate
			}

			if !from.After(to) {
				months := int64((to.Year()-from.Year())*12 + int(to.Month()-from.Month()) + 1)
				cost += rowCost * months
			}
		} else {
			cost += rowCost
		}
	}
	return cost, rows.Err()
}
