package queue

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

// ConnectRabbitMQ establishes a connection to the RabbitMQ server.
// It returns the connection object and any error encountered during the connection process.
func ConnectRabbitMQ() (*amqp091.Connection, error) {
	// Define the RabbitMQ connection URL
	rabbitMQURL := "amqp://guest:guest@localhost:5672/"

	// Attempt to establish a connection
	conn, err := amqp091.Dial(rabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Return the established connection
	return conn, nil
}
