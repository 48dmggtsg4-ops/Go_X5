package service

type DeliveryPricer interface {
	Price(weightGrams, distanceKM int) (int64, error)
}

type BaseTariff struct {
	BasePriceKopecks int64
	PerKMKopecks     int64
	MaxDistanceKM    int
}

func (b BaseTariff) validate(weightGrams, distanceKM int) error {
	// TODO: проверьте вес, положительную дистанцию и максимальную дистанцию.
	return nil
}

func billableKilograms(weightGrams int) int64 {
	// TODO: округлите граммы вверх до целого оплачиваемого килограмма.
	return 0
}

type Courier struct {
	BaseTariff
	PerKGKopecks int64
}

func NewCourier() Courier {
	return Courier{BaseTariff: BaseTariff{20_000, 1_500, 100}, PerKGKopecks: 5_000}
}

func (t Courier) Price(weightGrams, distanceKM int) (int64, error) {
	// TODO: реализуйте формулу тарифа.
	return 0, nil
}

type Pickup struct {
	BaseTariff
	PerKGKopecks int64
}

func NewPickup() Pickup {
	return Pickup{BaseTariff: BaseTariff{10_000, 500, 50}, PerKGKopecks: 2_000}
}

func (t Pickup) Price(weightGrams, distanceKM int) (int64, error) {
	// TODO: реализуйте формулу тарифа.
	return 0, nil
}

type Express struct {
	BaseTariff
	PerKGKopecks int64
}

func NewExpress() Express {
	return Express{BaseTariff: BaseTariff{50_000, 3_000, 300}, PerKGKopecks: 10_000}
}

func (t Express) Price(weightGrams, distanceKM int) (int64, error) {
	// TODO: реализуйте формулу тарифа.
	return 0, nil
}

var (
	_ DeliveryPricer = Courier{}
	_ DeliveryPricer = Pickup{}
	_ DeliveryPricer = Express{}
)
