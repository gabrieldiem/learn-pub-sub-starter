package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gabrieldiem/learn-pub-sub-starter/internal"
	"github.com/gabrieldiem/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server")
	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Println("Connection to RabbitMQ successful")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	rabbitChan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	message := routing.PlayingState{
		IsPaused: true,
	}

	err = internal.PublishJSON(rabbitChan, routing.ExchangePerilDirect, routing.PauseKey, message)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Sent PlayingState message")

	<-sigChan
	log.Println("Shutting down...")
}
