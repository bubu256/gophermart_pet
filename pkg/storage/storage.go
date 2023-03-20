package storage

import (
	"github.com/bubu256/gophermart_pet/internal/schema"
)

type Storage interface {
	SetUser(user, passwordHash string) error
	GetUserID(login string, hash string) (userID uint16, err error)
	SetOrder(userID uint16, number string) error
	SetOrderStatus(number string, status schema.StatusOrder, accrual float32) error
	GetOrders(userID uint16) ([]schema.Order, error)
	GetBalance(userID uint16) (schema.Balance, error)
	SetBonusFlow(userID uint16, orderNumber string, amount float32) error
	GetUserIDfromOrders(numberOrder string) (userID uint16, err error)
	GetBonusFlow(userID uint16) ([]schema.OrderSum, error)
	GetWaitingOrders() ([]schema.Order, error)
	Ping() error
}
