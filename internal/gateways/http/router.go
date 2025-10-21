package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
	"subs_tracker/internal/entity"
	"subs_tracker/internal/entity/generated"
	"subs_tracker/internal/usecase"
	"time"
)

// parseMonthYear parses several date layouts and normalizes to the first day of the month (UTC).
func parseMonthYear(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{"01-2006", "2006-01-02", "2006-01"}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err != nil {
			lastErr = err
			continue
		}
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	}
	if lastErr == nil {
		lastErr = errors.New("empty date value")
	}
	return time.Time{}, lastErr
}

// setupRouter wires all routes and basic middleware.
func setupRouter(r *gin.Engine, u UseCases) {
	r.HandleMethodNotAllowed = true
	r.Use(gin.Recovery())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	v1 := r.Group("api/v1/")
	setupSubscription(v1, u)
	setupSubscriptionsId(v1, u)
	setupSubscriptionsCost(v1, u)
}

// setupSubscription registers list/create routes for subscriptions.
func setupSubscription(r *gin.RouterGroup, u UseCases) {
	r.GET("/subscriptions", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}

		var f usecase.SubFilter

		if uidStr := c.Query("user_id"); uidStr != "" {
			uid, err := uuid.Parse(uidStr)
			if err != nil {
				jsonErr(c, http.StatusUnprocessableEntity, "uuid invalid")
				return
			}
			f.UserID = strfmt.UUID(uid.String())
		}

		if svc := c.Query("service_name"); svc != "" {
			f.ServiceName = &svc
		}

		if v := c.Query("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				jsonErr(c, http.StatusUnprocessableEntity, "invalid limit")
				return
			}
			f.Limit = n
		}
		if v := c.Query("offset"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				jsonErr(c, http.StatusUnprocessableEntity, "invalid offset")
				return
			}
			f.Offset = n
		}

		fromStr, toStr := c.Query("start_date"), c.Query("end_date")
		if fromStr != "" || toStr != "" {
			var p usecase.Period
			if fromStr != "" {
				t, err := parseMonthYear(fromStr)
				if err != nil {
					jsonErr(c, http.StatusUnprocessableEntity, "invalid period: from")
					return
				}
				p.From = t
			}
			if toStr != "" {
				t, err := parseMonthYear(toStr)
				if err != nil {
					jsonErr(c, http.StatusUnprocessableEntity, "invalid period: to")
					return
				}
				p.To = t
			}
			f.Period = &p
		}

		subs, err := u.Sub.ListSubsByFilter(c, f)
		if handled := handleUsecaseErr(c, err); handled {
			return
		}

		resp := make([]*generated.Subscription, 0, len(subs))
		for _, s := range subs {
			cp := s
			item := buildSubDTO(cp)
			resp = append(resp, &item)
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/subscriptions", func(c *gin.Context) {
		if !requireAcceptJSON(c) || !requireJSONContent(c) {
			return
		}

		var input *generated.SubscriptionInput
		if err := c.ShouldBindJSON(&input); err != nil {
			jsonErr(c, http.StatusBadRequest, err.Error())
			return
		}
		if err := input.Validate(strfmt.Default); err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, err.Error())
			return
		}

		dateFrom, err := parseMonthYear(*input.StartDate)
		if err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, "invalid period: date from")
			return
		}

		sub := &entity.Subscription{
			UserID:      *input.UserID,
			ServiceName: *input.ServiceName,
			Cost:        *input.Cost,
			DateFrom:    dateFrom,
		}
		if input.EndDate != "" {
			v, err := parseMonthYear(input.EndDate)
			if err != nil {
				jsonErr(c, http.StatusUnprocessableEntity, "invalid period: date to")
				return
			}
			sub.DateTo = &v
		}

		created, err := u.Sub.RegisterSub(c, sub)
		if handled := handleUsecaseErr(c, err); handled {
			return
		}
		if created == nil {
			jsonErr(c, http.StatusCreated, "nil result from RegisterSub")
			return
		}
		out := buildSubDTO(created)
		c.JSON(http.StatusCreated, out)
	})

	r.OPTIONS("/subscriptions", func(c *gin.Context) {
		c.Header("Allow", "POST,OPTIONS,GET")
		c.Status(http.StatusNoContent)
	})
}

// setupSubscriptionsId registers get/update/delete by id routes.
func setupSubscriptionsId(r *gin.RouterGroup, u UseCases) {
	r.GET("/subscriptions/:id", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, "invalid id")
			return
		}
		sub, err := u.Sub.GetSubByID(c, id)
		if errors.Is(err, usecase.ErrInvalidID) {
			jsonErr(c, http.StatusUnprocessableEntity, "invalid id")
			return
		}
		if err != nil {
			jsonErr(c, http.StatusInternalServerError, "internal error")
			return
		}
		if sub == nil {
			jsonErr(c, http.StatusNotFound, "not found")
			return
		}
		out := buildSubDTO(sub)
		c.JSON(http.StatusOK, out)
	})

	r.PUT("/subscriptions/:id", func(c *gin.Context) {
		if !requireAcceptJSON(c) || !requireJSONContent(c) {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, "invalid id")
			return
		}

		var input *generated.SubscriptionInput
		if err := c.ShouldBindJSON(&input); err != nil {
			jsonErr(c, http.StatusBadRequest, err.Error())
			return
		}
		if err := input.Validate(strfmt.Default); err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, err.Error())
			return
		}

		df, err := parseMonthYear(*input.StartDate)
		if err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, "invalid period: date from")
			return
		}

		newSub := entity.Subscription{
			ID:          id,
			UserID:      *input.UserID,
			ServiceName: *input.ServiceName,
			Cost:        *input.Cost,
			DateFrom:    df,
		}
		if input.EndDate != "" {
			v, err := parseMonthYear(input.EndDate)
			if err != nil {
				jsonErr(c, http.StatusUnprocessableEntity, "invalid period: date to")
				return
			}
			newSub.DateTo = &v
		}

		updated, err := u.Sub.UpdateSub(c, &newSub)
		switch {
		case errors.Is(err, usecase.ErrInvalidID):
			jsonErr(c, http.StatusUnprocessableEntity, "invalid id")
			return
		case errors.Is(err, usecase.ErrInvalidSubscription):
			jsonErr(c, http.StatusUnprocessableEntity, "invalid subscriptions data")
			return
		case errors.Is(err, usecase.ErrInvalidPeriod):
			jsonErr(c, http.StatusUnprocessableEntity, "invalid period")
			return
		case err != nil || updated == nil:
			jsonErr(c, http.StatusNotFound, "not found")
			return
		}

		out := buildSubDTO(updated)
		c.JSON(http.StatusOK, out)
	})

	r.DELETE("/subscriptions/:id", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			jsonErr(c, http.StatusBadRequest, "invalid id")
			return
		}
		deleted, err := u.Sub.DeleteSub(c, id)
		switch {
		case errors.Is(err, usecase.ErrInvalidID):
			jsonErr(c, http.StatusUnprocessableEntity, "invalid id")
			return
		case err != nil, deleted == nil:
			jsonErr(c, http.StatusNotFound, "not found")
			return
		}
		out := buildSubDTO(deleted)
		c.JSON(http.StatusOK, out)
	})

	r.OPTIONS("/subscriptions/:id", func(c *gin.Context) {
		c.Header("Allow", "PUT,OPTIONS,GET,DELETE")
		c.Status(http.StatusNoContent)
	})
}

// setupSubscriptionsCost registers aggregate cost endpoint.
func setupSubscriptionsCost(r *gin.RouterGroup, u UseCases) {
	methodNA := func(c *gin.Context) {
		c.Header("Allow", "GET,OPTIONS")
		jsonErr(c, http.StatusMethodNotAllowed, "method not allowed")
	}
	for _, m := range []string{http.MethodPut, http.MethodDelete} {
		r.Handle(m, "/subscriptions/cost", methodNA)
	}

	r.GET("/subscriptions/cost", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}

		fromTime, err := parseMonthYear(c.Query("start_date"))
		if err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, "invalid start_date")
			return
		}
		toTime, err := parseMonthYear(c.Query("end_date"))
		if err != nil {
			jsonErr(c, http.StatusUnprocessableEntity, "invalid end_date")
			return
		}
		if fromTime.After(toTime) {
			jsonErr(c, http.StatusUnprocessableEntity, "from must be <= to")
			return
		}

		f := usecase.SubFilter{
			Period: &usecase.Period{From: fromTime, To: toTime},
		}

		if userID := strings.TrimSpace(c.Query("user_id")); userID != "" {
			uid, err := uuid.Parse(userID)
			if err != nil {
				jsonErr(c, http.StatusUnprocessableEntity, "uuid invalid")
				return
			}
			f.UserID = strfmt.UUID(uid.String())
		}

		if sn := c.Query("service_name"); sn != "" {
			f.ServiceName = &sn
		}

		total, err := u.Sub.CostSubsByFilter(c, f)
		if handled := handleUsecaseErr(c, err); handled {
			return
		}
		c.JSON(http.StatusOK, gin.H{"total": total})
	})

	r.OPTIONS("/subscriptions/cost", func(c *gin.Context) {
		c.Header("Allow", "GET,OPTIONS")
		c.Status(http.StatusNoContent)
	})
}

// acceptsJSON checks if Accept header allows application/json.
func acceptsJSON(h string) bool {
	if h == "" || h == "*/*" {
		return true
	}
	for _, p := range strings.Split(h, ",") {
		mt := strings.TrimSpace(strings.SplitN(p, ";", 2)[0])
		if mt == "application/json" || mt == "*/*" {
			return true
		}
	}
	return false
}

// requireAcceptJSON enforces Accept: application/json.
func requireAcceptJSON(c *gin.Context) bool {
	if acceptsJSON(c.GetHeader("Accept")) {
		return true
	}
	jsonErr(c, http.StatusNotAcceptable, "Accept application/json only")
	return false
}

// requireJSONContent enforces Content-Type: application/json (if provided).
func requireJSONContent(c *gin.Context) bool {
	ct := strings.TrimSpace(c.ContentType())
	if ct == "" || ct == "application/json" {
		return true
	}
	jsonErr(c, http.StatusUnsupportedMediaType, "Use application/json")
	return false
}

// buildSubDTO maps domain Subscription to generated transport model.
func buildSubDTO(s *entity.Subscription) generated.Subscription {
	name := s.ServiceName
	cost := s.Cost
	uid := s.UserID
	start := s.DateFrom.Format("01-2006")
	var end string
	if s.DateTo != nil {
		end = s.DateTo.Format("01-2006")
	}
	return generated.Subscription{
		SubscriptionInput: generated.SubscriptionInput{
			ServiceName: &name,
			Cost:        &cost,
			UserID:      &uid,
			StartDate:   &start,
			EndDate:     end,
		},
		SubscriptionID: generated.SubscriptionID{ID: s.ID},
	}
}

// jsonErr sends a JSON error with status code.
func jsonErr(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// handleUsecaseErr maps domain errors to HTTP responses; returns true if handled.
func handleUsecaseErr(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, usecase.ErrInvalidID),
		errors.Is(err, usecase.ErrInvalidSubscription),
		errors.Is(err, usecase.ErrInvalidPagination),
		errors.Is(err, usecase.ErrInvalidPeriod):
		jsonErr(c, http.StatusUnprocessableEntity, strings.TrimPrefix(err.Error(), ": "))
		return true
	default:
		jsonErr(c, http.StatusInternalServerError, "internal error")
		return true
	}
}
