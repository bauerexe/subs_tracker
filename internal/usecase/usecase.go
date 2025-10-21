package usecase

import (
	"context"
	"errors"
	"github.com/go-openapi/strfmt"
	"time"

	"subs_tracker/internal/entity"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -destination=usecase_mock.go -package=usecase subs_tracker/internal/usecase SubscriptionRepository

var (
	ErrInvalidPeriod        = errors.New("invalid period")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrInvalidSubscription  = errors.New("invalid subscription")
	ErrInvalidID            = errors.New("invalid id")
	ErrInvalidPagination    = errors.New("invalid pagination")
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

// Period — period od subscription
type Period struct {
	// From - start time of the period (inclusive)
	From time.Time
	// To - end time of the period (inclusive)
	To time.Time
}

// SubFilter — common filter for queries/aggregations
type SubFilter struct {
	// UserID - ID of the user to filter by
	UserID strfmt.UUID
	// ServiceName - service name to filter by
	ServiceName *string
	// Period - period to filter by
	Period *Period
	// Limit - maximum number of records in the response
	Limit int
	// Offset - result set offset
	Offset int
}

// SubscriptionRepository — CRUD for subscriptions plus queries/aggregations
type SubscriptionRepository interface {
	// SaveSub - save a subscription
	SaveSub(ctx context.Context, s *entity.Subscription) (*entity.Subscription, error)
	// UpdateSub -  update subscription data
	UpdateSub(ctx context.Context, s *entity.Subscription) error
	// DeleteSub - delete a subscription
	DeleteSub(ctx context.Context, id int64) error
	// GetSubByID -  get a subscription by ID
	GetSubByID(ctx context.Context, id int64) (*entity.Subscription, error)
	// ListSubsByFilter - list subscriptions using SubFilter
	ListSubsByFilter(ctx context.Context, f SubFilter) ([]*entity.Subscription, error)
	// CostSubsByFilter -  get total subscription cost using SubFilter
	CostSubsByFilter(ctx context.Context, f SubFilter) (int64, error)
}
