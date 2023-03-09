package storage

import (
	"github.com/bubu256/gophermart_pet/internal/schema"
)

type Storage interface {
	SetUser(user, password_hash string) error
	GetPasswordHash(user string) (hash string, err error)
	SetOrder(user string, number string) error
	SetOrderStatus(number string, status string) error
	GetOrders(user string) ([]schema.Order, error)
	GetBalance(user string) (schema.Balance, error)
	SetBonusFlow(user string, amount float64) error
}
