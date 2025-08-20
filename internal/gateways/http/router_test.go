package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	cfg "subs_tracker/internal/config"
	"subs_tracker/internal/entity"
	"subs_tracker/internal/usecase"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var router = gin.New()

type stubSubRepo struct{}

func (s2 stubSubRepo) SaveSub(ctx context.Context, s *entity.Subscription) (*entity.Subscription, error) {
	return &entity.Subscription{ID: 1}, nil
}

func (s2 stubSubRepo) UpdateSub(ctx context.Context, s *entity.Subscription) error {
	return nil
}

func (s2 stubSubRepo) DeleteSub(ctx context.Context, id int64) error {
	return nil
}

func (s2 stubSubRepo) GetSubByID(ctx context.Context, id int64) (*entity.Subscription, error) {
	if id != 1 {
		return nil, nil
	}
	df := time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)
	dt := time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC)

	return &entity.Subscription{
		ID:          1,
		ServiceName: "Netflix",
		Cost:        999,
		UserID:      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
		DateFrom:    df,
		DateTo:      &dt,
	}, nil
}

func (s2 stubSubRepo) ListSubsByFilter(ctx context.Context, f usecase.SubFilter) ([]*entity.Subscription, error) {
	return nil, nil
}

func (s2 stubSubRepo) CostSubsByFilter(ctx context.Context, f usecase.SubFilter) (int64, error) {
	return 0, nil
}

func init() {
	router = SetupGin(cfg.Config{Env: "local"}, UseCases{
		Sub: usecase.NewSubscription(stubSubRepo{})}, slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	)
}

// Все неизвестные пути должны возвращать http.StatusNotFound.
func TestUnknownRoute(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{http.MethodGet, http.MethodGet, http.StatusNotFound},
		{http.MethodPost, http.MethodPost, http.StatusNotFound},
		{http.MethodPut, http.MethodPut, http.StatusNotFound},
		{http.MethodDelete, http.MethodDelete, http.StatusNotFound},
		{http.MethodHead, http.MethodHead, http.StatusNotFound},
		{http.MethodOptions, http.MethodOptions, http.StatusNotFound},
		{http.MethodPatch, http.MethodPatch, http.StatusNotFound},
		{http.MethodConnect, http.MethodConnect, http.StatusNotFound},
		{http.MethodTrace, http.MethodTrace, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.input, "/unknown", nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

// /api/v1/subscriptions
func TestSubscriptionsRoutes(t *testing.T) {
	base := "/api/v1/subscriptions"

	t.Run("GET_subscriptions", func(t *testing.T) {
		t.Run("success_200", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, base+"?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba", nil)
			req.Header.Add("Accept", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			if w.Body.Len() > 0 {
				assert.True(t, json.Valid(w.Body.Bytes()))
			}
		})

		t.Run("requested_unsupported_body_format_406", func(t *testing.T) {
			// Accept: xml → по swagger не поддерживается
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, base, nil)
			req.Header.Add("Accept", "application/xml")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotAcceptable, w.Code)
		})
	})

	t.Run("POST_subscriptions", func(t *testing.T) {
		t.Run("valid_request_201", func(t *testing.T) {
			body := `{
				"service_name": "Yandex Plus",
				"cost": 400,
				"user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date": "07-2025",
				"end_date": "12-2025"
			}`
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, base, bytes.NewBufferString(body))
			req.Header.Add("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusCreated, w.Code)
			assert.True(t, json.Valid(w.Body.Bytes()))
		})

		t.Run("request_body_has_syntax_error_400", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, base, bytes.NewBufferString("{ bad json }"))
			req.Header.Add("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("request_body_has_unsupported_format_415", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, base, bytes.NewBufferString("<xml></xml>"))
			req.Header.Add("Content-Type", "application/xml")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
		})

		t.Run("request_body_is_valid_but_it_has_invalid_data_422", func(t *testing.T) {
			body := `{
				"service_name": "",
				"cost": 0,
				"user_id": "",
				"date_from": "10-2026"
			}`
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, base, bytes.NewBufferString(body))
			req.Header.Add("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			log.Printf("error while testing: %s", w.Body.Bytes())

			assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		})
	})

	t.Run("OPTIONS_subscriptions_204", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, base, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		allowed := strings.Split(w.Header().Get("Allow"), ",")
		assert.Contains(t, allowed, http.MethodOptions)
		assert.Contains(t, allowed, http.MethodGet)
		assert.Contains(t, allowed, http.MethodPost)
	})

	t.Run("OTHER_subscriptions_405", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  int
		}{
			{http.MethodPut, http.MethodPut, http.StatusMethodNotAllowed},
			{http.MethodDelete, http.MethodDelete, http.StatusMethodNotAllowed},
			{http.MethodHead, http.MethodHead, http.StatusMethodNotAllowed},
			{http.MethodPatch, http.MethodPatch, http.StatusMethodNotAllowed},
			{http.MethodConnect, http.MethodConnect, http.StatusMethodNotAllowed},
			{http.MethodTrace, http.MethodTrace, http.StatusMethodNotAllowed},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(tt.input, base, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, tt.want, w.Code)
			})
		}
	})
}

// /api/v1/subscriptions/{id}
func TestSubscriptionsByIDRoutes(t *testing.T) {
	base := "/api/v1/subscriptions"

	t.Run("GET_subscriptions_id", func(t *testing.T) {
		t.Run("exists_200", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, base+"/1", nil)
			req.Header.Add("Accept", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			if w.Body.Len() > 0 {
				assert.True(t, json.Valid(w.Body.Bytes()))
			}
		})

		t.Run("id_has_invalid_format_422", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, base+"/abc", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		})

		t.Run("not_found_404", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, base+"/999999", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("PUT_subscriptions_id", func(t *testing.T) {
		t.Run("valid_request_200", func(t *testing.T) {
			body := `{
				"service_name": "Netflix",
				"cost": 999,
				"user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
				"start_date": "07-2025"
			}`
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, base+"/1", bytes.NewBufferString(body))
			req.Header.Add("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			if w.Body.Len() > 0 {
				assert.True(t, json.Valid(w.Body.Bytes()))
			}
		})

		t.Run("invalid_json_400", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, base+"/1", bytes.NewBufferString("{ bad json }"))
			req.Header.Add("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("unprocessable_entity_422", func(t *testing.T) {
			body := `{
				"service_name": "",
				"cost": -10,
				"user_id": "",
				"date_from": ""
			}`
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, base+"/1", bytes.NewBufferString(body))
			req.Header.Add("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		})

		t.Run("not_found_404", func(t *testing.T) {
			body := `{"service_name":"Spotify","cost":500,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025"}`
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, base+"/999999", bytes.NewBufferString(body))
			req.Header.Add("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("DELETE_subscriptions_id", func(t *testing.T) {
		t.Run("exists_200", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, base+"/1", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("not_found_404", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, base+"/999999", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("OPTIONS_subscriptions_id_204", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, base+"/1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		allowed := strings.Split(w.Header().Get("Allow"), ",")
		assert.Contains(t, allowed, http.MethodOptions)
		assert.Contains(t, allowed, http.MethodGet)
		assert.Contains(t, allowed, http.MethodPut)
		assert.Contains(t, allowed, http.MethodDelete)
	})

	t.Run("OTHER_subscriptions_id_405", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  int
		}{
			{http.MethodPost, http.MethodPost, http.StatusMethodNotAllowed},
			{http.MethodHead, http.MethodHead, http.StatusMethodNotAllowed},
			{http.MethodPatch, http.MethodPatch, http.StatusMethodNotAllowed},
			{http.MethodConnect, http.MethodConnect, http.StatusMethodNotAllowed},
			{http.MethodTrace, http.MethodTrace, http.StatusMethodNotAllowed},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(tt.input, base+"/1", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, tt.want, w.Code)
			})
		}
	})
}

// /api/v1/subscriptions/cost
func TestSubscriptionsCostRoute(t *testing.T) {
	base := "/api/v1/subscriptions/cost"

	t.Run("GET_subscriptions_cost_success_200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, base+"?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&start_date=07-2025&end_date=12-2025", nil)
		req.Header.Add("Accept", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		if w.Body.Len() > 0 {
			assert.True(t, json.Valid(w.Body.Bytes()))
		}
	})

	t.Run("requested_unsupported_body_format_406", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, base, nil)
		req.Header.Add("Accept", "application/xml")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotAcceptable, w.Code)
	})

	t.Run("OPTIONS_subscriptions_cost_204", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, base, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		allowed := strings.Split(w.Header().Get("Allow"), ",")
		assert.Contains(t, allowed, http.MethodOptions)
		assert.Contains(t, allowed, http.MethodGet)
	})

	t.Run("OTHER_subscriptions_cost_405", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  int
		}{
			{http.MethodPost, http.MethodPost, http.StatusMethodNotAllowed},
			{http.MethodPut, http.MethodPut, http.StatusMethodNotAllowed},
			{http.MethodDelete, http.MethodDelete, http.StatusMethodNotAllowed},
			{http.MethodHead, http.MethodHead, http.StatusMethodNotAllowed},
			{http.MethodPatch, http.MethodPatch, http.StatusMethodNotAllowed},
			{http.MethodConnect, http.MethodConnect, http.StatusMethodNotAllowed},
			{http.MethodTrace, http.MethodTrace, http.StatusMethodNotAllowed},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(tt.input, base, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, tt.want, w.Code)
			})
		}
	})
}
