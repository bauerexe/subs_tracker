package entity

import (
	"github.com/go-openapi/strfmt"
	"time"
)

// Subscription - entity with subscription information
type Subscription struct {
	// ID - subscription identifier in UUID format
	ID int64
	// UserID - identifier of the subscribed user
	UserID strfmt.UUID
	// ServiceName - name of the service providing the subscription
	ServiceName string
	// Cost - monthly subscription cost in rubles
	Cost int64
	// DateFrom - subscription start date (month and year)
	DateFrom time.Time
	// DateTo - subscription end date (month and year)
	DateTo *time.Time
}
