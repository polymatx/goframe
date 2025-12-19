package mqtt

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/polymatx/goframe/pkg/safe"
	"github.com/polymatx/goframe/pkg/xlog"
	"github.com/sirupsen/logrus"
)

var (
	clients          = make(map[string]*Client)
	clientLock       = &sync.RWMutex{}
	once             = &sync.Once{}
	mqttConnExpected = make([]mqttConfig, 0)
)

type mqttConfig struct {
	name     string
	broker   string
	clientID string
	username string
	password string
}

// Client wraps mqtt.Client with additional methods
type Client struct {
	client mqtt.Client
	name   string
}

// RegisterMqtt registers MQTT connection
func RegisterMqtt(name, broker, clientID, username, password string) error {
	mqttConnExpected = append(mqttConnExpected, mqttConfig{
		name:     name,
		broker:   broker,
		clientID: clientID,
		username: username,
		password: password,
	})
	return nil
}

// Initialize initializes all MQTT connections
func Initialize(ctx context.Context) error {
	var initErr error
	once.Do(func() {
		_ = safe.Try(func() error {
			for _, cfg := range mqttConnExpected {
				opts := mqtt.NewClientOptions().
					AddBroker(cfg.broker).
					SetClientID(cfg.clientID).
					SetKeepAlive(10 * time.Second).
					SetPingTimeout(5 * time.Second).
					SetAutoReconnect(true)

				if cfg.username != "" {
					opts.SetUsername(cfg.username)
				}
				if cfg.password != "" {
					opts.SetPassword(cfg.password)
				}

				mqttClient := mqtt.NewClient(opts)
				if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
					xlog.GetWithError(ctx, token.Error()).Error(token.Error())
					initErr = token.Error()
					return token.Error()
				}

				clientLock.Lock()
				clients[cfg.name] = &Client{
					client: mqttClient,
					name:   cfg.name,
				}
				clientLock.Unlock()

				logrus.Infof("successfully connected to mqtt: %s", cfg.broker)
			}
			return nil
		}, 30*time.Second)
	})
	return initErr
}

// GetMqttConnection returns MQTT client wrapper
func GetMqttConnection(name string) (*Client, error) {
	clientLock.RLock()
	defer clientLock.RUnlock()

	client, ok := clients[name]
	if !ok {
		return nil, fmt.Errorf("mqtt connection '%s' not found", name)
	}

	return client, nil
}

// MustGetMqttClient returns client or panics
func MustGetMqttClient(name string) mqtt.Client {
	clientLock.RLock()
	defer clientLock.RUnlock()
	val, ok := clients[name]
	if !ok {
		panic(fmt.Sprintf("mqtt client '%s' not found", name))
	}
	return val.client
}

// GetMqttClient returns an mqtt client or an error
func GetMqttClient(name string) (mqtt.Client, error) {
	clientLock.RLock()
	defer clientLock.RUnlock()
	val, ok := clients[name]
	if !ok {
		return nil, fmt.Errorf("mqtt client '%s' not found", name)
	}
	return val.client, nil
}

// Publish publishes a message to a topic
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	token := c.client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}

// Subscribe subscribes to a topic
func (c *Client) Subscribe(ctx context.Context, topic string, callback func(string, []byte) error) error {
	handler := func(client mqtt.Client, msg mqtt.Message) {
		if err := callback(msg.Topic(), msg.Payload()); err != nil {
			logrus.Errorf("MQTT handler error: %v", err)
		}
	}

	token := c.client.Subscribe(topic, 0, handler)
	token.Wait()
	return token.Error()
}

// Unsubscribe unsubscribes from a topic
func (c *Client) Unsubscribe(topic string) error {
	token := c.client.Unsubscribe(topic)
	token.Wait()
	return token.Error()
}

// IsConnected checks if client is connected
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// Disconnect disconnects the client
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
}

// Close closes all MQTT clients
func Close() {
	clientLock.Lock()
	defer clientLock.Unlock()

	for name, c := range clients {
		if c.client != nil && c.client.IsConnected() {
			c.client.Disconnect(250)
			logrus.Infof("Disconnected MQTT client: %s", name)
		}
	}
}
