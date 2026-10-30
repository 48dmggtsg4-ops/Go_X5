package handler

import (
	"example.com/orders/internal/model"
	"example.com/orders/internal/service"
)

type orderCreator interface {
	Create(customer string) (model.Order, error)
}
type Handler struct {
	orders orderCreator
}

func NewHandler(orders *service.Service) *Handler {
	return &Handler{orders: orders}
}

func (h *Handler) Create(customer string) (model.Order, error) {
	return h.orders.Create(customer)
}
