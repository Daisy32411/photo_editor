package service

import "context"

// TaskPublisher описывает отправку сообщений в очередь
type TaskPublisher interface {
	Publish(ctx context.Context, body []byte) error
}
