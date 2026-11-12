package service_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"example.com/order-service/internal/order"
	"example.com/order-service/internal/service"

	"github.com/stretchr/testify/require"
)

type repositoryStub struct {
	createFn  func(context.Context, order.Order) (order.Order, error)
	getByIDFn func(context.Context, int64) (order.Order, error)
	listFn    func(context.Context) ([]order.Order, error)
	updateFn  func(context.Context, order.Order) (order.Order, error)
}

func (r *repositoryStub) Create(
	ctx context.Context,
	value order.Order,
) (order.Order, error) {
	return r.createFn(ctx, value)
}

func (r *repositoryStub) GetByID(
	ctx context.Context,
	id int64,
) (order.Order, error) {
	return r.getByIDFn(ctx, id)
}

func (r *repositoryStub) List(
	ctx context.Context,
) ([]order.Order, error) {
	return r.listFn(ctx)
}

func (r *repositoryStub) Update(
	ctx context.Context,
	value order.Order,
) (order.Order, error) {
	return r.updateFn(ctx, value)
}

func TestOrderService_Create(t *testing.T) {
	tests := []struct {
		name      string
		items     []order.Item
		wantTotal int64
		wantErr   error
	}{
		{
			name: "successful creation",
			items: []order.Item{
				{Name: "Keyboard", Quantity: 2, Price: 500000},
			},
			wantTotal: 1000000,
		},
		{
			name: "total for several items",
			items: []order.Item{
				{Name: "Keyboard", Quantity: 2, Price: 500000},
				{Name: "Mouse", Quantity: 3, Price: 100000},
			},
			wantTotal: 1300000,
		},
		{
			name:    "no items",
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "empty name",
			items: []order.Item{
				{Name: "", Quantity: 1, Price: 100},
			},
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "whitespace name",
			items: []order.Item{
				{Name: "   ", Quantity: 1, Price: 100},
			},
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "zero quantity",
			items: []order.Item{
				{Name: "Mouse", Quantity: 0, Price: 100},
			},
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "negative quantity",
			items: []order.Item{
				{Name: "Mouse", Quantity: -1, Price: 100},
			},
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "zero price",
			items: []order.Item{
				{Name: "Mouse", Quantity: 1, Price: 0},
			},
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "negative price",
			items: []order.Item{
				{Name: "Mouse", Quantity: 1, Price: -100},
			},
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "item cost overflow",
			items: []order.Item{
				{Name: "Mouse", Quantity: 2, Price: math.MaxInt64},
			},
			wantErr: order.ErrInvalidOrder,
		},
		{
			name: "total cost overflow",
			items: []order.Item{
				{Name: "Keyboard", Quantity: 1, Price: math.MaxInt64},
				{Name: "Mouse", Quantity: 1, Price: 1},
			},
			wantErr: order.ErrInvalidOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createCalled := false

			repository := &repositoryStub{
				createFn: func(
					_ context.Context,
					value order.Order,
				) (order.Order, error) {
					createCalled = true
					value.ID = 1
					return value, nil
				},
			}

			s := service.NewOrderService(repository)

			got, err := s.Create(
				context.Background(),
				order.CreateOrderInput{Items: tt.items},
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.False(t, createCalled)
				return
			}

			require.NoError(t, err)
			require.True(t, createCalled)
			require.Equal(t, int64(1), got.ID)
			require.Equal(t, tt.items, got.Items)
			require.Equal(t, tt.wantTotal, got.TotalPrice)
			require.Equal(t, order.StatusNew, got.Status)
			require.False(t, got.CreatedAt.IsZero())

			require.Len(t, got.History, 1)
			require.Equal(t, order.StatusNew, got.History[0].Status)
			require.Equal(t, got.CreatedAt, got.History[0].ChangedAt)
		})
	}
}

func TestOrderService_GetByID(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	tests := []struct {
		name    string
		id      int64
		stored  order.Order
		repoErr error
		wantErr error
	}{
		{
			name: "existing order",
			id:   42,
			stored: order.Order{
				ID:     42,
				Status: order.StatusNew,
			},
		},
		{
			name:    "missing order",
			id:      99,
			repoErr: order.ErrOrderNotFound,
			wantErr: order.ErrOrderNotFound,
		},
		{
			name:    "repository error",
			id:      42,
			repoErr: storageErr,
			wantErr: storageErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &repositoryStub{
				getByIDFn: func(
					_ context.Context,
					id int64,
				) (order.Order, error) {
					require.Equal(t, tt.id, id)
					return tt.stored, tt.repoErr
				},
			}

			s := service.NewOrderService(repository)

			got, err := s.GetByID(context.Background(), tt.id)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.stored, got)
		})
	}
}

func TestOrderService_List(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	tests := []struct {
		name    string
		stored  []order.Order
		repoErr error
		wantErr error
	}{
		{
			name:   "empty list",
			stored: []order.Order{},
		},
		{
			name: "several orders",
			stored: []order.Order{
				{ID: 1, Status: order.StatusNew},
				{ID: 2, Status: order.StatusProcessing},
			},
		},
		{
			name:    "repository error",
			repoErr: storageErr,
			wantErr: storageErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &repositoryStub{
				listFn: func(
					_ context.Context,
				) ([]order.Order, error) {
					return tt.stored, tt.repoErr
				},
			}

			s := service.NewOrderService(repository)

			got, err := s.List(context.Background())

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.stored, got)
		})
	}
}

func TestOrderService_UpdateStatus(t *testing.T) {
	tests := []struct {
		name    string
		from    order.Status
		to      order.Status
		wantErr error
	}{
		{
			name: "new to processing",
			from: order.StatusNew,
			to:   order.StatusProcessing,
		},
		{
			name: "new to cancelled",
			from: order.StatusNew,
			to:   order.StatusCancelled,
		},
		{
			name: "processing to completed",
			from: order.StatusProcessing,
			to:   order.StatusCompleted,
		},
		{
			name: "processing to cancelled",
			from: order.StatusProcessing,
			to:   order.StatusCancelled,
		},
		{
			name:    "new to new",
			from:    order.StatusNew,
			to:      order.StatusNew,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "new to completed",
			from:    order.StatusNew,
			to:      order.StatusCompleted,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "processing to new",
			from:    order.StatusProcessing,
			to:      order.StatusNew,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "processing to processing",
			from:    order.StatusProcessing,
			to:      order.StatusProcessing,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "completed to new",
			from:    order.StatusCompleted,
			to:      order.StatusNew,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "completed to processing",
			from:    order.StatusCompleted,
			to:      order.StatusProcessing,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "completed to completed",
			from:    order.StatusCompleted,
			to:      order.StatusCompleted,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "completed to cancelled",
			from:    order.StatusCompleted,
			to:      order.StatusCancelled,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "cancelled to new",
			from:    order.StatusCancelled,
			to:      order.StatusNew,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "cancelled to processing",
			from:    order.StatusCancelled,
			to:      order.StatusProcessing,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "cancelled to completed",
			from:    order.StatusCancelled,
			to:      order.StatusCompleted,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "cancelled to cancelled",
			from:    order.StatusCancelled,
			to:      order.StatusCancelled,
			wantErr: order.ErrInvalidStatusTransition,
		},
		{
			name:    "unknown status",
			from:    order.StatusNew,
			to:      order.Status("ready"),
			wantErr: order.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changedAt := time.Now().Add(-time.Hour)

			initial := order.Order{
				ID:         1,
				TotalPrice: 500000,
				Status:     tt.from,
				CreatedAt:  changedAt,
				Items: []order.Item{
					{Name: "Keyboard", Quantity: 1, Price: 500000},
				},
				History: []order.StatusChange{
					{Status: tt.from, ChangedAt: changedAt},
				},
			}

			updateCalled := false
			var saved order.Order

			repository := &repositoryStub{
				getByIDFn: func(
					_ context.Context,
					id int64,
				) (order.Order, error) {
					require.Equal(t, initial.ID, id)
					return initial, nil
				},
				updateFn: func(
					_ context.Context,
					value order.Order,
				) (order.Order, error) {
					updateCalled = true
					saved = value
					return value, nil
				},
			}

			s := service.NewOrderService(repository)

			got, err := s.UpdateStatus(
				context.Background(),
				initial.ID,
				tt.to,
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.False(t, updateCalled)
				return
			}

			require.NoError(t, err)
			require.True(t, updateCalled)
			require.Equal(t, saved, got)

			require.Equal(t, initial.ID, got.ID)
			require.Equal(t, initial.Items, got.Items)
			require.Equal(t, initial.TotalPrice, got.TotalPrice)
			require.Equal(t, initial.CreatedAt, got.CreatedAt)
			require.Equal(t, tt.to, got.Status)

			require.Len(t, got.History, 2)
			require.Equal(t, initial.History[0], got.History[0])
			require.Equal(t, tt.to, got.History[1].Status)
			require.False(t, got.History[1].ChangedAt.IsZero())
			require.False(t,
				got.History[1].ChangedAt.Before(got.History[0].ChangedAt),
			)
		})
	}
}

func TestOrderService_History(t *testing.T) {
	firstTime := time.Now().Add(-2 * time.Hour)
	secondTime := firstTime.Add(time.Hour)
	storageErr := errors.New("storage unavailable")

	tests := []struct {
		name    string
		history []order.StatusChange
		repoErr error
		wantErr error
	}{
		{
			name: "initial history",
			history: []order.StatusChange{
				{Status: order.StatusNew, ChangedAt: firstTime},
			},
		},
		{
			name: "chronological history",
			history: []order.StatusChange{
				{Status: order.StatusNew, ChangedAt: firstTime},
				{Status: order.StatusProcessing, ChangedAt: secondTime},
			},
		},
		{
			name:    "missing order",
			repoErr: order.ErrOrderNotFound,
			wantErr: order.ErrOrderNotFound,
		},
		{
			name:    "repository error",
			repoErr: storageErr,
			wantErr: storageErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &repositoryStub{
				getByIDFn: func(
					_ context.Context,
					id int64,
				) (order.Order, error) {
					require.Equal(t, int64(1), id)

					return order.Order{
						ID:      id,
						History: tt.history,
					}, tt.repoErr
				},
			}

			s := service.NewOrderService(repository)

			got, err := s.History(context.Background(), 1)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.history, got)

			for i := 1; i < len(got); i++ {
				require.False(t,
					got[i].ChangedAt.Before(got[i-1].ChangedAt),
				)
			}
		})
	}
}
func TestOrderService_CreateRepositoryError(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	repository := &repositoryStub{
		createFn: func(
			_ context.Context,
			_ order.Order,
		) (order.Order, error) {
			return order.Order{}, storageErr
		},
	}

	s := service.NewOrderService(repository)

	_, err := s.Create(
		context.Background(),
		order.CreateOrderInput{
			Items: []order.Item{
				{Name: "Keyboard", Quantity: 1, Price: 500000},
			},
		},
	)

	require.ErrorIs(t, err, storageErr)
}

func TestOrderService_UpdateStatusRepositoryError(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	tests := []struct {
		name      string
		getErr    error
		updateErr error
		wantErr   error
	}{
		{
			name:    "missing order",
			getErr:  order.ErrOrderNotFound,
			wantErr: order.ErrOrderNotFound,
		},
		{
			name:    "read error",
			getErr:  storageErr,
			wantErr: storageErr,
		},
		{
			name:      "update error",
			updateErr: storageErr,
			wantErr:   storageErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalled := false

			repository := &repositoryStub{
				getByIDFn: func(
					_ context.Context,
					id int64,
				) (order.Order, error) {
					require.Equal(t, int64(1), id)

					return order.Order{
						ID:     id,
						Status: order.StatusNew,
					}, tt.getErr
				},
				updateFn: func(
					_ context.Context,
					value order.Order,
				) (order.Order, error) {
					updateCalled = true
					require.Equal(t, order.StatusProcessing, value.Status)
					return order.Order{}, tt.updateErr
				},
			}

			s := service.NewOrderService(repository)

			_, err := s.UpdateStatus(
				context.Background(),
				1,
				order.StatusProcessing,
			)

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.getErr == nil, updateCalled)
		})
	}
}
