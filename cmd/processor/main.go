package main

import (
	"log"
	"net/http"
	"os"

	"photo_editor/internal/broker"
	"photo_editor/internal/processor"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	metricsMux := http.NewServeMux()

	metricsMux.Handle(
		"/metrics",
		promhttp.Handler(),
	)

	go func() {
		log.Println("Processor metrics available on :9091/metrics")

		err := http.ListenAndServe(":9091", metricsMux)
		if err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://photo:photo@localhost:5672/"
	}

	apiUrl := os.Getenv("API_URL")
	if apiUrl == "" {
		apiUrl = "http://localhost:8001"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	if err := broker.DeclareQueues(ch); err != nil {
		log.Fatal(err)
	}

	err = ch.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		log.Fatal(err)
	}

	processorToken := os.Getenv("PROCESSOR_TOKEN")

	if processorToken == "" {
		log.Fatal("PROCESSOR_TOKEN environment variable is required")
	}

	consumer := processor.NewConsumer(ch, apiUrl, processorToken)
	log.Println("ImageProcessor started")

	err = consumer.Run()
	if err != nil {
		log.Fatal(err)
	}
}
