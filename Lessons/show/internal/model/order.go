package model

// Order сохраняет модель из предыдущего занятия.
type Order struct {
	ID       int
	Customer string
	Status   string
}

// ListOptions получает проверенные параметры: Limit 1..100 и Offset >= 0.
type ListOptions struct {
	Limit  int
	Offset int
	Status string
}
