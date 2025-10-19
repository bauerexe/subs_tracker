package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"subs_tracker/internal/entity"
	"subs_tracker/internal/repository/subscription/postgres/sqlc"
	"subs_tracker/internal/usecase"
	"time"
)

type SubRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

const defaultListLimit = 50

func NewSubRepository(pool *pgxpool.Pool) *SubRepository {
	return &SubRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

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
		Limit:  int32(limit),
		Offset: int32(offset),
	}
	if f.UserID.String() != "" {
		uid := f.UserID.String()
		params.UserID = &uid
	}
	if f.ServiceName != nil {
		params.ServiceName = f.ServiceName
	}
	if f.Period != nil {
		if !f.Period.From.IsZero() {
			from := f.Period.From
			params.PeriodFrom = &from
		}
		if !f.Period.To.IsZero() {
			to := f.Period.To
			params.PeriodTo = &to
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

func (r *SubRepository) CostSubsByFilter(ctx context.Context, f usecase.SubFilter) (int64, error) {
	if f.Period == nil || f.Period.From.IsZero() || f.Period.To.IsZero() {
		return 0, fmt.Errorf("cost subs by filter: %w", usecase.ErrInvalidPeriod)
	}
	params := sqlc.SumSubscriptionCostParams{
		PeriodFrom: f.Period.From,
		PeriodTo:   f.Period.To,
	}
	if f.UserID.String() != "" {
		uid := f.UserID.String()
		params.UserID = &uid
	}
	if f.ServiceName != nil {
		params.ServiceName = f.ServiceName
	}
	total, err := r.queries.SumSubscriptionCost(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("cost subs by filter: %w", err)
	}
	return total, nil
}

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
