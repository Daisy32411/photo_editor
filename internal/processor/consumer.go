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

	"photo_editor/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrTaskNotFound = errors.New("task not found on HTTP server")

type Consumer struct {
	channel *amqp.Channel
	apiURL  string
}

func NewConsumer(ch *amqp.Channel, apiURL string) *Consumer {
	return &Consumer{
		channel: ch,
		apiURL:  apiURL,
	}
}

// Run слушает очередь
func (c *Consumer) Run() error {
	deliveries, err := c.channel.Consume(
		"image_tasks",
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

		err = json.Unmarshal(delivery.Body, &message)
		if err != nil {
			log.Printf("Failed to decode message: %v", err)
			errRec := delivery.Reject(false)
			if errRec != nil {
				return errRec
			}
			continue
		}

		log.Printf("Received task: %s", message.TaskID)

		err = c.processMessage(message)
		if err != nil {
			if errors.Is(err, ErrTaskNotFound) {
				log.Printf(
					"Task %s not found on HTTP server, rejecting message",
					message.TaskID,
				)

				errRec := delivery.Reject(false)
				if errRec != nil {
					return errRec
				}

				continue
			}

			log.Printf(
				"Failed to process task %s: %v. Requeueing message",
				message.TaskID,
				err,
			)

			time.Sleep(3 * time.Second)

			errNack := delivery.Nack(false, true)
			if errNack != nil {
				return errNack
			}

			continue
		}

		err = delivery.Ack(false)
		if err != nil {
			return err
		}
	}

	return nil
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: task %s", ErrTaskNotFound, commit.TaskID)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("commit failed with status: %d", resp.StatusCode)
	}

	return nil
}
