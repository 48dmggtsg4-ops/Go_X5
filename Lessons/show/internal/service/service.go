package service

import (
	"errors"
	"strings"

	"example.com/lesson9/internal/model"
)

var ErrEmptyCustomer = errors.New("customer is required")

type Repository interface {
	Create(string) (model.Order, error)
	ByID(int) (model.Order, error)
	List(model.ListOptions) ([]model.Order, error)
}
type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(customer string) (model.Order, error) {
	customer = strings.TrimSpace(customer)
	if customer == "" {
		return model.Order{}, ErrEmptyCustomer
	}
	return s.repo.Create(customer)
}
func (s *Service) ByID(id int) (model.Order, error)                      { return s.repo.ByID(id) }
func (s *Service) List(options model.ListOptions) ([]model.Order, error) { return s.repo.List(options) }
