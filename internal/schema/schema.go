package schema

import (
	"fmt"
	"strings"
	"time"
)

// пакет содержит структуры данных используемые разными пакетами

// возможный статусы заказа
type StatusOrder string

const (
	StatusOrderNew        StatusOrder = "NEW"
	StatusOrderProcessing StatusOrder = "PROCESSING"
	StatusOrderInvalid    StatusOrder = "INVALID"
	StatusOrderProcessed  StatusOrder = "PROCESSED"
)

// возможные статусы ответа от аккрол сервиса
type AccrualStatus string

const (
	AccrualStatusRegistered AccrualStatus = "REGISTERED"
	AccrualStatusInvalid    AccrualStatus = "INVALID"
	AccrualStatusProcessing AccrualStatus = "PROCESSING"
	AccrualStatusProcessed  AccrualStatus = "PROCESSED"
)

// тип для представления кодирования времени в json в формат RFC3339
type TimeRFC3339 struct {
	time.Time
}

func (t *TimeRFC3339) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), `"`) // remove quotes
	if s == "null" {
		return
	}
	t.Time, err = time.Parse(time.RFC3339, s)
	return
}

func (t TimeRFC3339) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return []byte(fmt.Sprintf(`"%s"`, t.Time.Format(time.RFC3339))), nil
}

// структура для ответа от БД, а так для записи ответа сервера в виде json
type Order struct {
	Number     string      `json:"number"`
	Status     string      `json:"status"`
	Accrual    float32     `json:"accrual,omitempty"` // заполняется только для статуса PROCESSED
	UploadedAt TimeRFC3339 `json:"uploaded_at"`
}

// структура для ответа БД о кол-ве бонусов, а так для записи ответа сервера в виде json
type Balance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

// структура для json запроса на списание средств
type OrderSum struct {
	Order       string      `json:"order"`
	Sum         float32     `json:"sum"`
	ProcessedAt TimeRFC3339 `json:"processed_at,omitempty"`
}

// структура для json ответа от сервиса аккрола
type AnswerAccrualService struct {
	Order   string        `json:"order"`
	Status  AccrualStatus `json:"status"` // REGISTERED, INVALID, PROCESSING, PROCESSED
	Accrual float32       `json:"accrual,omitempty"`
}

type LoginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
