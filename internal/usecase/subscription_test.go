package usecase

import (
	"context"
	"errors"
	"github.com/go-openapi/strfmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"subs_tracker/internal/entity"
)

func Test_subscription_RegisterSub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("err, invalid period", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().SaveSub(gomock.Any(), gomock.Any()).Times(0)

		uc := NewSubscription(repo)

		start := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, -1, 0)
		_, err := uc.RegisterSub(ctx, &entity.Subscription{
			ID:          0,
			UserID:      strfmt.UUID(uuid.New().String()),
			ServiceName: "Skillbox",
			Cost:        10000,
			DateFrom:    start,
			DateTo:      &end,
		})
		assert.ErrorIs(t, err, ErrInvalidPeriod)
	})

	t.Run("err, repo returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		expected := errors.New("save error")
		repo.EXPECT().SaveSub(ctx, gomock.Any()).Times(1).Return(nil, expected)

		uc := NewSubscription(repo)

		start := time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)
		_, err := uc.RegisterSub(ctx, &entity.Subscription{
			ID:          0,
			UserID:      strfmt.UUID(uuid.New().String()),
			ServiceName: "Netflix",
			Cost:        499,
			DateFrom:    start,
		})
		assert.ErrorIs(t, err, expected)
	})

	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().SaveSub(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, s *entity.Subscription) (*entity.Subscription, error) {
				assert.Equal(t, 1, s.DateFrom.Day())
				if s.DateTo != nil {
					assert.Equal(t, 1, s.DateTo.Day())
				}
				s.ID = 42
				return s, nil
			}).Times(1)

		uc := NewSubscription(repo)

		start := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
		got, err := uc.RegisterSub(ctx, &entity.Subscription{
			ID:          0,
			UserID:      strfmt.UUID(uuid.New().String()),
			ServiceName: "YouTube",
			Cost:        199,
			DateFrom:    start,
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(42), got.ID)
		assert.Equal(t, 1, got.DateFrom.Day())
	})
}

func Test_subscription_UpdateSub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("err, invalid period", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().UpdateSub(gomock.Any(), gomock.Any()).Times(0)

		uc := NewSubscription(repo)

		start := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, -1, 0)

		_, err := uc.UpdateSub(ctx, &entity.Subscription{
			ID:          10,
			UserID:      strfmt.UUID(uuid.New().String()),
			ServiceName: "A",
			Cost:        1,
			DateFrom:    start,
			DateTo:      &end,
		})
		assert.ErrorIs(t, err, ErrInvalidPeriod)
	})

	t.Run("ok, update then get", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)

		start := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
		id := int64(77)
		user := uuid.New()

		repo.EXPECT().UpdateSub(ctx, gomock.Any()).Times(1).Return(nil)
		repo.EXPECT().GetSubByID(ctx, id).Times(1).Return(&entity.Subscription{
			ID:          id,
			UserID:      strfmt.UUID(user.String()),
			ServiceName: "Pro",
			Cost:        500,
			DateFrom:    start,
		}, nil)

		uc := NewSubscription(repo)

		got, err := uc.UpdateSub(ctx, &entity.Subscription{
			ID:          id,
			UserID:      strfmt.UUID(user.String()),
			ServiceName: "Pro",
			Cost:        500,
			DateFrom:    start.AddDate(0, 0, 15),
		})
		assert.NoError(t, err)
		assert.Equal(t, id, got.ID)
		assert.Equal(t, 500, int(got.Cost))
		assert.Equal(t, 1, got.DateFrom.Day())
	})
}

func Test_subscription_DeleteSub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("err, not found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().GetSubByID(ctx, int64(123)).Times(1).Return(nil, ErrSubscriptionNotFound)

		uc := NewSubscription(repo)

		_, err := uc.DeleteSub(ctx, 123)
		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
	})

	t.Run("ok, return deleted entity", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		id := int64(5)
		user := uuid.New()
		existing := &entity.Subscription{
			ID:          id,
			UserID:      strfmt.UUID(user.String()),
			ServiceName: "Skillbox",
			Cost:        10000,
			DateFrom:    time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		}

		repo.EXPECT().GetSubByID(ctx, id).Times(1).Return(existing, nil)
		repo.EXPECT().DeleteSub(ctx, id).Times(1).Return(nil)

		uc := NewSubscription(repo)

		got, err := uc.DeleteSub(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, existing, got)
	})
}

func Test_subscription_GetSubByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("repo error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().GetSubByID(ctx, int64(1)).Times(1).Return(nil, errors.New("boom"))

		uc := NewSubscription(repo)

		_, err := uc.GetSubByID(ctx, 1)
		assert.Error(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		user := uuid.New()
		repo.EXPECT().GetSubByID(ctx, int64(2)).Times(1).Return(&entity.Subscription{
			ID:          2,
			UserID:      strfmt.UUID(user.String()),
			ServiceName: "Netflix",
			Cost:        499,
			DateFrom:    time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		}, nil)

		uc := NewSubscription(repo)

		got, err := uc.GetSubByID(ctx, 2)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), got.ID)
	})
}

func Test_subscription_ListSubsByFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("repo error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().ListSubsByFilter(ctx, gomock.Any()).Times(1).Return(nil, errors.New("oops"))

		uc := NewSubscription(repo)

		_, err := uc.ListSubsByFilter(ctx, SubFilter{})
		assert.Error(t, err)
	})

	t.Run("ok list", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		list := []*entity.Subscription{
			{ID: 1, UserID: strfmt.UUID(uuid.New().String()), ServiceName: "A", Cost: 10, DateFrom: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)},
			{ID: 2, UserID: strfmt.UUID(uuid.New().String()), ServiceName: "B", Cost: 20, DateFrom: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)},
		}
		repo.EXPECT().ListSubsByFilter(ctx, gomock.Any()).Times(1).Return(list, nil)

		uc := NewSubscription(repo)

		got, err := uc.ListSubsByFilter(ctx, SubFilter{})
		assert.NoError(t, err)
		assert.Len(t, got, 2)
	})
}

func Test_subscription_CostSubsByFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("repo error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().CostSubsByFilter(ctx, gomock.Any()).Times(1).Return(int64(0), errors.New("sum err"))

		uc := NewSubscription(repo)

		period := &Period{From: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)}

		_, err := uc.CostSubsByFilter(ctx, SubFilter{Period: period})
		assert.Error(t, err)
	})

	t.Run("ok sum", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().CostSubsByFilter(ctx, gomock.Any()).Times(1).Return(int64(12345), nil)

		uc := NewSubscription(repo)

		period := &Period{From: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)}

		sum, err := uc.CostSubsByFilter(ctx, SubFilter{Period: period})
		assert.NoError(t, err)
		assert.Equal(t, int64(12345), sum)
	})
}
