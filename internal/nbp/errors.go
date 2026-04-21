package nbp

import "errors"

var (
	ErrNoData     = errors.New("brak danych w API NBP")
	ErrTimeout    = errors.New("przekroczono limit czasu żądania")
	ErrConnection = errors.New("brak połączenia z API NBP")
	ErrNotFound   = errors.New("nie znaleziono kursu")
)
