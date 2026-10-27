package analytics

type Sale struct {
	Product          string
	Category         string
	Quantity         int
	UnitPriceKopecks int64
	Cancelled        bool
}

type Report struct {
	RevenueKopecks    int64
	UnitsSold         int
	RevenueByCategory map[string]int64
	TopProduct        string
	InvalidSales      int
}

func AnalyzeSales(sales []Sale) Report {
	report := Report{RevenueByCategory: make(map[string]int64)}
	unitsByProduct := make(map[string]int)
	for _, s := range sales {
		if s.Cancelled {
			continue
		}
		if s.Product == "" || s.Category == "" || s.Quantity <= 0 || s.UnitPriceKopecks <= 0 {
			report.InvalidSales++
			continue
		}
		revenue := int64(s.Quantity) * s.UnitPriceKopecks
		report.RevenueKopecks += revenue
		report.UnitsSold += s.Quantity
		report.RevenueByCategory[s.Category] += revenue
		unitsByProduct[s.Product] += s.Quantity
		bestUnits := 0
		for prod, units := range unitsByProduct {
			if units > bestUnits || (units == bestUnits && prod < report.TopProduct) {
				bestUnits = units
				report.TopProduct = prod
			}
		}
	}
	return report
}
