package helpfunc

import "crypto/rand"

// пакет с вспомогательными функциями

// генерирует рандомный набор байт
func GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
