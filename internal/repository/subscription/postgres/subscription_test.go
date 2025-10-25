package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"subs_tracker/internal/entity"
	"subs_tracker/internal/usecase"
)

var pgContainer *postgres.PostgresContainer

func cleanup() {
	if pgContainer != nil {
		_ = pgContainer.Terminate(context.Background())
	}
}

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(1)
	}()

	c, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("subs_db"),
		postgres.WithUsername("subs_user"),
		postgres.WithPassword("subs_password"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "run container: %v\n", err)
		cleanup()
		os.Exit(1)
	}
	pgContainer = c

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "conn string: %v\n", err)
		cleanup()
		os.Exit(1)
	}

	migDir, err := filepath.Abs("../../../../migrations")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "migrations path: %v\n", err)
		cleanup()
		os.Exit(1)
	}
	if err := runMigrations(connStr, "file:///"+migDir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
		cleanup()
		os.Exit(1)
	}

	code := m.Run()

	cleanup()
	os.Exit(code)
}

func runMigrations(connStr, srcURL string) error {
	m, err := migrate.New(srcURL, connStr)
	if err != nil {
		return err
	}
	defer func(m *migrate.Migrate) {
		_, _ = m.Close()
	}(m)
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func TestSubRepository_SaveSub(t *testing.T) {
	ctx := context.Background()
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	_, _ = pool.Exec(ctx, `TRUNCATE TABLE subscriptions RESTART IDENTITY`)
	require.NoError(t, err)
	defer pool.Close()
	sr := NewSubRepository(pool)

	start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	uid := uuid.New()
	tcases := []struct {
		Name    string
		ForSave entity.Subscription
		Error   error
	}{
		{
			Name: "valid test SaveSub, without DateTo",
			ForSave: entity.Subscription{
				ID:          0,
				UserID:      strfmt.UUID(uid.String()),
				ServiceName: "Skillbox",
				Cost:        10_000,
				DateFrom:    start,
				DateTo:      nil,
			},
			Error: nil,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, err)
			})
			created, err := sr.SaveSub(ctx, &tc.ForSave)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
				if !errors.Is(err, tc.Error) {
					t.Fatalf("Invalid error while save: %s", err)
				}
				return
			}
			require.NoError(t, err)
			var got entity.Subscription
			row := pool.QueryRow(ctx, `
				SELECT id, user_id, service_name, cost, start_date, end_date
				FROM subscriptions
				WHERE id = $1`, created.ID)
			require.NoError(t, row.Scan(
				&got.ID, &got.UserID, &got.ServiceName, &got.Cost, &got.DateFrom, &got.DateTo,
			))
			b, _ := json.Marshal(got)
			fmt.Println(string(b))
			assert.Equal(t, got, *created)
		})
	}
}

func TestSubRepository_UpdateSub(t *testing.T) {
	ctx := context.Background()
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	_, _ = pool.Exec(ctx, `TRUNCATE TABLE subscriptions RESTART IDENTITY`)

	require.NoError(t, err)
	defer pool.Close()
	sr := NewSubRepository(pool)

	start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	afterStart := start.AddDate(0, 2, 0)
	afterStart2 := afterStart.AddDate(0, 3, 0)
	uid := uuid.New()
	tcases := []struct {
		Name      string
		ForSave   entity.Subscription
		ForUpdate entity.Subscription
		Error     error
	}{
		{
			Name: "valid test UpdateSub",
			ForSave: entity.Subscription{
				ID:          0,
				UserID:      strfmt.UUID(uid.String()),
				ServiceName: "Skillbox",
				Cost:        10_000,
				DateFrom:    start,
				DateTo:      nil,
			},
			ForUpdate: entity.Subscription{
				ID:          0,
				UserID:      strfmt.UUID(uid.String()),
				ServiceName: "SKILLBOX",
				Cost:        100_000,
				DateFrom:    afterStart,
				DateTo:      &afterStart2,
			},
			Error: nil,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, err)
			})
			created, err := sr.SaveSub(ctx, &tc.ForSave)
			if err != nil {
				t.Fatalf("Error while save: %s", err)
			}
			tc.ForUpdate.ID = created.ID
			err = sr.UpdateSub(ctx, &tc.ForUpdate)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
				if !errors.Is(err, tc.Error) {
					t.Fatalf("Invalid error while save: %s", err)
				}
				return
			}
			require.NoError(t, err)
			var got entity.Subscription
			row := pool.QueryRow(ctx, `
				SELECT id, user_id, service_name, cost, start_date, end_date
				FROM subscriptions
				WHERE id = $1`, created.ID)
			require.NoError(t, row.Scan(
				&got.ID, &got.UserID, &got.ServiceName, &got.Cost, &got.DateFrom, &got.DateTo,
			))
			b, _ := json.Marshal(got)
			fmt.Println(string(b))
			assert.Equal(t, got, tc.ForUpdate)
		})
	}
}

func TestSubRepository_DeleteSub(t *testing.T) {
	ctx := context.Background()
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	_, _ = pool.Exec(ctx, `TRUNCATE TABLE subscriptions RESTART IDENTITY`)

	require.NoError(t, err)
	defer pool.Close()
	sr := NewSubRepository(pool)

	start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	uid := uuid.New()
	tcases := []struct {
		Name    string
		ForSave entity.Subscription
		Error   error
	}{
		{
			Name: "valid test DeleteSub",
			ForSave: entity.Subscription{
				ID:          0,
				UserID:      strfmt.UUID(uid.String()),
				ServiceName: "Skillbox",
				Cost:        10_000,
				DateFrom:    start,
				DateTo:      nil,
			},
			Error: nil,
		},
		{
			Name: "error test DeleteSub, not found",
			ForSave: entity.Subscription{
				ID:          0,
				UserID:      strfmt.UUID(uid.String()),
				ServiceName: "Skillbox",
				Cost:        10_000,
				DateFrom:    start,
				DateTo:      nil,
			},
			Error: usecase.ErrSubscriptionNotFound,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, err)
			})
			created, err := sr.SaveSub(ctx, &tc.ForSave)
			require.NoError(t, err)
			delID := created.ID
			if tc.Error != nil {
				delID = created.ID + 1
			}
			err = sr.DeleteSub(ctx, delID)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
				return
			}
			require.NoError(t, err)
			var got entity.Subscription
			row := pool.QueryRow(ctx, `
				SELECT id, user_id, service_name, cost, start_date, end_date
				FROM subscriptions
				WHERE id = $1`, delID)
			scanErr := row.Scan(&got.ID, &got.UserID, &got.ServiceName, &got.Cost, &got.DateFrom, &got.DateTo)
			assert.ErrorIs(t, scanErr, pgx.ErrNoRows)
		})
	}
}

func TestSubRepository_GetSubByID(t *testing.T) {
	ctx := context.Background()
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	_, _ = pool.Exec(ctx, `TRUNCATE TABLE subscriptions RESTART IDENTITY`)

	require.NoError(t, err)
	defer pool.Close()
	sr := NewSubRepository(pool)

	start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	uid := uuid.New()
	tcases := []struct {
		Name    string
		ForSave entity.Subscription
		Error   error
	}{
		{
			Name: "valid test GetSubByID",
			ForSave: entity.Subscription{
				ID:          0,
				UserID:      strfmt.UUID(uid.String()),
				ServiceName: "Skillbox",
				Cost:        10_000,
				DateFrom:    start,
				DateTo:      nil,
			},
			Error: nil,
		},
		{
			Name: "error test GetSubByID, not found",
			ForSave: entity.Subscription{
				ID:          0,
				UserID:      strfmt.UUID(uid.String()),
				ServiceName: "Skillbox",
				Cost:        10_000,
				DateFrom:    start,
				DateTo:      nil,
			},
			Error: usecase.ErrSubscriptionNotFound,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, err)
			})
			created, err := sr.SaveSub(ctx, &tc.ForSave)
			require.NoError(t, err)
			id := created.ID
			if tc.Error != nil {
				id = created.ID + 1
			}
			got, err := sr.GetSubByID(ctx, id)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, *created, *got)
		})
	}
}

func TestSubRepository_ListSubsByFilter(t *testing.T) {
	ctx := context.Background()
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	_, _ = pool.Exec(ctx, `TRUNCATE TABLE subscriptions RESTART IDENTITY`)
	require.NoError(t, err)
	defer pool.Close()
	r := NewSubRepository(pool)

	start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	prev2 := start.AddDate(0, -2, 0)
	next1 := start.AddDate(0, 1, 0)
	userA := uuid.New()
	userB := uuid.New()
	s1, err := r.SaveSub(ctx, &entity.Subscription{
		UserID:      strfmt.UUID(userA.String()),
		ServiceName: "Skillbox",
		Cost:        10000,
		DateFrom:    start,
		DateTo:      nil,
	})
	require.NoError(t, err)
	s2, err := r.SaveSub(ctx, &entity.Subscription{
		UserID:      strfmt.UUID(userA.String()),
		ServiceName: "Netflix",
		Cost:        499,
		DateFrom:    prev2,
		DateTo:      &start,
	})

	require.NoError(t, err)
	s3, err := r.SaveSub(ctx, &entity.Subscription{
		UserID:      strfmt.UUID(userB.String()),
		ServiceName: "Spotify",
		Cost:        299,
		DateFrom:    prev2,
		DateTo:      nil,
	})
	require.NoError(t, err)
	period := &usecase.Period{From: start, To: next1}
	serviceNetflix := "Netflix"
	nonexistentUser := uuid.New()
	tcases := []struct {
		Name     string
		Filter   usecase.SubFilter
		WantLen  int
		AssertFn func(t *testing.T, got []*entity.Subscription)
	}{
		{
			Name:    "period only",
			Filter:  usecase.SubFilter{Period: period},
			WantLen: 3,
			AssertFn: func(t *testing.T, got []*entity.Subscription) {
				assert.True(t, got[0].DateFrom.Before(got[len(got)-1].DateFrom) || got[0].DateFrom.Equal(got[len(got)-1].DateFrom))
			},
		},
		{
			Name:    "filter by user",
			Filter:  usecase.SubFilter{Period: period, UserID: strfmt.UUID(userA.String())},
			WantLen: 2,
			AssertFn: func(t *testing.T, got []*entity.Subscription) {
				assert.Equal(t, strfmt.UUID(userA.String()), got[0].UserID)
				assert.Equal(t, strfmt.UUID(userA.String()), got[1].UserID)
			},
		},
		{
			Name:    "filter by service",
			Filter:  usecase.SubFilter{Period: period, ServiceName: &serviceNetflix},
			WantLen: 1,
			AssertFn: func(t *testing.T, got []*entity.Subscription) {
				assert.Equal(t, "Netflix", got[0].ServiceName)
				assert.Equal(t, s2.ID, got[0].ID)
			},
		},
		{
			Name:     "empty by user",
			Filter:   usecase.SubFilter{Period: period, UserID: strfmt.UUID(nonexistentUser.String())},
			WantLen:  0,
			AssertFn: func(t *testing.T, got []*entity.Subscription) {},
		},
		{
			Name:    "by userA and period returns specific ids",
			Filter:  usecase.SubFilter{Period: period},
			WantLen: 3,
			AssertFn: func(t *testing.T, got []*entity.Subscription) {
				ids := []int64{got[0].ID, got[1].ID, got[2].ID}
				assert.Contains(t, ids, s1.ID)
				assert.Contains(t, ids, s2.ID)
				assert.Contains(t, ids, s3.ID)
			},
		},
	}
	t.Cleanup(func() {
		require.NoError(t, err)
	})
	for _, tc := range tcases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := r.ListSubsByFilter(ctx, tc.Filter)
			require.NoError(t, err)
			assert.Equal(t, tc.WantLen, len(got))
			tc.AssertFn(t, got)
		})
	}
}

func TestSubRepository_CostSubsByFilter(t *testing.T) {
	ctx := context.Background()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	_, _ = pool.Exec(ctx, `TRUNCATE TABLE subscriptions RESTART IDENTITY`)

	r := NewSubRepository(pool)

	start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	prev2 := start.AddDate(0, -2, 0)
	next1 := start.AddDate(0, 1, 0)

	userA := uuid.New()

	_, err = r.SaveSub(ctx, &entity.Subscription{
		UserID:      strfmt.UUID(userA.String()),
		ServiceName: "Skillbox",
		Cost:        10000,
		DateFrom:    start,
		DateTo:      nil,
	})
	require.NoError(t, err)

	_, err = r.SaveSub(ctx, &entity.Subscription{
		UserID:      strfmt.UUID(userA.String()),
		ServiceName: "Netflix",
		Cost:        499,
		DateFrom:    prev2,
		DateTo:      &start,
	})
	require.NoError(t, err)

	_, err = r.SaveSub(ctx, &entity.Subscription{
		UserID:      strfmt.UUID(uuid.New().String()),
		ServiceName: "Spotify",
		Cost:        299,
		DateFrom:    prev2,
		DateTo:      nil,
	})
	require.NoError(t, err)

	period := &usecase.Period{From: start, To: next1}
	serviceNetflix := "Netflix"
	nonexistentUser := uuid.New()

	tcases := []struct {
		Name   string
		Filter usecase.SubFilter
		Want   int64
	}{
		{
			Name:   "period only",
			Filter: usecase.SubFilter{Period: period},
			Want:   20000 + 499 + 299 + 299,
		},
		{
			Name:   "filter by userA",
			Filter: usecase.SubFilter{Period: period, UserID: strfmt.UUID(userA.String())},
			Want:   20000 + 499,
		},
		{
			Name:   "filter by service Netflix",
			Filter: usecase.SubFilter{Period: period, ServiceName: &serviceNetflix},
			Want:   499,
		},
		{
			Name:   "empty by nonexistent user",
			Filter: usecase.SubFilter{Period: period, UserID: strfmt.UUID(nonexistentUser.String())},
			Want:   0,
		},
		{
			Name:   "filter without user",
			Filter: usecase.SubFilter{Period: period},
			Want:   21097,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := r.CostSubsByFilter(ctx, tc.Filter)
			require.NoError(t, err)
			assert.Equal(t, tc.Want, got)
		})
	}
}
