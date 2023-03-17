package errorapp

import "errors"

// пакет содержит кастомные ошибки проекта

var ErrDuplicate error = errors.New("recording is not possible due to duplication;")
var ErrEmptyInsert error = errors.New("empty insert;")
var ErrWrongLoginPassword error = errors.New("wrong login or password")
var ErrAlreadyAdded error = errors.New("order number already added")
var ErrEmptyResult error = errors.New("empty result for query")
var ErrNotEnoughFunds error = errors.New("there are not enough bonuses on the balance")
