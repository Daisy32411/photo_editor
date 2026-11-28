package broker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	mu      sync.Mutex
	url     string
	conn    *amqp.Connection
	channel *amqp.Channel
	closed  bool
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	publisher := &RabbitMQPublisher{
		url: url,
	}

	if err := publisher.connect(); err != nil {
		return nil, err
	}

	return publisher, nil
}

func (p *RabbitMQPublisher) connect() error {
	p.disconnect()

	conn, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("create RabbitMQ channel: %w", err)
	}

	if err := DeclareQueues(channel); err != nil {
		conn.Close()
		return err
	}

	if err := channel.Confirm(false); err != nil {
		conn.Close()
		return fmt.Errorf("enable publisher confirms: %w", err)
	}

	p.conn = conn
	p.channel = channel

	log.Println("RabbitMQ publisher connected")

	return nil
}

func (p *RabbitMQPublisher) disconnect() {
	if p.conn != nil {
		p.conn.Close()
	}

	p.channel = nil
	p.conn = nil
}

func (p *RabbitMQPublisher) Publish(
	ctx context.Context,
	body []byte,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("RabbitMQ publisher is closed")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if p.conn == nil ||
		p.conn.IsClosed() ||
		p.channel == nil ||
		p.channel.IsClosed() {

		log.Println("RabbitMQ connection unavailable, reconnecting")

		if err := p.connect(); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		"",
		TaskQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)

	if err != nil {
		p.disconnect()
		return fmt.Errorf("publish message: %w", err)
	}

	if confirmation == nil {
		p.disconnect()
		return errors.New("publisher confirmation is unavailable")
	}

	acknowledged, err := confirmation.WaitContext(ctx)
	if err != nil {
		p.disconnect()
		return fmt.Errorf("wait for publisher confirmation: %w", err)
	}

	if !acknowledged {
		p.disconnect()
		return errors.New("RabbitMQ did not acknowledge message")
	}

	return nil
}

func (p *RabbitMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	p.disconnect()
}
