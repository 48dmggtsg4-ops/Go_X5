package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"example.com/lesson9/internal/model"
	"example.com/lesson9/internal/repository"
	"example.com/lesson9/internal/service"
)

func newHandler() http.Handler {
	return New(service.New(repository.New()), slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}
func request(h http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	out := httptest.NewRecorder()
	h.ServeHTTP(out, req)
	return out
}
func assertJSON(t *testing.T, out *httptest.ResponseRecorder, status int, dst any) {
	t.Helper()
	if out.Code != status {
		t.Fatalf("status=%d want=%d body=%s", out.Code, status, out.Body.String())
	}
	if got := out.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type=%q", got)
	}
	if err := json.Unmarshal(out.Body.Bytes(), dst); err != nil {
		t.Fatal(err)
	}
}
func TestCreateReadList(t *testing.T) {
	h := newHandler()
	for _, customer := range []string{"  Алиса  ", "Боб", "Алиса"} {
		out := request(h, "POST", "/api/v1/orders", `{"customer":"`+customer+`"}`, "application/json; charset=utf-8")
		var created OrderResponse
		assertJSON(t, out, 201, &created)
		if created.ID < 1 || created.Status != "new" || created.Customer != strings.TrimSpace(customer) {
			t.Fatalf("order=%+v", created)
		}
		location := out.Header().Get("Location")
		var read OrderResponse
		assertJSON(t, request(h, "GET", location, "", ""), 200, &read)
		if read != created {
			t.Fatalf("created=%+v read=%+v", created, read)
		}
	}
	for _, test := range []struct {
		query string
		ids   []int
	}{
		{"", []int{1, 2, 3}},
		{"?limit=1&offset=1&status=new", []int{2}},
		{"?limit=2&offset=2", []int{3}},
		{"?offset=999", []int{}},
		{"?status=cancelled", []int{}},
	} {
		t.Run(test.query, func(t *testing.T) {
			var orders []OrderResponse
			out := request(h, "GET", "/api/v1/orders"+test.query, "", "")
			assertJSON(t, out, 200, &orders)
			if orders == nil {
				t.Fatal("list must be [] rather than null")
			}
			if len(orders) != len(test.ids) {
				t.Fatalf("orders=%+v want ids=%v", orders, test.ids)
			}
			for i, id := range test.ids {
				if orders[i].ID != id {
					t.Fatalf("orders=%+v", orders)
				}
			}
		})
	}
}
func TestErrors(t *testing.T) {
	cases := []struct {
		name, method, path, body, contentType string
		status                                int
		code                                  string
	}{
		{"missing customer", "POST", "/api/v1/orders", `{}`, "application/json", 400, "VALIDATION_ERROR"},
		{"blank customer", "POST", "/api/v1/orders", `{"customer":" \t "}`, "application/json", 400, "VALIDATION_ERROR"},
		{"invalid json", "POST", "/api/v1/orders", `{`, "application/json", 400, "INVALID_JSON"},
		{"unknown field", "POST", "/api/v1/orders", `{"customer":"Alice","id":8}`, "application/json", 400, "INVALID_JSON"},
		{"null", "POST", "/api/v1/orders", `null`, "application/json", 400, "INVALID_JSON"},
		{"array", "POST", "/api/v1/orders", `[]`, "application/json", 400, "INVALID_JSON"},
		{"wrong type", "POST", "/api/v1/orders", `{"customer":42}`, "application/json", 400, "INVALID_JSON"},
		{"two values", "POST", "/api/v1/orders", `{"customer":"Alice"} {}`, "application/json", 400, "INVALID_JSON"},
		{"trailing garbage", "POST", "/api/v1/orders", `{"customer":"Alice"} x`, "application/json", 400, "INVALID_JSON"},
		{"no media type", "POST", "/api/v1/orders", `{"customer":"Alice"}`, "", 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"wrong media type", "POST", "/api/v1/orders", `{"customer":"Alice"}`, "text/plain", 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"large body", "POST", "/api/v1/orders", `{"customer":"` + strings.Repeat("a", 1024) + `"}`, "application/json", 413, "BODY_TOO_LARGE"},
		{"large whitespace", "POST", "/api/v1/orders", `{"customer":"Alice"}` + strings.Repeat(" ", 1024), "application/json", 413, "BODY_TOO_LARGE"},
		{"bad id", "GET", "/api/v1/orders/abc", "", "", 400, "INVALID_ID"},
		{"zero id", "GET", "/api/v1/orders/0", "", "", 400, "INVALID_ID"},
		{"negative id", "GET", "/api/v1/orders/-1", "", "", 400, "INVALID_ID"},
		{"missing order", "GET", "/api/v1/orders/999", "", "", 404, "ORDER_NOT_FOUND"},
		{"unknown route", "GET", "/api/v1/missing", "", "", 404, "ROUTE_NOT_FOUND"},
		{"unsupported method", "DELETE", "/api/v1/orders/1", "", "", 405, "METHOD_NOT_ALLOWED"},
	}
	for _, query := range []string{"limit=0", "limit=101", "offset=-1", "limit=", "offset=x", "limit=2&limit=3", "status=unknown", "status=", "page=1", "offset=9999999999999999999999999999", "limit=%ZZ", "limit=%2B1"} {
		cases = append(cases, struct {
			name, method, path, body, contentType string
			status                                int
			code                                  string
		}{"query " + query, "GET", "/api/v1/orders?" + query, "", "", 400, "INVALID_QUERY"})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var out ErrorResponse
			h := newHandler()
			res := request(h, test.method, test.path, test.body, test.contentType)
			assertJSON(t, res, test.status, &out)
			if out.Code != test.code || out.Error == "" {
				t.Fatalf("error=%+v want=%s", out, test.code)
			}
			if res.Header().Get("Location") != "" {
				t.Fatal("error response has Location")
			}
			if test.status == 405 && res.Header().Get("Allow") != "GET, HEAD" {
				t.Fatalf("Allow=%q", res.Header().Get("Allow"))
			}
			if test.method == "POST" {
				var orders []OrderResponse
				assertJSON(t, request(h, "GET", "/api/v1/orders", "", ""), 200, &orders)
				if len(orders) != 0 {
					t.Fatal("rejected request created an order")
				}
			}
		})
	}
	var empty []OrderResponse
	assertJSON(t, request(newHandler(), "GET", "/api/v1/orders", "", ""), 200, &empty)
	if empty == nil {
		t.Fatal("empty repository must be []")
	}
}

type failedOrders struct{}

func (failedOrders) Create(string) (model.Order, error) {
	return model.Order{}, errors.New("secret database password")
}
func (failedOrders) ByID(int) (model.Order, error) {
	return model.Order{}, errors.New("secret database password")
}
func (failedOrders) List(model.ListOptions) ([]model.Order, error) {
	return nil, errors.New("secret database password")
}
func TestInternalErrorDoesNotLeak(t *testing.T) {
	h := New(failedOrders{}, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
	for _, test := range []struct{ method, path, body string }{{"POST", "/api/v1/orders", `{"customer":"Alice"}`}, {"GET", "/api/v1/orders/1", ""}, {"GET", "/api/v1/orders", ""}} {
		var out ErrorResponse
		assertJSON(t, request(h, test.method, test.path, test.body, "application/json"), 500, &out)
		if out.Code != "INTERNAL_ERROR" || out.Error != "internal server error" {
			t.Fatalf("leaked error=%+v", out)
		}
	}
}
func TestConcurrentCreate(t *testing.T) {
	h := newHandler()
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out := request(h, "POST", "/api/v1/orders", `{"customer":"Alice"}`, "application/json")
			if out.Code != 201 {
				t.Errorf("status=%d", out.Code)
			}
		}()
	}
	wg.Wait()
	var orders []OrderResponse
	assertJSON(t, request(h, "GET", "/api/v1/orders?limit=100", "", ""), 200, &orders)
	if len(orders) != 40 {
		t.Fatalf("orders=%d", len(orders))
	}
	for i, order := range orders {
		if order.ID != i+1 {
			t.Fatalf("id=%d at %d", order.ID, i)
		}
	}
}

func TestSwaggerUIAndContract(t *testing.T) {
	h := newHandler()
	for _, path := range []string{"/swagger/index.html", "/swagger/swagger-ui.css", "/swagger/swagger-ui-bundle.js"} {
		out := request(h, "GET", path, "", "")
		if out.Code != 200 || out.Body.Len() == 0 {
			t.Fatalf("%s status=%d bytes=%d", path, out.Code, out.Body.Len())
		}
	}
	var spec struct {
		Swagger  string `json:"swagger"`
		BasePath string `json:"basePath"`
		Paths    map[string]map[string]struct {
			Responses  map[string]json.RawMessage `json:"responses"`
			Parameters []struct {
				Name     string   `json:"name"`
				In       string   `json:"in"`
				Required bool     `json:"required"`
				Minimum  *float64 `json:"minimum"`
				Maximum  *float64 `json:"maximum"`
				Default  any      `json:"default"`
			} `json:"parameters"`
		} `json:"paths"`
		Definitions map[string]struct {
			Required   []string                   `json:"required"`
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"definitions"`
	}
	out := request(h, "GET", "/swagger/doc.json", "", "")
	if out.Code != 200 {
		t.Fatalf("spec status=%d", out.Code)
	}
	if err := json.Unmarshal(out.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Swagger != "2.0" || spec.BasePath != "/api/v1" {
		t.Fatalf("swagger=%s basePath=%s", spec.Swagger, spec.BasePath)
	}
	for _, test := range []struct {
		path, method string
		statuses     []string
	}{
		{"/status", "get", []string{"200", "405"}},
		{"/orders", "post", []string{"201", "400", "405", "413", "415", "500"}},
		{"/orders", "get", []string{"200", "400", "405", "500"}},
		{"/orders/{id}", "get", []string{"200", "400", "404", "405", "500"}},
	} {
		operation, ok := spec.Paths[test.path][test.method]
		if !ok {
			t.Fatalf("missing %s %s", test.method, test.path)
		}
		for _, status := range test.statuses {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("missing response %s %s %s", test.method, test.path, status)
			}
		}
	}
	for _, test := range []struct {
		schema string
		fields []string
	}{
		{"httpapi.CreateOrderRequest", []string{"customer"}},
		{"httpapi.OrderResponse", []string{"id", "customer", "status"}},
		{"httpapi.ErrorResponse", []string{"error", "code"}},
	} {
		schema, ok := spec.Definitions[test.schema]
		if !ok {
			t.Fatalf("missing schema %s", test.schema)
		}
		for _, field := range test.fields {
			if _, ok := schema.Properties[field]; !ok {
				t.Errorf("missing %s.%s", test.schema, field)
			}
			found := false
			for _, required := range schema.Required {
				if required == field {
					found = true
				}
			}
			if !found {
				t.Errorf("%s.%s must be required", test.schema, field)
			}
		}
	}
	params := spec.Paths["/orders"]["get"].Parameters
	if len(params) != 3 {
		t.Fatalf("list params=%d", len(params))
	}
	for _, p := range params {
		switch p.Name {
		case "limit":
			if p.Minimum == nil || *p.Minimum != 1 || p.Maximum == nil || *p.Maximum != 100 || p.Default != float64(20) {
				t.Errorf("limit constraints=%+v", p)
			}
		case "offset":
			if p.Minimum == nil || *p.Minimum != 0 || p.Default != float64(0) {
				t.Errorf("offset constraints=%+v", p)
			}
		}
	}
	// Swagger UI публикует сгенерированный документ, а не вычисляет его по маршрутам.
}
