package practice

type Notifier interface {
	Send(message string) error
}

type BaseNotifier struct {
	Channel string
}

func (b BaseNotifier) Label(message string) string {
	return "[" + b.Channel + "] " + message
}

type EmailNotifier struct {
	BaseNotifier
	Address  string
	Messages []string
}

type PushNotifier struct {
	BaseNotifier
	DeviceID string
	Messages []string
}

func (n *EmailNotifier) Send(message string) error {
	// TODO: добавьте форматированное сообщение в Messages.
	return nil
}

func (n *PushNotifier) Send(message string) error {
	// TODO: добавьте форматированное сообщение в Messages.
	return nil
}

func Dispatch(all []Notifier, message string) error {
	// TODO: вызовите Send у каналов по порядку, верните первую ошибку.
	return nil
}

func EmailAddress(n Notifier) (string, bool) {
	// TODO: безопасно проверьте тип и указатель.
	return "", false
}

func Describe(n Notifier) string {
	// TODO: опишите email и push, остальные значения дают unknown.
	return "unknown"
}
