package internal

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	marshalledVal, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	mandatory := false
	immediate := false

	message := amqp.Publishing{
		ContentType: "application/json",
		Body:        marshalledVal,
	}

	return ch.PublishWithContext(context.Background(), exchange, key, mandatory, immediate, message)
}
