package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/middleware"
	"github.com/polymatx/goframe/pkg/mqtt"
	"github.com/sirupsen/logrus"
)

type Message struct {
	Topic     string    `json:"topic"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	ctx := context.Background()

	// Register MQTT
	mqtt.RegisterMqtt(
		"main",
		"tcp://localhost:1883",
		"goframe_client",
		"",
		"",
	)

	if err := mqtt.Initialize(ctx); err != nil {
		panic(err)
	}

	// Subscribe to topics
	go subscribe()

	// Create app
	a := app.New(&app.Config{
		Name: "mqtt-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	api := a.Group("/api/v1")
	api.POST("/publish", publishMessage)
	api.GET("/health", healthCheck)

	fmt.Println("MQTT example running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /api/v1/publish")
	fmt.Println("  GET  /api/v1/health")

	a.StartWithGracefulShutdown()
}

func publishMessage(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var msg Message
	if err := ctx.Bind(&msg); err != nil {
		ctx.JSONError(400, err)
		return
	}

	msg.Timestamp = time.Now()
	data, _ := json.Marshal(msg)

	client, _ := mqtt.GetMqttConnection("main")
	if err := client.Publish(r.Context(), msg.Topic, data); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, map[string]interface{}{
		"message":   "published",
		"topic":     msg.Topic,
		"timestamp": msg.Timestamp,
	})
}

func subscribe() {
	time.Sleep(2 * time.Second) // Wait for connection

	client, _ := mqtt.GetMqttConnection("main")

	callback := func(topic string, payload []byte) error {
		var msg Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			logrus.Errorf("Failed to unmarshal: %v", err)
			return err
		}

		logrus.WithFields(logrus.Fields{
			"topic":   topic,
			"payload": msg.Payload,
			"time":    msg.Timestamp,
		}).Info("Received message")

		return nil
	}

	topics := []string{"sensors/#", "devices/#", "alerts/#"}
	for _, topic := range topics {
		if err := client.Subscribe(context.Background(), topic, callback); err != nil {
			logrus.Errorf("Failed to subscribe to %s: %v", topic, err)
		} else {
			logrus.Infof("Subscribed to topic: %s", topic)
		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	ctx.JSON(200, map[string]string{"status": "ok"})
}
