package mediator

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/bubu256/gophermart_pet/config"
	"github.com/bubu256/gophermart_pet/internal/schema"
	"github.com/bubu256/gophermart_pet/pkg/helpfunc"
	"github.com/bubu256/gophermart_pet/pkg/storage"
	"github.com/rs/zerolog"
)

// реализация бизнес логики приложения, условно посредник между БД и хендлерами

type Mediator struct {
	DB     storage.Storage
	logger zerolog.Logger
	key    []byte
}

func New(db storage.Storage, cfg config.CfgMediator, logger zerolog.Logger) *Mediator {
	if cfg.SecretKey == "" {
		cfg.SecretKey = "Need_Generate_Key"
	}
	key, err := hex.DecodeString(cfg.SecretKey)
	if err != nil {
		logger.Warn().Err(err).Msg("неудачное декодирования ключа; error is here 16888793;")
		key, err = helpfunc.GenerateRandomBytes(32)
		if err != nil {
			logger.Fatal().Err(err).Msg("не удалось сгенерировать набор байт для ключа; error is here 334654654;")
		}
		logger.Warn().Msgf("Сгенерирован новый секретный ключ %x", key)
	}
	return &Mediator{DB: db, logger: logger, key: key}
}

// принимает структуру логин_пароль, хеширует пароль и пишет базу
func (m *Mediator) SetNewUser(loginPassword schema.LoginPassword) error {
	hash := m.getStringHash256(loginPassword.Password)
	err := m.DB.SetUser(loginPassword.Login, hash)
	return err
}

// принимает LoginPassword структуру, проверяет логин пароль и возвращает токен
func (m *Mediator) GetTokenAuthorization(loginPassword schema.LoginPassword) (string, error) {
	hashString := m.getStringHash256(loginPassword.Password)
	userID, err := m.DB.GetUserID(loginPassword.Login, hashString)
	if err != nil {
		m.logger.Debug().Err(err).Msg("error from m.DB.GetUserID(loginPassword.Login, hashString)")
		return "", err
	}
	// генерируем токен на основе userID
	token, err := m.generateNewToken(userID)
	if err != nil {
		m.logger.Debug().Err(err).Msg("error from generateNewToken")
		return "", err
	}
	return token, nil
}

func (m *Mediator) getStringHash256(str string) string {
	byteHash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(byteHash[:])
}

// генерирует новый токен для userID
func (m *Mediator) generateNewToken(userID uint16) (token string, err error) {

	h := hmac.New(sha256.New, m.key)
	// кодируем userID в слайс байт и создаем подпись
	bytesUserID := make([]byte, 16)
	_, err = h.Write(binary.LittleEndian.AppendUint16(bytesUserID, userID))
	if err != nil {
		return "", err
	}
	dst := h.Sum(nil)
	dst = append(bytesUserID, dst...) // содержит байты id и подписи
	// кодируем в hex и отдаем как токен в виде строки
	return hex.EncodeToString(dst), nil
}

// Возвращает userID по токену.
// Внимание! Проверки подлинности токена тут нет.
func (m *Mediator) getUserIDfromToken(token string) (userID uint16, err error) {
	decodeToken, err := hex.DecodeString(token)
	if err != nil {
		return 0, err
	}
	bytesUserID := decodeToken[:16]
	userID = binary.LittleEndian.Uint16(bytesUserID)
	return userID, nil
}

// проверяет подлинность токена
func (m *Mediator) CheckToken(token string) bool {
	decodeToken, err := hex.DecodeString(token)
	if err != nil {
		return false
	}
	bytesUserID := decodeToken[:16]
	sing := decodeToken[4:]
	h := hmac.New(sha256.New, m.key)
	h.Write(bytesUserID)
	dst := h.Sum(nil)
	return hmac.Equal(sing, dst)
}
