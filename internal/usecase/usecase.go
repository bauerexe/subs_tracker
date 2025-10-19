package usecase

import (
	"context"
	"errors"
	"github.com/go-openapi/strfmt"
	"time"

	"subs_tracker/internal/entity"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -destination=mock/usecase_mock.go -package=mock subs_tracker/internal/usecase SubscriptionRepository

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

// Period — период подписки
type Period struct {
	// From - время начала периода (включительно)
	From time.Time
	// To - время конца периода (включительно)
	To time.Time
}

// SubFilter — общий фильтр для выборок/агрегаций.
type SubFilter struct {
	// UserID - идентификатор пользователя который пропустит фильтр
	UserID strfmt.UUID
	// ServiceName - название сервиса который пропустит филтр
	ServiceName *string
	// Period - период который пропустит фильтр
	Period *Period
	// Limit - максимальное количество записей в ответе
	Limit int
	// Offset - смещение выборки
	Offset int
}

// SubscriptionRepository — CRUD по подпискам + выборки/агрегаты.
type SubscriptionRepository interface {
	// SaveSub - функция сохранения подписки
	SaveSub(ctx context.Context, s *entity.Subscription) (*entity.Subscription, error)
	// UpdateSub - функция обновления данных подпски
	UpdateSub(ctx context.Context, s *entity.Subscription) error
	// DeleteSub - функция удаления подписки
	DeleteSub(ctx context.Context, id int64) error
	// GetSubByID - функция получения подписок по идентификатору подписки
	GetSubByID(ctx context.Context, id int64) (*entity.Subscription, error)
	// ListSubsByFilter - функция получения подписок по SubFilter - фильтру
	ListSubsByFilter(ctx context.Context, f SubFilter) ([]*entity.Subscription, error)
	// CostSubsByFilter - функция получения суммы стоимостей подписок по  SubFilter - фильтру
	CostSubsByFilter(ctx context.Context, f SubFilter) (int64, error)
}
