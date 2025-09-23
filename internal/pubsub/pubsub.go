package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	rabbitChan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	durableProp := false
	autoDelete := false
	exclusive := false
	noWait := false

	if queueType == Durable {
		durableProp = true
	} else if queueType == Transient {
		exclusive = true
		autoDelete = true
	}

	table := amqp.Table{}
	table["x-dead-letter-exchange"] = "peril_dlx"

	newQ, err := rabbitChan.QueueDeclare(queueName, durableProp, autoDelete, exclusive, noWait, table)
	if err != nil {
		log.Fatal(err)
	}

	err = rabbitChan.QueueBind(queueName, key, exchange, noWait, nil)
	if err != nil {
		log.Fatal(err)
	}

	return rabbitChan, newQ, err
}

func processDeliveries[T any](deliveryChan <-chan amqp.Delivery, handler func(T) AckType) {
	for delivery := range deliveryChan {
		var val T
		err := json.Unmarshal(delivery.Body, &val)
		if err != nil {
			log.Fatal(err)
			continue
		}

		acktype := handler(val)

		switch acktype {
		case Ack:
			delivery.Ack(false)
			log.Println("Acked delivery")

		case NackRequeue:
			delivery.Nack(false, true)
			log.Println("Nacked delivery and asked to requeue")

		case NackDiscard:
			delivery.Nack(false, false)
			log.Println("Nacked delivery and asked to discard")

		}
	}
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	rabbitChan, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		log.Fatal(err)
		return err
	}

	autoGenerateStringConsumerName := ""
	autoAck := false
	exclusive := false
	noLocal := false
	noWait := false
	deliveryChan, err := rabbitChan.Consume(queueName, autoGenerateStringConsumerName, autoAck, exclusive, noLocal, noWait, nil)
	if err != nil {
		log.Fatal(err)
		return err
	}

	go processDeliveries(deliveryChan, handler)

	return nil
}
