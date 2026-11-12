package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"example.com/order-service/internal/order"
)

type OrderService struct {
	repository order.Repository
}

func NewOrderService(repository order.Repository) *OrderService {
	return &OrderService{repository: repository}
}

func (s *OrderService) Create(ctx context.Context, input order.CreateOrderInput) (order.Order, error) {
	if len(input.Items) == 0 {
		return order.Order{}, fmt.Errorf(
			"%w: items must not be empty",
			order.ErrInvalidOrder,
		)
	}

	var totalPrice int64

	for i, item := range input.Items {
		if strings.TrimSpace(item.Name) == "" {
			return order.Order{}, fmt.Errorf(
				"%w: item %d name must not be empty",
				order.ErrInvalidOrder, i+1,
			)
		}

		if item.Quantity <= 0 {
			return order.Order{}, fmt.Errorf(
				"%w: item %d quantity must be positive",
				order.ErrInvalidOrder, i+1,
			)
		}

		if item.Price <= 0 {
			return order.Order{}, fmt.Errorf(
				"%w: item %d price must be positive",
				order.ErrInvalidOrder, i+1,
			)
		}

		quantity := int64(item.Quantity)

		if item.Price > math.MaxInt64/quantity {
			return order.Order{}, fmt.Errorf(
				"%w: item %d cost is too large",
				order.ErrInvalidOrder, i+1,
			)
		}

		itemTotal := item.Price * quantity

		if totalPrice > math.MaxInt64-itemTotal {
			return order.Order{}, fmt.Errorf(
				"%w: total price is too large",
				order.ErrInvalidOrder,
			)
		}

		totalPrice += itemTotal
	}

	now := time.Now()

	value := order.Order{
		Items:      input.Items,
		TotalPrice: totalPrice,
		Status:     order.StatusNew,
		CreatedAt:  now,
		History: []order.StatusChange{
			{
				Status:    order.StatusNew,
				ChangedAt: now,
			},
		},
	}

	created, err := s.repository.Create(ctx, value)
	if err != nil {
		return order.Order{}, fmt.Errorf("create order: %w", err)
	}

	return created, nil
}

func (s *OrderService) GetByID(ctx context.Context, id int64) (order.Order, error) {
	value, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return order.Order{}, fmt.Errorf("get order %d: %w", id, err)
	}

	return value, nil
}

func (s *OrderService) List(ctx context.Context) ([]order.Order, error) {
	values, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}

	return values, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, id int64, status order.Status) (order.Order, error) {
	switch status {
	case order.StatusNew,
		order.StatusProcessing,
		order.StatusCompleted,
		order.StatusCancelled:
	default:
		return order.Order{}, fmt.Errorf(
			"%w: %q",
			order.ErrInvalidStatus, status,
		)
	}

	value, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return order.Order{}, fmt.Errorf("get order %d: %w", id, err)
	}

	allowed := false

	switch value.Status {
	case order.StatusNew:
		allowed = status == order.StatusProcessing ||
			status == order.StatusCancelled

	case order.StatusProcessing:
		allowed = status == order.StatusCompleted ||
			status == order.StatusCancelled
	}

	if !allowed {
		return order.Order{}, fmt.Errorf(
			"%w: %s -> %s",
			order.ErrInvalidStatusTransition,
			value.Status,
			status,
		)
	}

	value.Status = status
	value.History = append(value.History, order.StatusChange{
		Status:    status,
		ChangedAt: time.Now(),
	})

	updated, err := s.repository.Update(ctx, value)
	if err != nil {
		return order.Order{}, fmt.Errorf("update order %d: %w", id, err)
	}

	return updated, nil
}

func (s *OrderService) History(ctx context.Context, id int64) ([]order.StatusChange, error) {
	value, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf(
			"get history for order %d: %w",
			id, err,
		)
	}

	return value.History, nil
}
