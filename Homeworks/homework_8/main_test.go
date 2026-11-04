package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	h := newRouter(NewStore(), testLogger())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type=%q", rec.Header().Get("Content-Type"))
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("status in body=%q", body.Status)
	}
}

func TestCreateAndGetOrder(t *testing.T) {
	t.Parallel()
	h := newRouter(NewStore(), testLogger())
	want := `{"id":1,"customer":"Анна","status":"new"}`

	created := do(h, http.MethodPost, "/orders", `{"customer":" Анна "}`)
	require.Equal(t, http.StatusCreated, created.Code)
	assert.Equal(t, "application/json", created.Header().Get("Content-Type"))
	assert.Equal(t, "/orders/1", created.Header().Get("Location"))
	assert.JSONEq(t, want, created.Body.String())

	got := do(h, http.MethodGet, "/orders/1", "")
	require.Equal(t, http.StatusOK, got.Code)
	assert.Equal(t, "application/json", got.Header().Get("Content-Type"))
	assert.JSONEq(t, want, got.Body.String())

	second := do(h, http.MethodPost, "/orders", `{"customer":"Иван"}`)
	require.Equal(t, http.StatusCreated, second.Code)
	assert.Equal(t, "/orders/2", second.Header().Get("Location"))
}

func TestOrderErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"unknown order", http.MethodGet, "/orders/1", "", http.StatusNotFound},
		{"non-numeric id", http.MethodGet, "/orders/abc", "", http.StatusBadRequest},
		{"zero id", http.MethodGet, "/orders/0", "", http.StatusBadRequest},
		{"negative id", http.MethodGet, "/orders/-1", "", http.StatusBadRequest},
		{"too large id", http.MethodGet, "/orders/99999999999999999999", "", http.StatusBadRequest},
		{"empty body", http.MethodPost, "/orders", "", http.StatusBadRequest},
		{"broken json", http.MethodPost, "/orders", `{"customer":`, http.StatusBadRequest},
		{"wrong customer type", http.MethodPost, "/orders", `{"customer":123}`, http.StatusBadRequest},
		{"blank customer", http.MethodPost, "/orders", `{"customer":"  "}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := NewStore()

			rec := do(newRouter(store, testLogger()), tt.method, tt.path, tt.body)

			assert.Equal(t, tt.want, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.NotEmpty(t, body["error"])
			_, created := store.ByID(1)
			assert.False(t, created)
		})
	}
}

func TestRecovery(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	// nil-хранилище паникует в store.ByID до записи ответа.
	rec := do(newRouter(nil, logger), http.MethodGet, "/orders/1", "")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"внутренняя ошибка сервера"}`, rec.Body.String())
	id := rec.Header().Get("X-Request-ID")
	require.NotEmpty(t, id)

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, 2)
	var entry struct {
		RequestID string `json:"request_id"`
		Status    int    `json:"status"`
	}
	for _, line := range lines {
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		assert.Equal(t, id, entry.RequestID)
	}
	assert.Equal(t, http.StatusInternalServerError, entry.Status)
}
