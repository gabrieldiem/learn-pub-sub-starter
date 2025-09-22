package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gabrieldiem/learn-pub-sub-starter/internal/gamelogic"
	"github.com/gabrieldiem/learn-pub-sub-starter/internal/pubsub"
	"github.com/gabrieldiem/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.Paused = ps.IsPaused
		fmt.Println("Game pause toggled from server")
	}
}

func connectExchange(conn *amqp.Connection, username string) (*amqp.Channel, amqp.Queue, error) {
	exchange := routing.ExchangePerilDirect
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	routingKey := routing.PauseKey
	var queueType pubsub.SimpleQueueType = pubsub.Transient
	return pubsub.DeclareAndBind(conn, exchange, queueName, routingKey, queueType)
}

func processInput(rabbitChan *amqp.Channel, input []string, gameState *gamelogic.GameState) bool {
	action := input[0]

	switch action {
	case "spawn":
		err := gameState.CommandSpawn(input)
		if err != nil {
			log.Println(err)
		}

	case "move":
		_, err := gameState.CommandMove(input)
		if err != nil {
			log.Println(err)
		}

	case "status":
		gameState.CommandStatus()

	case "help":
		gamelogic.PrintClientHelp()

	case "spam":
		log.Println("Spamming not allowed yet!")

	case "quit":
		gamelogic.PrintQuit()
		return true

	default:
		log.Println("Did not understand")
	}

	return false
}

func gameloop(conn *amqp.Connection) bool {
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
		return true
	}

	rabbitChan, _, err := connectExchange(conn, username)
	if err != nil {
		log.Fatal(err)
		return true
	}

	gameState := gamelogic.NewGameState(username)
	queueName := fmt.Sprintf("pause.%s", username)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.Transient, handlerPause(gameState))

	var input []string
	for len(input) == 0 {
		input = gamelogic.GetInput()

		if len(input) != 0 {
			shouldExit := processInput(rabbitChan, input, gameState)
			if shouldExit {
				return true
			}
			input = []string{}
		}
	}
	return false
}

func main() {
	fmt.Println("Starting Peril client...")
	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Println("Connection to RabbitMQ successful")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	shouldExit := gameloop(conn)
	log.Println("Finished loop")

	if shouldExit {
		log.Println("Exiting via loop exit")
		return
	}

	<-sigChan
	log.Println("Shutting down...")
}
