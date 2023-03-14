package mediator

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/bubu256/gophermart_pet/config"
	"github.com/bubu256/gophermart_pet/internal/schema"
	"github.com/bubu256/gophermart_pet/pkg/storage"
	"github.com/rs/zerolog"
)

// реализация бизнес логики приложения, условно посредник между БД и хендлерами

type Mediator struct {
	DB     storage.Storage
	logger zerolog.Logger
}

func New(db storage.Storage, cfg config.CfgMediator, logger zerolog.Logger) *Mediator {
	return &Mediator{DB: db, logger: logger}
}

// принимает структуру логин_пароль, хеширует пароль и пишет базу
func (m *Mediator) SetNewUser(loginPassword schema.LoginPassword) error {
	byte_hash := sha256.Sum256([]byte(loginPassword.Password))
	hash := hex.EncodeToString(byte_hash[:])
	err := m.DB.SetUser(loginPassword.Login, hash)
	return err
}
