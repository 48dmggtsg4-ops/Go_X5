package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const maxBodyBytes = 1024

type requestIDKey struct{}

type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.wrote = true
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) Write(data []byte) (int, error) {
	rec.wrote = true
	return rec.ResponseWriter.Write(data)
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 8)
		_, _ = rand.Read(data)
		id := hex.EncodeToString(data)

		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func requestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				logger.Error("panic",
					"request_id", requestID(r.Context()),
					"panic", v,
					"stack", string(debug.Stack()),
				)
				if rec, ok := w.(*statusRecorder); ok && rec.wrote {
					return
				}
				writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.Info("request",
				"request_id", requestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", float64(time.Since(start).Microseconds())/1000,
			)
		})
	}
}

func getOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "id должен быть положительным целым числом")
			return
		}

		order, ok := store.ByID(id)
		if !ok {
			writeError(w, http.StatusNotFound, "заказ не найден")
			return
		}

		writeJSON(w, http.StatusOK, order)
	}
}

func newRouter(store *Store, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(withRequestID, accessLog(logger), recovery(logger))

	r.Get("/health", health)
	r.Post("/orders", createOrder(store))
	r.Get("/orders/{id}", getOrder(store))

	return r
}

func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createOrderRequest struct {
	Customer string `json:"customer"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "тело запроса слишком большое")
		return
	}
	writeError(w, http.StatusBadRequest, "некорректный JSON")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type должен быть application/json")
		return false
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		writeDecodeError(w, err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeDecodeError(w, err)
		return false
	}
	return true
}

func createOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createOrderRequest
		if !decodeJSON(w, r, &req) {
			return
		}

		customer := strings.TrimSpace(req.Customer)
		if customer == "" {
			writeError(w, http.StatusBadRequest, "имя клиента не должно быть пустым")
			return
		}

		order := store.Create(customer)
		w.Header().Set("Location", "/orders/"+strconv.Itoa(order.ID))
		writeJSON(w, http.StatusCreated, order)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("encode response: %v", err)
		status = http.StatusInternalServerError
		data = []byte(`{"error":"internal server error"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(data, '\n')); err != nil {
		log.Printf("write response: %v", err)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	server := &http.Server{
		Addr: "127.0.0.1:8080", Handler: newRouter(NewStore(), logger),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second, WriteTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
