package main

import (
	"log"
	"net/http"
	"photo_editor/internal/handlers"
	"photo_editor/internal/service"
	"photo_editor/internal/storage"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "photo_editor/docs"
)

// @title Photo Editor API
// @version 1.0
// @description HTTP API for asynchronous image processing tasks.
// @host localhost:8001
// @BasePath /
func main() {
	st := storage.NewMemoryStorage()

	svc := service.NewTaskService(st)

	hs := handlers.NewTaskHandler(svc)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /task", hs.CreateTask)
	mux.HandleFunc("GET /status/{id}", hs.GetStatus)
	mux.HandleFunc("GET /result/{id}", hs.GetResult)
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	log.Println("Server started on :8001")

	err := http.ListenAndServe(":8001", mux)

	if err != nil {
		log.Fatal(err)
	}
}
