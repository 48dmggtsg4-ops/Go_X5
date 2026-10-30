package model

import "errors"

type Order struct {
	ID       int
	Customer string
	Status   string
}

var ErrNotFound = errors.New("заказ не найден")
