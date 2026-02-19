package domain

import "errors"

// ErrNotFound Если в хранилищне не нашлось нужных данных
var ErrNotFound = errors.New("not found")
var ErrTimeout = errors.New("timeout exceeded")

// ErrCancelled если произошла отмена
var ErrCancelled = errors.New("cancelled")

// ErrInternal что-то неожиданное или баг
var ErrInternal = errors.New("internal error")

// ErrUnavailable внешний источник недоступен,при это ошибка retrayble
var ErrUnavailable = errors.New("unavailable error")

// ErrPermanent ошибки, которым не поможет ретрай
var ErrPermanent = errors.New("unavailable error")

// ErrRepeatedRequest говорит нам о том, что в ДБ уже пошел запрос на получения значения по ключу. Если в этот момент
// приходят такие же запросы, то мы их отрубаем. Они потом сходят в кэш
var ErrRepeatedRequest = errors.New("repeated request")
