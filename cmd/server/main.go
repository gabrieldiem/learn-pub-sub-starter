package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gabrieldiem/learn-pub-sub-starter/internal"
	"github.com/gabrieldiem/learn-pub-sub-starter/internal/gamelogic"
	"github.com/gabrieldiem/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func processInput(rabbitChan *amqp.Channel, input []string) bool {
	for _, action := range input {

		var message routing.PlayingState
		shouldSend := false

		if action == "pause" {
			message = routing.PlayingState{
				IsPaused: true,
			}
			shouldSend = true
			log.Println("Sent Pause message")

		} else if action == "resume" {
			message = routing.PlayingState{
				IsPaused: false,
			}
			shouldSend = true
			log.Println("Sent Resume message")

		} else if action == "quit" {
			log.Println("Exiting")
			return true

		} else {
			log.Println("Did not understand")
		}

		if shouldSend {
			err := internal.PublishJSON(rabbitChan, routing.ExchangePerilDirect, routing.PauseKey, message)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	return false
}

func startLoop(rabbitChan *amqp.Channel) bool {

	var input []string
	for len(input) == 0 {
		input = gamelogic.GetInput()

		if len(input) != 0 {
			shouldExit := processInput(rabbitChan, input)
			if shouldExit {
				return true
			}
			input = []string{}
		}
	}
	return false
}

func connectExchange(conn *amqp.Connection) (*amqp.Channel, amqp.Queue, error) {
	exchange := routing.ExchangePerilTopic
	queueName := routing.GameLogSlug
	routingKey := fmt.Sprintf("%s.*", routing.GameLogSlug)
	var queueType internal.SimpleQueueType = internal.Durable
	return internal.DeclareAndBind(conn, exchange, queueName, routingKey, queueType)
}

func main() {
	fmt.Println("Starting Peril server")
	gamelogic.PrintServerHelp()
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

	connectExchange(conn)

	shouldExit := startLoop(rabbitChan)
	log.Println("Finished loop")

	if shouldExit {
		log.Println("Exiting via loop exit")
		return
	}

	<-sigChan
	log.Println("Shutting down...")
}
