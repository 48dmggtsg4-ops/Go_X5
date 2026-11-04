package service

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidWeight    = errors.New("некорректный вес")
	ErrInvalidDistance  = errors.New("некорректная дистанция")
	ErrDistanceTooLarge = errors.New("слишком большая дистанция")
	ErrUnknownTariff    = errors.New("неизвестный тариф")
)

type OrderPriceError struct {
	OrderID string
	Tariff  string
	Err     error
}

func (e *OrderPriceError) Error() string {
	// TODO: добавьте контекст заказа и тарифа, не теряя причину.
	return fmt.Sprintf("заказ %s, тариф %s", e.OrderID, e.Tariff)
}

func (e *OrderPriceError) Unwrap() error {
	// TODO: верните исходную причину.
	return nil
}
