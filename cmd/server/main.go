package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"photo_editor/internal/broker"
	"photo_editor/internal/handlers"
	"photo_editor/internal/service"
	"photo_editor/internal/storage"
	"time"

	_ "photo_editor/docs"

	"github.com/jackc/pgx/v5/pgxpool"
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
	postgresURL := os.Getenv("POSTGRES_URL")

	if postgresURL == "" {
		postgresURL = "postgres://photo:photo@localhost:5430/photo_editor?sslmode=disable"
	}

	dbCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := pgxpool.New(dbCtx, postgresURL)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL pool: %v", err)
	}
	defer db.Close()

	if err := db.Ping(dbCtx); err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	log.Println("Connected to PostgreSQL")

	st := storage.NewPostgresTaskStorage(db)

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

	redisAddr := os.Getenv("REDIS_ADDR")

	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer redisClient.Close()

	redisCtx, redisCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer redisCancel()

	if err := redisClient.Ping(redisCtx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis")

	userStorage := storage.NewPostgresUserStorage(db)
	sessionStorage := storage.NewRedisSessionStorage(redisClient)

	authSvc := service.NewAuthService(
		userStorage,
		sessionStorage,
	)

	authHandler := handlers.NewAuthHandler(authSvc)

	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())
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
