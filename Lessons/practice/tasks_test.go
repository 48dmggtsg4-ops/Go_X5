package practice

import (
	"reflect"
	"testing"
)

type testError struct{}

func (*testError) Error() string {
	return "сбой канала"
}

type spy struct {
	id       int
	calls    *[]int
	messages *[]string
	err      error
}

func (s spy) Send(msg string) error {
	*s.calls = append(*s.calls, s.id)
	*s.messages = append(*s.messages, msg)
	return s.err
}

func TestDispatch(t *testing.T) {
	for _, tc := range []struct {
		name        string
		count, fail int
	}{
		{"nil", 0, -1}, {"single", 1, -1}, {"all", 3, -1}, {"first_error", 3, 0}, {"middle_error", 3, 1}, {"last_error", 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []int
			var msgs []string
			var all []Notifier
			var wantErr error
			failure := &testError{}
			for i := 0; i < tc.count; i++ {
				var err error
				if i == tc.fail {
					err = failure
					wantErr = err
				}
				all = append(all, spy{i, &calls, &msgs, err})
			}
			got := Dispatch(all, "Заказ №42")
			if got != wantErr {
				t.Fatalf("error=%v, want %v", got, wantErr)
			}
			n := tc.count
			if tc.fail >= 0 {
				n = tc.fail + 1
			}
			if len(calls) != n {
				t.Fatalf("calls=%v, want %d", calls, n)
			}
			for i := 0; i < n; i++ {
				if calls[i] != i || msgs[i] != "Заказ №42" {
					t.Fatalf("calls=%v messages=%v", calls, msgs)
				}
			}
		})
	}
	t.Run("empty_slice", func(t *testing.T) {
		if err := Dispatch([]Notifier{}, "x"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestChannels(t *testing.T) {
	for _, tc := range []struct {
		name, msg string
	}{{"text", "Заказ готов"}, {"empty", ""}, {"unicode", "🛒 №42"}} {
		t.Run(tc.name, func(t *testing.T) {
			email := &EmailNotifier{BaseNotifier: BaseNotifier{Channel: "email"}, Address: "a@example.com"}
			push := &PushNotifier{BaseNotifier: BaseNotifier{Channel: "push"}, DeviceID: "phone"}
			for i := 0; i < 2; i++ {
				if err := Dispatch([]Notifier{email, push}, tc.msg); err != nil {
					t.Fatal(err)
				}
			}
			for _, check := range []struct {
				got     []string
				channel string
			}{{email.Messages, "email"}, {push.Messages, "push"}} {
				want := []string{"[" + check.channel + "] " + tc.msg, "[" + check.channel + "] " + tc.msg}
				if !reflect.DeepEqual(check.got, want) {
					t.Errorf("%s: got %q want %q", check.channel, check.got, want)
				}
			}
		})
	}
	t.Run("independent", func(t *testing.T) {
		a, b := &EmailNotifier{}, &EmailNotifier{}
		_ = a.Send("x")
		if len(a.Messages) != 1 || len(b.Messages) != 0 {
			t.Fatalf("a=%v b=%v", a.Messages, b.Messages)
		}
	})
	t.Run("promoted_label", func(t *testing.T) {
		e := EmailNotifier{BaseNotifier: BaseNotifier{Channel: "email"}}
		if e.Label("x") != "[email] x" {
			t.Fatal(e.Label("x"))
		}
	})
}

func TestEmailAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    Notifier
		want string
		ok   bool
	}{
		{"email", &EmailNotifier{Address: "a@example.com"}, "a@example.com", true},
		{"empty_address", &EmailNotifier{}, "", true},
		{"push", &PushNotifier{}, "", false}, {"nil", nil, "", false},
		{"typed_nil", (*EmailNotifier)(nil), "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := EmailAddress(tc.n)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("got %q,%v want %q,%v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	var calls []int
	var msgs []string
	for _, tc := range []struct {
		name string
		n    Notifier
		want string
	}{
		{"email", &EmailNotifier{Address: "a@example.com"}, "email: a@example.com"},
		{"push", &PushNotifier{DeviceID: "phone"}, "push: phone"},
		{"other", spy{calls: &calls, messages: &msgs}, "unknown"}, {"nil", nil, "unknown"},
		{"nil_email", (*EmailNotifier)(nil), "unknown"}, {"nil_push", (*PushNotifier)(nil), "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Describe(tc.n); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
