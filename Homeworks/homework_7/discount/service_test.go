package discount

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePercent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		percent int
		wantErr error
	}{
		{"zero", 0, nil},
		{"10", 10, nil},
		{"100", 100, nil},
		{"negative -1", -1, ErrInvalidPercent},
		{"over 101", 101, ErrInvalidPercent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePercent(tt.percent)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestApply(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		price   int
		percent int
		want    int
		wantErr error
	}{
		{"default", 50, 10, 45, nil},
		{"zero discount", 50, 0, 50, nil},
		{"100 percents discount", 50, 100, 0, nil},
		{"negative price", -50, 15, 0, ErrNegativePrice},
		{"negative percent", 50, -15, 0, ErrInvalidPercent},
		{"percent more than 100", 50, 101, 0, ErrInvalidPercent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Apply(tt.price, tt.percent)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Zero(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFinalPrice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		items   []Item
		want    int
		wantErr error
	}{
		{
			name:  "nil-list",
			items: nil,
			want:  0,
		},
		{
			name:  "empty",
			items: []Item{},
			want:  0,
		},
		{
			name:  "one item",
			items: []Item{{Price: 500, DiscountPercent: 20}},
			want:  400,
		},
		{
			name: "a few items",
			items: []Item{
				{Price: 1000, DiscountPercent: 10},
				{Price: 500, DiscountPercent: 0},
				{Price: 200, DiscountPercent: 100},
			},
			want: 1400,
		},
		{
			name: "invalide price",
			items: []Item{
				{Price: 1000, DiscountPercent: 10},
				{Price: -5, DiscountPercent: 10},
			},
			wantErr: ErrNegativePrice,
		},
		{
			name: "invalide percent",
			items: []Item{
				{Price: 1000, DiscountPercent: 10},
				{Price: 500, DiscountPercent: 150},
			},
			wantErr: ErrInvalidPercent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FinalPrice(tt.items)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Zero(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
