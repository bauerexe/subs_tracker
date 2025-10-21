package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"subs_tracker/internal/entity"
	"subs_tracker/internal/repository/subscription/postgres/sqlc"
	"subs_tracker/internal/usecase"
)

// SubRepository wraps a pgx pool and sqlc-generated Queries to persist subscriptions
type SubRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

const defaultListLimit = 50

// NewSubRepository creates a repository bound to the given pgx connection pool
func NewSubRepository(pool *pgxpool.Pool) *SubRepository {
	return &SubRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// SaveSub inserts a new subscription via sqlc and returns the created entity
func (r *SubRepository) SaveSub(ctx context.Context, sub *entity.Subscription) (*entity.Subscription, error) {
	if sub == nil {
		return nil, fmt.Errorf("save sub: %w", usecase.ErrInvalidSubscription)
	}

	params := sqlc.CreateSubscriptionParams{
		UserID:      sub.UserID.String(),
		ServiceName: sub.ServiceName,
		Cost:        sub.Cost,
		StartDate:   sub.DateFrom,
	}
	if sub.DateTo != nil {
		params.EndDate = sub.DateTo
	}

	out, err := r.queries.CreateSubscription(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("save sub: %w", err)
	}
	return toEntity(out), nil
}

// UpdateSub updates an existing subscription by ID and reports not-found if no rows were affected
func (r *SubRepository) UpdateSub(ctx context.Context, sub *entity.Subscription) error {
	if sub == nil {
		return fmt.Errorf("update sub: %w", usecase.ErrInvalidSubscription)
	}

	params := sqlc.UpdateSubscriptionParams{
		ID:          sub.ID,
		UserID:      sub.UserID.String(),
		ServiceName: sub.ServiceName,
		Cost:        sub.Cost,
		StartDate:   sub.DateFrom,
	}
	if sub.DateTo != nil {
		params.EndDate = sub.DateTo
	}

	rows, err := r.queries.UpdateSubscription(ctx, params)
	if err != nil {
		return fmt.Errorf("update sub: %w", err)
	}
	if rows == 0 {
		return usecase.ErrSubscriptionNotFound
	}
	return nil
}

// DeleteSub removes a subscription by ID and reports not-found if no rows were affected
func (r *SubRepository) DeleteSub(ctx context.Context, id int64) error {
	rows, err := r.queries.DeleteSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("delete sub: %w", err)
	}
	if rows == 0 {
		return usecase.ErrSubscriptionNotFound
	}
	return nil
}

// GetSubByID fetches a subscription by its ID, mapping pgx.ErrNoRows to a domain not-found error
func (r *SubRepository) GetSubByID(ctx context.Context, id int64) (*entity.Subscription, error) {
	sub, err := r.queries.GetSubscription(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, usecase.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get sub by id=%d: %w", id, err)
	}
	return toEntity(sub), nil
}

// ListSubsByFilter converts a SubFilter to sqlc params (handling nullable fields) and returns matching rows
func (r *SubRepository) ListSubsByFilter(ctx context.Context, f usecase.SubFilter) ([]*entity.Subscription, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	params := sqlc.ListSubscriptionsParams{
		PageLimit:   int32(limit),
		PageOffset:  int32(offset),
		UserID:      pgtype.UUID{Valid: false},
		ServiceName: pgtype.Text{Valid: false},
		PeriodFrom:  pgtype.Date{Valid: false},
		PeriodTo:    pgtype.Date{Valid: false},
	}
	if f.UserID.String() != "" {
		uid, err := toPgUUID(f.UserID.String())
		if err != nil {
			return nil, fmt.Errorf("list subs by filter: %w", err)
		}
		params.UserID = uid
	}
	if f.ServiceName != nil {
		params.ServiceName = pgtype.Text{
			String: *f.ServiceName,
			Valid:  true,
		}
	}
	if f.Period != nil {
		if !f.Period.From.IsZero() {
			params.PeriodFrom = pgtype.Date{
				Time:  f.Period.From,
				Valid: true,
			}
		}
		if !f.Period.To.IsZero() {
			params.PeriodTo = pgtype.Date{
				Time:  f.Period.To,
				Valid: true,
			}
		}
	}

	rows, err := r.queries.ListSubscriptions(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list subs by filter: %w", err)
	}
	out := make([]*entity.Subscription, 0, len(rows))
	for _, item := range rows {
		out = append(out, toEntity(item))
	}
	return out, nil
}

// CostSubsByFilter validates the period and computes the total monthly cost using the aggregate sqlc query
func (r *SubRepository) CostSubsByFilter(ctx context.Context, f usecase.SubFilter) (int64, error) {
	if f.Period == nil || f.Period.From.IsZero() || f.Period.To.IsZero() {
		return 0, fmt.Errorf("cost subs by filter: %w", usecase.ErrInvalidPeriod)
	}
	params := sqlc.SumSubscriptionCostParams{
		PeriodFrom: f.Period.From,
		PeriodTo:   &f.Period.To,
	}
	uid, err := toPgUUID(f.UserID.String())
	if err != nil {
		return 0, fmt.Errorf("cost subs by filter: %w", err)
	}
	params.UserID = uid
	if f.ServiceName != nil {
		params.ServiceName = pgtype.Text{
			String: *f.ServiceName,
			Valid:  true,
		}
	}
	total, err := r.queries.SumSubscriptionCost(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("cost subs by filter: %w", err)
	}
	return total, nil
}

// toEntity maps a sqlc row to the domain Subscription, handling a nullable end_date safely
func toEntity(s sqlc.Subscription) *entity.Subscription {
	var end *time.Time
	if s.EndDate != nil {
		t := *s.EndDate
		end = &t
	}
	return &entity.Subscription{
		ID:          s.ID,
		UserID:      strfmt.UUID(s.UserID),
		ServiceName: s.ServiceName,
		Cost:        s.Cost,
		DateFrom:    s.StartDate,
		DateTo:      end,
	}
}

// toPgUUID parses a string UUID into pgtype.UUID, returning an invalid value when the input is empty
func toPgUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if s == "" {
		return pgtype.UUID{Valid: false}, nil
	}
	return u, u.Scan(s)
}
