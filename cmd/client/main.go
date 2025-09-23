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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.Paused = ps.IsPaused
		fmt.Println("Game pause toggled from server")
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func connectToPauseQueue(conn *amqp.Connection, username string) (*amqp.Channel, amqp.Queue, error) {
	exchange := routing.ExchangePerilDirect
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	routingKey := routing.PauseKey
	var queueType pubsub.SimpleQueueType = pubsub.Transient
	return pubsub.DeclareAndBind(conn, exchange, queueName, routingKey, queueType)
}

func connectToMovesQueue(conn *amqp.Connection, username string) (*amqp.Channel, amqp.Queue, error) {
	exchange := routing.ExchangePerilTopic
	queueName := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
	routingKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
	var queueType pubsub.SimpleQueueType = pubsub.Transient
	return pubsub.DeclareAndBind(conn, exchange, queueName, routingKey, queueType)
}

func processInput(pauseChannel *amqp.Channel, movesChannel *amqp.Channel, input []string, gameState *gamelogic.GameState, username string) bool {
	action := input[0]

	switch action {
	case "spawn":
		err := gameState.CommandSpawn(input)
		if err != nil {
			log.Println(err)
		}

	case "move":
		move, err := gameState.CommandMove(input)
		if err != nil {
			log.Println(err)
		}
		pubsub.PublishJSON(movesChannel, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username), move)

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

	pauseChannel, _, err := connectToPauseQueue(conn, username)
	if err != nil {
		log.Fatal(err)
		return true
	}

	movesChannel, _, err := connectToMovesQueue(conn, username)
	if err != nil {
		log.Fatal(err)
		return true
	}

	gameState := gamelogic.NewGameState(username)
	queueName := fmt.Sprintf("pause.%s", username)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.Transient, handlerPause(gameState))

	queueName = fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
	routingKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueName, routingKey, pubsub.Transient, handlerMove(gameState))

	var input []string
	for len(input) == 0 {
		input = gamelogic.GetInput()

		if len(input) != 0 {
			shouldExit := processInput(pauseChannel, movesChannel, input, gameState, username)
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
