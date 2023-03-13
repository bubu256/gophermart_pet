package schema

import "time"

// пакет содержит структуры данных используемые разными пакетами

// type StatusOrder int

// const (
// 	StatusOrderNew StatusOrder = iota
// 	StatusOrderProcessing
// 	StatusOrderInvalid
// 	StatusOrderProcessed
// )

// структура для ответа от БД, а так для записи ответа сервера в виде json
type Order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float32   `json:"accrual,omitempty"` // заполняется только для статуса PROCESSED
	UploadedAt time.Time `json:"uploaded_at"`
}

// структура для ответа БД о кол-ве бонусов, а так для записи ответа сервера в виде json
type Balance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

// структура для json запроса на списание средств
type OrderSum struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

// структура для json ответа от сервиса аккрола
type AnswerAccrualStatus struct {
	Order   string `json:"order"`
	Status  string `json:"status"` // REGISTERED, INVALID, PROCESSING, PROCESSED
	Accrual int    `json:"accrual"`
}
