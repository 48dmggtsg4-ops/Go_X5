package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"example.com/order-service/internal/order"
	"github.com/go-chi/chi/v5"
)

// createOrder godoc
// @Summary Создать заказ
// @Description Позиции обязательны. Название непустое, количество и цена положительные. Цена указана в копейках. Начальный статус — new.
// @Tags orders
// @Accept json
// @Produce json
// @Param input body order.CreateOrderInput true "Данные заказа"
// @Success 201 {object} order.Order
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders [post]
func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var input order.CreateOrderInput

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid request body",
			Code:  "INVALID_JSON",
		})
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "request body must contain a single JSON object",
			Code:  "INVALID_JSON",
		})
		return
	}

	value, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) writeServiceError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	var status int
	var response errorResponse

	switch {
	case errors.Is(err, order.ErrInvalidOrder):
		status = http.StatusBadRequest
		response = errorResponse{
			Error: "invalid order",
			Code:  "INVALID_ORDER",
		}

	case errors.Is(err, order.ErrInvalidStatus):
		status = http.StatusBadRequest
		response = errorResponse{
			Error: "invalid status",
			Code:  "INVALID_STATUS",
		}

	case errors.Is(err, order.ErrOrderNotFound):
		status = http.StatusNotFound
		response = errorResponse{
			Error: "order not found",
			Code:  "ORDER_NOT_FOUND",
		}

	case errors.Is(err, order.ErrInvalidStatusTransition):
		status = http.StatusConflict
		response = errorResponse{
			Error: "status transition is not allowed",
			Code:  "INVALID_STATUS_TRANSITION",
		}

	default:
		h.logger.ErrorContext(
			r.Context(),
			"request failed",
			"error", err,
		)

		status = http.StatusInternalServerError
		response = errorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		}
	}

	writeJSON(w, status, response)
}

// getOrder godoc
// @Summary Получить заказ по ID
// @Tags orders
// @Produce json
// @Param id path int true "Положительный ID заказа"
// @Success 200 {object} order.Order
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders/{id} [get]
func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "order id must be a positive integer",
			Code:  "INVALID_ID",
		})
		return
	}

	value, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, value)
}

// listOrders godoc
// @Summary Получить список заказов
// @Description Возвращает заказы в порядке создания. Если заказов нет, возвращает пустой массив.
// @Tags orders
// @Produce json
// @Success 200 {array} order.Order
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders [get]
func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	values, err := h.service.List(r.Context())
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, values)
}

// updateOrderStatus godoc
// @Summary Изменить статус заказа
// @Description Статусы: new, processing, completed, cancelled. Разрешены переходы new → processing, new → cancelled, processing → completed, processing → cancelled.
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "Положительный ID заказа"
// @Param input body order.UpdateStatusInput true "Новый статус"
// @Success 200 {object} order.Order
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders/{id}/status [patch]
func (h *Handler) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "order id must be a positive integer",
			Code:  "INVALID_ID",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var input order.UpdateStatusInput

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid request body",
			Code:  "INVALID_JSON",
		})
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "request body must contain a single JSON object",
			Code:  "INVALID_JSON",
		})
		return
	}

	value, err := h.service.UpdateStatus(r.Context(), id, input.Status)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, value)
}

// getOrderHistory godoc
// @Summary Получить историю статусов
// @Description Возвращает статусы и время изменений в хронологическом порядке.
// @Tags orders
// @Produce json
// @Param id path int true "Положительный ID заказа"
// @Success 200 {array} order.StatusChange
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders/{id}/history [get]
func (h *Handler) getOrderHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "order id must be a positive integer",
			Code:  "INVALID_ID",
		})
		return
	}

	history, err := h.service.History(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, history)
}
