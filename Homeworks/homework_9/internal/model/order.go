package model

// Order сохраняет модель из предыдущего занятия.
type Order struct {
	ID       int
	Customer string
	Status   string
}

type ListOptions struct {
	Limit  int
	Offset int
}
