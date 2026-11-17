package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"example.com/lesson9/internal/model"
	"example.com/lesson9/internal/repository"
	"example.com/lesson9/internal/service"
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

// status godoc
// @Summary Статус учебного сервиса
// @Tags status
// @Produce json
// @Success 200 {object} StatusResponse
// @Failure 405 {object} ErrorResponse "METHOD_NOT_ALLOWED"
// @Router /status [get]
func (a *API) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, StatusResponse{Status: "ok"})
}

// createOrder godoc
// @Summary Создать заказ
// @Description Имя клиента обрезается по краям и должно содержать непробельные символы. Тело: один JSON-объект, только customer, не более 1024 байт. Повторные заказы клиента разрешены. Сервер назначает id и status=new. Неизвестные поля и null запрещены.
// @Tags orders
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "Данные заказа"
// @Success 201 {object} OrderResponse
// @Header 201 {string} Location "Относительный URI созданного заказа, например /api/v1/orders/1"
// @Failure 400 {object} ErrorResponse "INVALID_JSON или VALIDATION_ERROR"
// @Failure 413 {object} ErrorResponse "BODY_TOO_LARGE"
// @Failure 415 {object} ErrorResponse "UNSUPPORTED_MEDIA_TYPE"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Failure 405 {object} ErrorResponse "METHOD_NOT_ALLOWED"
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

// getOrder godoc
// @Summary Получить заказ по номеру
// @Tags orders
// @Produce json
// @Param id path int true "Положительный номер заказа" minimum(1)
// @Success 200 {object} OrderResponse
// @Failure 400 {object} ErrorResponse "INVALID_ID"
// @Failure 404 {object} ErrorResponse "ORDER_NOT_FOUND"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Failure 405 {object} ErrorResponse "METHOD_NOT_ALLOWED"
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

// listOrders godoc
// @Summary Список заказов
// @Description Сортировка id по возрастанию. Фильтр применяется до пагинации. Пустая страница: []. Доступны статусы new и cancelled, но это демо создаёт только new, поэтому фильтр cancelled возвращает []. Неизвестные и повторные query-параметры запрещены. Числа передаются десятичными цифрами.
// @Tags orders
// @Produce json
// @Param limit query int false "Размер страницы" minimum(1) maximum(100) default(20)
// @Param offset query int false "Смещение" minimum(0) default(0)
// @Param status query string false "Статус заказа" Enums(new,cancelled)
// @Success 200 {array} OrderResponse
// @Failure 400 {object} ErrorResponse "INVALID_QUERY"
// @Failure 500 {object} ErrorResponse "INTERNAL_ERROR"
// @Failure 405 {object} ErrorResponse "METHOD_NOT_ALLOWED"
// @Router /orders [get]
func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeError(w, 400, "invalid query string", "INVALID_QUERY")
		return
	}
	options, ok := parseListOptions(values)
	if !ok {
		writeError(w, 400, "use limit 1..100, offset >= 0 and status new or cancelled; parameters must be unique", "INVALID_QUERY")
		return
	}
	orders, err := a.orders.List(options)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	// make гарантирует [], даже если хранилище вернуло nil.
	out := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		out = append(out, orderResponse(order))
	}
	writeJSON(w, 200, out)
}
func parseListOptions(values url.Values) (model.ListOptions, bool) {
	options := model.ListOptions{Limit: 20}
	for key, items := range values {
		if len(items) != 1 {
			return options, false
		}
		value := items[0]
		switch key {
		case "limit", "offset":
			if value == "" {
				return options, false
			}
			for _, char := range value {
				if char < '0' || char > '9' {
					return options, false
				}
			}
			number, err := strconv.Atoi(value)
			if err != nil {
				return options, false
			}
			if key == "limit" {
				if number < 1 || number > 100 {
					return options, false
				}
				options.Limit = number
			} else {
				options.Offset = number
			}
		case "status":
			if value != "new" && value != "cancelled" {
				return options, false
			}
			options.Status = value
		default:
			return options, false
		}
	}
	return options, true
}
func (a *API) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.ErrorContext(r.Context(), "order operation failed", "error", err)
	writeError(w, 500, "internal server error", "INTERNAL_ERROR")
}
