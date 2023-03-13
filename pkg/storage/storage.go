package storage

import (
	"github.com/bubu256/gophermart_pet/internal/schema"
)

type Storage interface {
	SetUser(user, password_hash string) error
	GetUserID(login string, hash string) (user_id uint16, err error)
	SetOrder(user_id uint16, number string) error
	SetOrderStatus(number string, status string) error
	GetOrders(user_id uint16) ([]schema.Order, error)
	GetBalance(user_id uint16) (schema.Balance, error)
	SetBonusFlow(user_id uint16, order_number string, amount float64) error
}
