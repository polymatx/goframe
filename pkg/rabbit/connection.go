package rabbit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
)

// Connection wraps RabbitMQ operations
type Connection struct {
	name string
}

// GetConnection returns RabbitMQ connection wrapper
func GetConnection(name string) (*Connection, error) {
	connRngLock.RLock()
	defer connRngLock.RUnlock()

	if _, ok := connRng[name]; !ok {
		return nil, fmt.Errorf("rabbitmq connection '%s' not found", name)
	}

	return &Connection{name: name}, nil
}

// Publish publishes a message to queue
func (c *Connection) Publish(ctx context.Context, queue string, body []byte) error {
	rngLock.Lock()
	r := rng[c.name]
	cl := r.Value.(*chnlLock)
	rng[c.name] = r.Next()
	rngLock.Unlock()

	cl.lock.Lock()
	defer cl.lock.Unlock()

	if cl.closed {
		return fmt.Errorf("channel closed")
	}

	// Declare queue
	connRngLock.RLock()
	conn := connRng[c.name].Value.(*amqp.Connection)
	connRngLock.RUnlock()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	cl.wg.Add(1)

	err = cl.chn.Publish(
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)

	if err != nil {
		cl.wg.Done()
		return err
	}

	cl.wg.Wait()
	return nil
}

// PublishJSON publishes JSON message
func (c *Connection) PublishJSON(ctx context.Context, queue string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.Publish(ctx, queue, body)
}

// Consume consumes messages from queue
func (c *Connection) Consume(ctx context.Context, queue string, handler func([]byte) error) error {
	connRngLock.RLock()
	conn := connRng[c.name].Value.(*amqp.Connection)
	connRngLock.RUnlock()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Declare queue
	q, err := ch.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			if err := handler(msg.Body); err != nil {
				_ = msg.Nack(false, true) // requeue
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

// RegisterRabbitMq is an alias for RegisterRabbit
func RegisterRabbitMq(name, host string, port int, user, password, vhost string) {
	RegisterRabbit(name, host, user, password, vhost, port)
}
