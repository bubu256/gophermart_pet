package errorapp

import "errors"

// пакет содержит кастомные ошибки проекта

var ErrDuplicate error = errors.New("recording is not possible due to duplication;")
var ErrEmptyInsert error = errors.New("empty insertl;")
