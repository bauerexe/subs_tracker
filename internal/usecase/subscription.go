package usecase

import (
	"context"
	"fmt"
	"strings"
	"subs_tracker/internal/entity"
	"time"
)

// Subscription coordinates subscription use cases via the repository
type Subscription struct {
	Sr SubscriptionRepository
}

// NewSubscription creates a use case service with the given repository
func NewSubscription(sr SubscriptionRepository) *Subscription {
	return &Subscription{
		Sr: sr,
	}
}

// RegisterSub validates/normalizes and saves a new subscription
func (s *Subscription) RegisterSub(ctx context.Context, sub *entity.Subscription) (*entity.Subscription, error) {
	if err := s.validateAndNormalize(sub); err != nil {
		return nil, err
	}
	created, err := s.Sr.SaveSub(ctx, sub)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateSub validates/normalizes and updates an existing subscription by ID, returning the fresh copy
func (s *Subscription) UpdateSub(ctx context.Context, sub *entity.Subscription) (*entity.Subscription, error) {
	if sub == nil || sub.ID <= 0 {
		return nil, ErrInvalidID
	}
	if err := s.validateAndNormalize(sub); err != nil {
		return nil, err
	}
	if err := s.Sr.UpdateSub(ctx, sub); err != nil {
		return nil, err
	}

	return s.Sr.GetSubByID(ctx, sub.ID)
}

// DeleteSub removes a subscription by ID and returns the previously stored record
func (s *Subscription) DeleteSub(ctx context.Context, ID int64) (*entity.Subscription, error) {
	if ID <= 0 {
		return nil, ErrInvalidID
	}

	existing, err := s.Sr.GetSubByID(ctx, ID)
	if err != nil {
		return nil, err
	}
	if err := s.Sr.DeleteSub(ctx, ID); err != nil {
		return nil, err
	}
	return existing, nil
}

// GetSubByID fetches a subscription by its ID
func (s *Subscription) GetSubByID(ctx context.Context, ID int64) (*entity.Subscription, error) {
	if ID <= 0 {
		return nil, ErrInvalidID
	}
	return s.Sr.GetSubByID(ctx, ID)
}

// ListSubsByFilter normalizes the filter and returns matching subscriptions
func (s *Subscription) ListSubsByFilter(ctx context.Context, filter SubFilter) ([]*entity.Subscription, error) {
	nf, err := normalizeFilter(filter)
	if err != nil {
		return nil, err
	}
	return s.Sr.ListSubsByFilter(ctx, nf)
}

// CostSubsByFilter normalizes the filter and returns the total cost for matching subscriptions
func (s *Subscription) CostSubsByFilter(ctx context.Context, filter SubFilter) (int64, error) {
	nf, err := normalizeFilter(filter)
	if err != nil {
		return 0, err
	}
	return s.Sr.CostSubsByFilter(ctx, nf)
}

// monthStart truncates a time to the first day of its month in UTC
func monthStart(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// validateAndNormalize enforces business rules and aligns dates to month starts
func (s *Subscription) validateAndNormalize(sub *entity.Subscription) error {
	if sub == nil {
		return fmt.Errorf("%w: nil", ErrInvalidSubscription)
	}
	sub.ServiceName = strings.TrimSpace(sub.ServiceName)
	if sub.ServiceName == "" {
		return fmt.Errorf("%w: empty service_name", ErrInvalidSubscription)
	}
	if sub.Cost <= 0 {
		return fmt.Errorf("%w: cost must be > 0", ErrInvalidSubscription)
	}
	if sub.UserID.String() == "" {
		return fmt.Errorf("%w: empty user_id", ErrInvalidSubscription)
	}
	if sub.DateFrom.IsZero() {
		return fmt.Errorf("%w: empty start_date", ErrInvalidSubscription)
	}

	sub.DateFrom = monthStart(sub.DateFrom)
	if sub.DateTo != nil && !sub.DateTo.IsZero() {
		d := monthStart(*sub.DateTo)
		sub.DateTo = &d
		if d.Before(sub.DateFrom) {
			return fmt.Errorf("%w: end_date before start_date", ErrInvalidPeriod)
		}
	}
	return nil
}

// normalizeFilter validates period and pagination
func normalizeFilter(f SubFilter) (SubFilter, error) {
	if f.Period != nil {
		from := monthStart(f.Period.From)
		to := monthStart(f.Period.To)
		if from.IsZero() {
			return f, fmt.Errorf("%w: empty period bound", ErrInvalidPeriod)
		}
		if !to.IsZero() {
			if to.Before(from) {
				return f, fmt.Errorf("%w: to < from", ErrInvalidPeriod)
			}
		}

		ff := f
		ff.Period = &Period{From: from, To: to}
		f = ff
	}

	if f.Offset < 0 {
		return f, fmt.Errorf("%w: offset must be >= 0", ErrInvalidPagination)
	}
	limit := f.Limit
	switch {
	case limit <= 0:
		limit = defaultListLimit
	case limit > maxListLimit:
		limit = maxListLimit
	}

	ff := f
	ff.Limit = limit
	ff.Offset = f.Offset
	return ff, nil
}
