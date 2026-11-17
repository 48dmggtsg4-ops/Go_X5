package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"example.com/lesson9/internal/httpapi"
	"example.com/lesson9/internal/repository"
	"example.com/lesson9/internal/service"
)

// @title Orders API
// @version 1.0
// @description Учебный контракт сервиса заказов. Данные находятся в памяти и исчезают после перезапуска. GET-маршруты ServeMux также принимают HEAD. Ошибки под /api/v1/ имеют формат {error,code}, в том числе неизвестный маршрут (404 ROUTE_NOT_FOUND) и неподдерживаемый метод (405 METHOD_NOT_ALLOWED и заголовок Allow). Swagger UI доступен локально на /swagger/index.html.
// @BasePath /api/v1
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	api := httpapi.New(service.New(repository.New()), logger)
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second}
	logger.Info("orders API", "url", "http://127.0.0.1:8080/swagger/index.html")
	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
