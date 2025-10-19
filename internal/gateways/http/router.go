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

func parseMonthYear(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		"01-2006",
		"2006-01-02",
		"2006-01",
	}
	var lastErr error
	for _, layout := range layouts {
		if layout == "" {
			continue
		}
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

func setupRouter(r *gin.Engine, u UseCases) {
	r.HandleMethodNotAllowed = true

	r.Use(gin.Recovery())

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	{
		v1 := r.Group("api/v1/")
		setupSubscription(v1, u)
		setupSubscriptionsId(v1, u)
		setupSubscriptionsCost(v1, u)
	}
}

func setupSubscription(r *gin.RouterGroup, u UseCases) {
	r.GET("/subscriptions", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}

		uidStr := c.Query("user_id")
		svc := c.Query("service_name")
		fromStr := c.Query("start_date")
		toStr := c.Query("end_date")
		limitStr := c.Query("limit")
		offsetStr := c.Query("offset")

		var f usecase.SubFilter

		var uid uuid.UUID
		var err error

		if uid, err = uuid.Parse(uidStr); err != nil && uidStr != "" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "uuid invalid"})
			return
		}

		if uid != uuid.Nil {
			f.UserID = strfmt.UUID(uid.String())
		}

		if svc != "" {
			f.ServiceName = &svc
		}

		if limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit < 0 {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid limit"})
				return
			}
			f.Limit = limit
		}

		if offsetStr != "" {
			offset, err := strconv.Atoi(offsetStr)
			if err != nil || offset < 0 {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid offset"})
				return
			}
			f.Offset = offset
		}

		if fromStr != "" || toStr != "" {
			var p usecase.Period
			if fromStr != "" {
				t, err := parseMonthYear(fromStr)
				if err != nil {
					c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period: from"})
					return
				}
				p.From = t
			}
			if toStr != "" {
				t, err := parseMonthYear(toStr)
				if err != nil {
					c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period: to"})
					return
				}
				p.To = t
			}
			f.Period = &p
		}

		subs, err := u.Sub.ListSubsByFilter(c, f)
		switch {
		case errors.Is(err, usecase.ErrInvalidSubscription):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid subscriptions data"})
			return
		case errors.Is(err, usecase.ErrInvalidPeriod):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period"})
			return
		case errors.Is(err, usecase.ErrInvalidPagination):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid pagination"})
			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		resp := make([]*generated.Subscription, 0, len(subs))
		for _, s := range subs {
			sn := s.ServiceName
			cost := s.Cost

			uidSF := strfmt.UUID(s.UserID.String())
			df := s.DateFrom.Format("01-2006")
			var dt string
			if s.DateTo != nil {
				dt = s.DateTo.Format("01-2006")
			}

			item := &generated.Subscription{
				SubscriptionInput: generated.SubscriptionInput{
					ServiceName: &sn,
					Cost:        &cost,
					UserID:      &uidSF,
					StartDate:   &df,
					EndDate:     dt,
				},
				SubscriptionID: generated.SubscriptionID{ID: s.ID},
			}
			resp = append(resp, item)
		}

		c.JSON(http.StatusOK, resp)
	})

	r.POST("/subscriptions", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}
		if c.ContentType() != "" && c.ContentType() != "application/json" {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Use application/json"})
			return
		}

		var input *generated.SubscriptionInput

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := input.Validate(strfmt.Default); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		dateFrom, err := parseMonthYear(*input.StartDate)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, errors.New("invalid period: date from"))
			return
		}

		var dateTo time.Time

		subscription := &entity.Subscription{
			ID:          0,
			UserID:      *input.UserID,
			ServiceName: *input.ServiceName,
			Cost:        *input.Cost,
			DateFrom:    dateFrom,
		}
		if input.EndDate != "" {
			dateTo, err = parseMonthYear(input.EndDate)
			if err != nil {
				c.JSON(http.StatusUnprocessableEntity, errors.New("invalid period: date to"))
				return
			}
			subscription.DateTo = &dateTo
		}

		created, err := u.Sub.RegisterSub(c, subscription)

		switch {
		case errors.Is(err, usecase.ErrInvalidSubscription):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid subscriptions data"})
			return
		case errors.Is(err, usecase.ErrInvalidPeriod):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period"})
			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		case created == nil:
			c.JSON(http.StatusCreated, gin.H{"error": "nil result from RegisterSub"})
			return
		}
		resp := generated.Subscription{
			SubscriptionInput: *input,
			SubscriptionID:    generated.SubscriptionID{ID: created.ID},
		}
		c.JSON(http.StatusCreated, resp)
	})

	r.OPTIONS("/subscriptions", func(c *gin.Context) {
		c.Writer.Header().Set("Allow", "POST,OPTIONS,GET")
		c.Status(http.StatusNoContent)
	})
}

func setupSubscriptionsId(r *gin.RouterGroup, u UseCases) {
	r.GET("/subscriptions/:id", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid id"})
			return
		}

		sub, err := u.Sub.GetSubByID(c, id)
		switch {
		case errors.Is(err, usecase.ErrInvalidID):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid id"})
			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		case sub == nil:
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		name := sub.ServiceName
		cost := sub.Cost
		uid := sub.UserID
		df := sub.DateFrom.Format("01-2006")
		var dt string
		if sub.DateTo != nil {
			dt = sub.DateTo.Format("01-2006")
		}

		resp := generated.Subscription{
			SubscriptionInput: generated.SubscriptionInput{
				ServiceName: &name,
				Cost:        &cost,
				UserID:      &uid,
				StartDate:   &df,
				EndDate:     dt,
			},
			SubscriptionID: generated.SubscriptionID{ID: sub.ID},
		}
		c.JSON(http.StatusOK, resp)
	})

	r.PUT("/subscriptions/:id", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}
		if c.ContentType() != "" && c.ContentType() != "application/json" {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Use application/json"})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid id"})
			return
		}

		var input *generated.SubscriptionInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := input.Validate(strfmt.Default); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		df, err := parseMonthYear(*input.StartDate)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period: date from"})
			return
		}

		var newSub = entity.Subscription{
			ID:          id,
			UserID:      *input.UserID,
			ServiceName: *input.ServiceName,
			Cost:        *input.Cost,
			DateFrom:    df,
		}

		var dt *time.Time
		if input.EndDate != "" {
			v, err := parseMonthYear(input.EndDate)
			if err != nil {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period: date to"})
				return
			}
			dt = &v
			newSub.DateTo = dt
		}

		updated, err := u.Sub.UpdateSub(c, &newSub)

		if errors.Is(err, usecase.ErrInvalidID) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid id"})
			return
		}
		if errors.Is(err, usecase.ErrInvalidSubscription) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid subscriptions data"})
			return
		}
		if errors.Is(err, usecase.ErrInvalidPeriod) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period"})
			return
		}
		if err != nil || updated == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		name := updated.ServiceName
		cost := updated.Cost
		uid := updated.UserID
		outDF := updated.DateFrom.Format("01-2006")
		var outDT string
		if updated.DateTo != nil {
			outDT = updated.DateTo.Format("01-2006")
		}

		resp := generated.Subscription{
			SubscriptionInput: generated.SubscriptionInput{
				ServiceName: &name,
				Cost:        &cost,
				UserID:      &uid,
				StartDate:   &outDF,
				EndDate:     outDT,
			},
			SubscriptionID: generated.SubscriptionID{ID: updated.ID},
		}
		c.JSON(http.StatusOK, resp)
	})

	r.DELETE("/subscriptions/:id", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		deleted, err := u.Sub.DeleteSub(c, id)
		switch {
		case errors.Is(err, usecase.ErrInvalidID):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid id"})
			return
		case err != nil:
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		case deleted == nil:
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		name := deleted.ServiceName
		cost := deleted.Cost
		uid := deleted.UserID
		df := deleted.DateFrom.Format("01-2006")
		var dt string
		if deleted.DateTo != nil {
			dt = deleted.DateTo.Format("01-2006")
		}

		resp := generated.Subscription{
			SubscriptionInput: generated.SubscriptionInput{
				ServiceName: &name,
				Cost:        &cost,
				UserID:      &uid,
				StartDate:   &df,
				EndDate:     dt,
			},
			SubscriptionID: generated.SubscriptionID{ID: deleted.ID},
		}
		c.JSON(http.StatusOK, resp)
	})

	r.OPTIONS("/subscriptions/:id", func(c *gin.Context) {
		c.Writer.Header().Set("Allow", "PUT,OPTIONS,GET,DELETE")
		c.Status(http.StatusNoContent)
	})
}

func setupSubscriptionsCost(r *gin.RouterGroup, u UseCases) {
	methodNA := func(c *gin.Context) {
		c.Header("Allow", "GET,OPTIONS")
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
	for _, m := range []string{
		http.MethodPut,
		http.MethodDelete,
	} {
		r.Handle(m, "/subscriptions/cost", methodNA)
	}

	r.GET("/subscriptions/cost", func(c *gin.Context) {
		if !requireAcceptJSON(c) {
			return
		}

		userID := strings.TrimSpace(c.Query("user_id"))

		serviceName := c.Query("service_name")
		fromStr := c.Query("start_date")
		toStr := c.Query("end_date")

		var fromTime, toTime time.Time
		var err error

		if fromTime, err = parseMonthYear(fromStr); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid start_date"})
			return
		}
		if toTime, err = parseMonthYear(toStr); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid end_date"})
			return
		}
		if fromTime.After(toTime) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "from must be <= to"})
			return
		}
		var uid uuid.UUID

		if uid, err = uuid.Parse(userID); err != nil && userID != "" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "uuid invalid"})
			return
		}

		p := usecase.Period{
			From: fromTime,
			To:   toTime,
		}
		period := &p

		f := usecase.SubFilter{
			Period: period,
		}

		if uid != uuid.Nil {
			f.UserID = strfmt.UUID(uid.String())
		}

		if serviceName != "" {
			f.ServiceName = &serviceName
		}

		total, err := u.Sub.CostSubsByFilter(c, f)

		switch {
		case errors.Is(err, usecase.ErrInvalidPeriod):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid period"})
			return
		case errors.Is(err, usecase.ErrInvalidPagination):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid pagination"})
			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"total": total})
	})

	r.OPTIONS("/subscriptions/cost", func(c *gin.Context) {
		c.Writer.Header().Set("Allow", "GET,OPTIONS")
		c.Status(http.StatusNoContent)
	})
}

func acceptsJSON(h string) bool {
	if h == "" || h == "*/*" {
		return true
	}
	parts := strings.Split(h, ",")
	for _, p := range parts {
		mt := strings.TrimSpace(strings.SplitN(p, ";", 2)[0])
		if mt == "application/json" || mt == "*/*" {
			return true
		}
	}
	return false
}

func requireAcceptJSON(c *gin.Context) bool {
	if acceptsJSON(c.GetHeader("Accept")) {
		return true
	}
	c.JSON(http.StatusNotAcceptable, gin.H{"error": "Accept application/json only"})
	return false
}
