package main

import (
	"errors"
	"fmt"

	"lesson5/student/practice"
)

func main() {
	fmt.Println("---Случай 1: успешная рассылка---")
	email := &practice.EmailNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "email"},
		Address:      "mephi007@example.test",
	}
	push := &practice.PushNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "push"},
		DeviceID:     "device-001",
	}
	err := practice.Dispatch([]practice.Notifier{email, push}, "Заказ готов")
	if err == nil {
		fmt.Println("ошибки нет")
	} else {
		fmt.Println("ошибка:", err)
	}
	fmt.Println("история email:", email.Messages)
	fmt.Println("история push:", push.Messages)

	fmt.Println("---Случай 2: пустой текст---")
	emailBefore, pushBefore := len(email.Messages), len(push.Messages)
	err = practice.Dispatch([]practice.Notifier{email, push}, "")
	if errors.Is(err, practice.ErrEmptyMessage) {
		fmt.Println("распознано ErrEmptyMessage")
	} else {
		fmt.Println("ErrEmptyMessage не найден, ошибка:", err)
	}
	fmt.Printf("длина истории email: было %d, стало %d\n", emailBefore, len(email.Messages))
	fmt.Printf("длина истории push: было %d, стало %d\n", pushBefore, len(push.Messages))

	fmt.Println("---Случай 3: нет настройки---")
	noDevice := &practice.PushNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "push"},
	}
	err = practice.Dispatch([]practice.Notifier{noDevice}, "Заказ готов")
	var sendErr *practice.SendError
	if errors.As(err, &sendErr) {
		fmt.Println("извлечён *SendError")
		fmt.Println("Channel:", sendErr.Channel)
		fmt.Println("Reason:", sendErr.Reason)
	} else {
		fmt.Println("*SendError не найден, ошибка:", err)
	}
	fmt.Println("история push без настройки:", noDevice.Messages)

	fmt.Println("---Случай 4: сломанная реализация---")
	broken := &practice.BrokenNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "broken"},
	}
	workingPush := &practice.PushNotifier{
		BaseNotifier: practice.BaseNotifier{Channel: "push"},
		DeviceID:     "device-002",
	}
	errs := practice.SafeDispatch([]practice.Notifier{broken, workingPush}, "Заказ готов")
	fmt.Println("число ошибок:", len(errs))
	if errors.Is(errors.Join(errs...), practice.ErrNotifierPanic) {
		fmt.Println("распознано ErrNotifierPanic")
	} else {
		fmt.Println("ErrNotifierPanic не найден, ошибки:", errs)
	}
	fmt.Println("история исправного push:", workingPush.Messages)
}
