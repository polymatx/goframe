package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/middleware"
	"github.com/polymatx/goframe/pkg/rabbit"
	"github.com/sirupsen/logrus"
)

type Task struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Payload string    `json:"payload"`
	Created time.Time `json:"created"`
}

func main() {
	ctx := context.Background()

	// Register RabbitMQ
	rabbit.RegisterRabbitMq(
		"main",
		"localhost",
		5672,
		"goframe",
		"goframe",
		"/",
	)

	rabbit.Initialize(ctx)

	// Start consumer
	go startConsumer()

	// Create app
	a := app.New(&app.Config{
		Name: "rabbitmq-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	api := a.Group("/api/v1")
	api.POST("/tasks", publishTask)
	api.GET("/health", healthCheck)

	fmt.Println("RabbitMQ example running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /api/v1/tasks")
	fmt.Println("  GET  /api/v1/health")

	a.StartWithGracefulShutdown()
}

func publishTask(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var req struct {
		Type    string `json:"type"`
		Payload string `json:"payload"`
	}

	if err := ctx.Bind(&req); err != nil {
		ctx.JSONError(400, err)
		return
	}

	task := Task{
		ID:      fmt.Sprintf("task_%d", time.Now().Unix()),
		Type:    req.Type,
		Payload: req.Payload,
		Created: time.Now(),
	}

	data, _ := json.Marshal(task)

	conn, _ := rabbit.GetConnection("main")
	if err := conn.Publish(r.Context(), "tasks_queue", data); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, map[string]interface{}{
		"message": "task published",
		"task":    task,
	})
}

func startConsumer() {
	conn, _ := rabbit.GetConnection("main")

	callback := func(body []byte) error {
		var task Task
		if err := json.Unmarshal(body, &task); err != nil {
			return err
		}

		logrus.WithFields(logrus.Fields{
			"id":      task.ID,
			"type":    task.Type,
			"payload": task.Payload,
		}).Info("Processing task")

		// Simulate work
		time.Sleep(2 * time.Second)

		logrus.Infof("Task %s completed", task.ID)
		return nil
	}

	logrus.Info("Starting RabbitMQ consumer...")
	if err := conn.Consume(context.Background(), "tasks_queue", callback); err != nil {
		logrus.Errorf("Consumer error: %v", err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	ctx.JSON(200, map[string]string{"status": "ok"})
}
