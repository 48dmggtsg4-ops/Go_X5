package repository

import (
	"errors"
	"sort"
	"sync"

	"example.com/lesson9/internal/model"
)

var ErrNotFound = errors.New("order not found")

type Memory struct {
	mu     sync.Mutex
	nextID int
	orders map[int]model.Order
}

func New() *Memory { return &Memory{nextID: 1, orders: make(map[int]model.Order)} }

func (m *Memory) Create(customer string) (model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order := model.Order{ID: m.nextID, Customer: customer, Status: "new"}
	m.orders[order.ID] = order
	m.nextID++
	return order, nil
}
func (m *Memory) ByID(id int) (model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, ok := m.orders[id]
	if !ok {
		return model.Order{}, ErrNotFound
	}
	return order, nil
}
func (m *Memory) List(options model.ListOptions) ([]model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	orders := make([]model.Order, 0)
	for _, order := range m.orders {
		if options.Status == "" || order.Status == options.Status {
			orders = append(orders, order)
		}
	}
	// Почему нельзя сначала обрезать результат, а потом применить фильтр?
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	if options.Offset >= len(orders) {
		return []model.Order{}, nil
	}
	orders = orders[options.Offset:]
	// Сравниваем длину, не складываем offset+limit: сумма может переполниться.
	if len(orders) > options.Limit {
		orders = orders[:options.Limit]
	}
	return orders, nil
}
