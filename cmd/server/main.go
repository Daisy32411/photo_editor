package main

import (
	"log"
	"net/http"
	"os"
	"photo_editor/internal/broker"
	"photo_editor/internal/handlers"
	"photo_editor/internal/service"
	"photo_editor/internal/storage"

	_ "photo_editor/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @version 3.0
// @description Asynchronous image processing API with RabbitMQ and session-based authentication.
// @description Supported filters: Negative, Blur, Sharpen, and FlipX.
// @host localhost:8001
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your session token in the format: Bearer <token>
func main() {
	st := storage.NewMemoryStorage()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://photo:photo@localhost:5672/"
	}

	publisher, err := broker.NewRabbitMQPublisher(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer publisher.Close()

	svc := service.NewTaskService(st, publisher)
	hs := handlers.NewTaskHandler(svc)

	authStore := storage.NewAuthMemoryStorage()
	authSvc := service.NewAuthService(authStore)
	authHandler := handlers.NewAuthHandler(authSvc)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /commit", hs.CommitResult)

	mux.HandleFunc(
		"POST /task",
		handlers.AuthMiddleware(authSvc, hs.CreateTask),
	)
	mux.HandleFunc(
		"GET /status/{id}",
		handlers.AuthMiddleware(authSvc, hs.GetStatus),
	)
	mux.HandleFunc(
		"GET /result/{id}",
		handlers.AuthMiddleware(authSvc, hs.GetResult),
	)
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	log.Println("Server started on :8001")

	err = http.ListenAndServe(":8001", mux)

	if err != nil {
		log.Fatal(err)
	}
}
