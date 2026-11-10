package service_test

import (
	"errors"
	"testing"

	"example.com/delivery/model"
	"example.com/delivery/repository"
	"example.com/delivery/service"
)

func TestCalculatorCourier(t *testing.T) {
	calculator := service.NewCalculator(repository.Default())
	order := model.Order{ID: "order-1", Tariff: "courier", WeightGrams: 2_500, DistanceKM: 10}

	got, err := calculator.Calculate(order)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	const want int64 = 50_000
	if got != want {
		t.Fatalf("Calculate() = %d, want %d", got, want)
	}
}

func TestCalculatorUnknownTariff(t *testing.T) {
	calculator := service.NewCalculator(repository.Default())
	_, err := calculator.Calculate(model.Order{ID: "order-2", Tariff: "drone", WeightGrams: 1_000, DistanceKM: 5})

	if !errors.Is(err, service.ErrUnknownTariff) {
		t.Fatalf("errors.Is(%v, ErrUnknownTariff) = false", err)
	}
	var priceErr *service.OrderPriceError
	if !errors.As(err, &priceErr) {
		t.Fatalf("errors.As(%v, *OrderPriceError) = false", err)
	}
}

func TestCourierRejectsInvalidWeight(t *testing.T) {
	_, err := service.NewCourier().Price(0, 10)
	if !errors.Is(err, service.ErrInvalidWeight) {
		t.Fatalf("errors.Is(%v, ErrInvalidWeight) = false", err)
	}
}

func TestCalculatorPrices(t *testing.T) {
	tests := []struct {
		name     string
		tariff   string
		weight   int
		distance int
		want     int64
	}{
		{"минимальный вес", "courier", 1, 1, 26500},
		{"курьер (максимум км)", "courier", 1000, 100, 175000},
		{"чуть больше 1кг", "courier", 1001, 1, 31500},
		{"самовывоз", "pickup", 1, 1, 12500},
		{"самовывоз (максимум км)", "pickup", 1, 50, 37000},
		{"экспресс", "express", 1, 1, 63000},
		{"экспресс (максимум км)", "express", 1, 300, 960000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator := service.NewCalculator(repository.Default())
			order := model.Order{
				ID:          "test-order",
				Tariff:      tt.tariff,
				WeightGrams: tt.weight,
				DistanceKM:  tt.distance,
			}
			got, err := calculator.Calculate(order)
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Результат %d не совпадает с ожиданием %d", got, tt.want)
			}
		})
	}
}
