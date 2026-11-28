package broker

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TaskQueue          = "image_tasks"
	DeadLetterQueue    = "image_tasks_dlq"
	DeadLetterExchange = "image_tasks_dlx"
	MaxRedeliveries    = 3
)

func DeclareQueues(ch *amqp.Channel) error {
	err := ch.ExchangeDeclare(
		DeadLetterExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare dead letter exchange: %w", err)
	}

	_, err = ch.QueueDeclare(
		DeadLetterQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare dead letter queue: %w", err)
	}

	err = ch.QueueBind(
		DeadLetterQueue,
		DeadLetterQueue,
		DeadLetterExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("bind dead letter queue: %w", err)
	}

	_, err = ch.QueueDeclare(
		TaskQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-queue-type":              "quorum",
			"x-delivery-limit":          int32(MaxRedeliveries),
			"x-dead-letter-exchange":    DeadLetterExchange,
			"x-dead-letter-routing-key": DeadLetterQueue,
		},
	)
	if err != nil {
		return fmt.Errorf("declare task queue: %w", err)
	}

	return nil
}
