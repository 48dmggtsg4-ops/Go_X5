package repository

import (
	"testing"

	"example.com/lesson9/internal/model"
)

func TestListFiltersBeforePaginationAndSorts(t *testing.T) {
	// Состояние cancelled здесь задано фикстурой: HTTP-демо пока не умеет отменять заказ.
	repo := &Memory{orders: map[int]model.Order{
		4: {ID: 4, Customer: "Дина", Status: "new"},
		1: {ID: 1, Customer: "Алиса", Status: "cancelled"},
		3: {ID: 3, Customer: "Вера", Status: "cancelled"},
		2: {ID: 2, Customer: "Боб", Status: "new"},
	}}
	page, err := repo.List(model.ListOptions{Limit: 1, Offset: 1, Status: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].ID != 4 {
		t.Fatalf("page=%+v; want only id=4", page)
	}
	empty, err := repo.List(model.ListOptions{Limit: 2, Offset: 4})
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty page=%+v err=%v", empty, err)
	}
}
