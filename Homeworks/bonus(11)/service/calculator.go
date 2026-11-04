package service

import "example.com/delivery/model"

type TariffRepository interface {
	Find(name string) (DeliveryPricer, error)
}

type Calculator struct {
	repository TariffRepository
}

func NewCalculator(repository TariffRepository) *Calculator {
	return &Calculator{repository: repository}
}

func (c *Calculator) Calculate(order model.Order) (int64, error) {
	// TODO: получите тариф, рассчитайте цену и оберните ошибку в OrderPriceError.
	return 0, nil
}
