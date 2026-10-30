package discount

const discountAmount = 10

// Discount уменьшает цену на 10.
func Discount(price int) int {
	return price - discountAmount
}
