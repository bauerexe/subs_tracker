package entity

import (
	"github.com/go-openapi/strfmt"
	"time"
)

// Subscription - структура хранения подписок
type Subscription struct {
	// ID - идентификатор подписки в формате UUID
	ID int64
	// UserID - идентификатор подписанного пользователя
	UserID strfmt.UUID
	// ServiceName - название сервиса, предоставляющего подписку
	ServiceName string
	// Cost - стоимость месячной подписки в рублях
	Cost int64
	// DateFrom - дата начала подписки (месяц и год)
	DateFrom time.Time
	// DateTo - дата оканчания подписки (месяц и год)
	DateTo *time.Time
}
