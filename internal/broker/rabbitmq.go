package broker

import (
	"context"
	"errors"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

// NewRabbitMQPublisher подключается к RabbitMQ
func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	queue, err := ch.QueueDeclare(
		"image_tasks",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQPublisher{
		conn:      conn,
		channel:   ch,
		queueName: queue.Name,
	}, nil
}

// Publish отправляет сообщение в RabbitMQ
func (p *RabbitMQPublisher) Publish(ctx context.Context, body []byte) error {
	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		"",
		p.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	if confirmation == nil {
		return errors.New("publisher confirms are not enabled")
	}
	confirmed, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for RabbitMQ confirmation: %w", err)
	}
	if !confirmed {
		return errors.New("RabbitMQ rejected the message")
	}

	return nil
}

// Close закрывает соединение с RabbitMQ
func (p *RabbitMQPublisher) Close() {
	p.channel.Close()
	p.conn.Close()
}
