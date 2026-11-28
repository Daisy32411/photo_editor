package processor

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"time"

	"photo_editor/internal/broker"
	"photo_editor/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

const MaxImagePixels = 32_000_000

var ErrTaskNotFound = errors.New("task not found on HTTP server")
var ErrPermanentCommit = errors.New("permanent commit error")

type Consumer struct {
	channel        *amqp.Channel
	apiURL         string
	processorToken string
}

func NewConsumer(
	ch *amqp.Channel,
	apiURL string,
	processorToken string,
) *Consumer {
	return &Consumer{
		channel:        ch,
		apiURL:         apiURL,
		processorToken: processorToken,
	}
}

// Run слушает очередь
func (c *Consumer) Run() error {
	deliveries, err := c.channel.Consume(
		broker.TaskQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for delivery := range deliveries {
		var message models.ImageTaskMessage

		if err := json.Unmarshal(delivery.Body, &message); err != nil {
			log.Printf("Invalid message, sending to DLQ: %v", err)

			if err := delivery.Reject(false); err != nil {
				return err
			}

			continue
		}

		log.Printf("Received task: %s", message.TaskID)

		err := c.processMessage(message)
		if err != nil {
			if errors.Is(err, ErrTaskNotFound) ||
				errors.Is(err, ErrPermanentCommit) {

				log.Printf(
					"Permanent error for task %s, sending to DLQ: %v",
					message.TaskID,
					err,
				)

				if err := delivery.Reject(false); err != nil {
					return err
				}

				continue
			}

			log.Printf(
				"Temporary error for task %s: %v. Retrying",
				message.TaskID,
				err,
			)

			time.Sleep(3 * time.Second)

			if err := delivery.Reject(true); err != nil {
				return err
			}

			continue
		}

		if err := delivery.Ack(false); err != nil {
			return err
		}
	}

	return errors.New("RabbitMQ deliveries channel closed")
}

// processMessage
func (c *Consumer) processMessage(message models.ImageTaskMessage) error {
	filterName := metricFilterName(message.Filter.Name)

	filterUsageTotal.WithLabelValues(filterName).Inc()

	started := time.Now()

	result, err := processImage(message)

	processingDuration.
		WithLabelValues(filterName).
		Observe(time.Since(started).Seconds())

	if err != nil {
		tasksTotal.
			WithLabelValues(filterName, "failed").
			Inc()

		log.Printf("Task %s failed: %v", message.TaskID, err)

		commit := models.CommitRequest{
			TaskID: message.TaskID,
			Status: "failed",
			Result: "",
		}

		return c.sendCommit(commit)
	}

	tasksTotal.
		WithLabelValues(filterName, "success").
		Inc()

	commit := models.CommitRequest{
		TaskID: message.TaskID,
		Status: "ready",
		Result: result,
	}

	err = c.sendCommit(commit)
	if err != nil {
		return err
	}

	log.Printf("Task %s processed successfully", message.TaskID)

	return nil
}

// processImage обрабатывает изображение и возвращает PNG в Base64
func processImage(message models.ImageTaskMessage) (string, error) {
	imageBytes, err := base64.StdEncoding.DecodeString(message.Image)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	config, err := png.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return "", fmt.Errorf("failed to decode PNG config: %w", err)
	}

	if config.Width <= 0 || config.Height <= 0 {
		return "", errors.New("invalid image dimensions")
	}

	if config.Width > MaxImagePixels/config.Height {
		return "", fmt.Errorf("image dimensions exceed maximum allowed pixels: %dx%d", config.Width, config.Height)
	}

	img, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return "", fmt.Errorf("failed to decode PNG: %w", err)
	}

	resultImage, err := ApplyFilter(img, message.Filter)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	err = png.Encode(&buf, resultImage)
	if err != nil {
		return "", err
	}

	encodedImage := base64.StdEncoding.EncodeToString(buf.Bytes())

	return encodedImage, nil
}

// sendCommit отправляет результат выполнения задачи HTTP-серверу
func (c *Consumer) sendCommit(commit models.CommitRequest) error {
	body, err := json.Marshal(commit)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.apiURL+"/commit",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Processor-Token", c.processorToken)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf(
			"%w: task %s",
			ErrTaskNotFound,
			commit.TaskID,
		)
	}

	if resp.StatusCode >= 400 &&
		resp.StatusCode < 500 &&
		resp.StatusCode != http.StatusRequestTimeout &&
		resp.StatusCode != http.StatusTooManyRequests {

		return fmt.Errorf(
			"%w: HTTP %d",
			ErrPermanentCommit,
			resp.StatusCode,
		)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf(
			"commit failed with status: %d",
			resp.StatusCode,
		)
	}

	return nil
}
