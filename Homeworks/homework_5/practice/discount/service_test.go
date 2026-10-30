package discount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDiscount(t *testing.T) {
	tests := []struct {
		price int
		want  int
	}{
		{100, 90},
		{50, 40},
		{10, 0},
		{0, -10},
		{-5, -15},
	}

	for _, tt := range tests {
		got := Discount(tt.price)
		// if got != tt.want {
		// 	t.Errorf("Discount(%d) = %d; want %d", tt.price, got, tt.want)
		// }
		assert.Equal(t, tt.want, got)
	}

}

// func TestDiscount50(t *testing.T) {
// 	got := Discount(50)
// 	want := 40
// 	if got != want {
// 		t.Errorf("Discount(50) = %d; want %d", got, want)
// 	}
// }

// func TestDiscount10(t *testing.T) {
// 	got := Discount(10)
// 	want := 0
// 	if got != want {
// 		t.Errorf("Discount(10) = %d; want %d", got, want)
// 	}
// }

// func TestDiscount0(t *testing.T) {
// 	got := Discount(0)
// 	want := -10
// 	if got != want {
// 		t.Errorf("Discount(0) = %d; want %d", got, want)
// 	}
// }

// func TestDiscountNegative(t *testing.T) {
// 	got := Discount(-5)
// 	want := -15
// 	if got != want {
// 		t.Errorf("Discount(-5) = %d; want %d", got, want)
// 	}
// }
