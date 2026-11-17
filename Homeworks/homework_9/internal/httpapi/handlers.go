package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"example.com/lesson9practice/internal/model"
	"example.com/lesson9practice/internal/repository"
	"example.com/lesson9practice/internal/service"
)

type Orders interface {
	Create(string) (model.Order, error)
	ByID(int) (model.Order, error)
	List(model.ListOptions) ([]model.Order, error)
}
type API struct {
	orders Orders
	logger *slog.Logger
}

func New(orders Orders, logger *slog.Logger) *API { return &API{orders: orders, logger: logger} }

// @Summary Статус сервиса
// @Tag status
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /status [get]
func (a *API) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, StatusResponse{Status: "ok"})
}

// @Summary Создать заказ
// @Description Тело запроса содержит один JSON-объект с полем customer. Максимальный размер тела — 1024 байта. Имя должно содержать хотя бы один непробельный символ.
// @Tags orders
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "Данные заказа"
// @Success 201 {object} OrderResponse
// @Header 201 {string} Location "Путь к созданному заказу"
// @Failure 400 {object} ErrorResponse "INVALID_JSON или VALIDATION_ERROR"
// @Failure 413 {object} ErrorResponse "BODY_TOO_LARGE"
// @Failure 415 {object} ErrorResponse "UNSUPPORTED_MEDIA_TYPE"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Router /orders [post]
func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	// Указатель позволяет отличить null от объекта. Теги DTO сами не валидируют запрос.
	var in *CreateOrderRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, err.status, err.message, err.code)
		return
	}
	if in == nil {
		writeError(w, 400, "body must be a JSON object", "INVALID_JSON")
		return
	}
	order, err := a.orders.Create(in.Customer)
	if errors.Is(err, service.ErrEmptyCustomer) {
		writeError(w, 400, "customer is required", "VALIDATION_ERROR")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/orders/"+strconv.Itoa(order.ID))
	writeJSON(w, 201, orderResponse(order))
}

// @Summary Получить заказ по номеру
// @Tags orders
// @Produce json
// @Param id path int true "Номер заказа" minimum(1)
// @Success 200 {object} OrderResponse
// @Failure 400 {object} ErrorResponse "INVALID_ID"
// @Failure 404 {object} ErrorResponse "ORDER_NOT_FOUND"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Router /orders/{id} [get]
func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, 400, "id must be a positive integer", "INVALID_ID")
		return
	}
	order, err := a.orders.ByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, 404, "order not found", "ORDER_NOT_FOUND")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, orderResponse(order))
}

func (a *API) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.ErrorContext(r.Context(), "order operation failed", "error", err)
	writeError(w, 500, "internal server error", "INTERNAL_ERROR")
}

// @Summary Список заказов
// @Description Заказы отсортированы по ID по возрастанию. Пустая страница возвращается как []. Параметры принимают только цифры 0–9. Повторные и неизвестные параметры запрещены.
// @Tags orders
// @Produce json
// @Param limit query int false "Максимальное количество заказов" minimum(1) maximum(100) default(20)
// @Param offset query int false "Количество заказов, которые нужно пропустить" minimum(0) default(0)
// @Success 200 {array} OrderResponse
// @Failure 400 {object} ErrorResponse "INVALID_QUERY"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Router /orders [get]
func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	options, ok := parseListOptions(r.URL.RawQuery)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid query parameters", "INVALID_QUERY")
		return
	}

	orders, err := a.orders.List(options)
	if err != nil {
		a.internalError(w, r, err)
		return
	}

	response := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, orderResponse(order))
	}

	writeJSON(w, http.StatusOK, response)
}

func parseListOptions(rawQuery string) (model.ListOptions, bool) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return model.ListOptions{}, false
	}

	options := model.ListOptions{Limit: 20, Offset: 0}

	for name, items := range values {
		if name != "limit" && name != "offset" {
			return model.ListOptions{}, false
		}
		if len(items) != 1 || items[0] == "" {
			return model.ListOptions{}, false
		}

		value := items[0]
		for _, digit := range value {
			if digit < '0' || digit > '9' {
				return model.ListOptions{}, false
			}
		}

		number, err := strconv.Atoi(value)
		if err != nil {
			return model.ListOptions{}, false
		}

		if name == "offset" {
			options.Offset = number
			continue
		}

		if number < 1 || number > 100 {
			return model.ListOptions{}, false
		}
		options.Limit = number
	}

	return options, true
}
