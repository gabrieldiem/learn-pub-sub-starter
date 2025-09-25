package main

import (
	"fmt"
	"log"

	"github.com/gabrieldiem/learn-pub-sub-starter/internal/gamelogic"
	"github.com/gabrieldiem/learn-pub-sub-starter/internal/pubsub"
	"github.com/gabrieldiem/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")

		moveOutcome := gs.HandleMove(move)
		switch moveOutcome {
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.Ack
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.NackRequeue
		}

		fmt.Println("error: unknown move outcome")
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState) func(dw gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(dw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		warOutcome, _, _ := gs.HandleWar(dw)
		switch warOutcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		}

		fmt.Println("error: unknown war outcome")
		return pubsub.NackDiscard
	}
}

// // func connectToPauseQueue(conn *amqp.Connection, username string) (*amqp.Channel, amqp.Queue, error) {
// // 	exchange := routing.ExchangePerilDirect
// // 	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
// // 	routingKey := routing.PauseKey
// // 	var queueType pubsub.SimpleQueueType = pubsub.Transient
// // 	return pubsub.DeclareAndBind(conn, exchange, queueName, routingKey, queueType)
// // }

// // func connectToMovesQueue(conn *amqp.Connection, username string) (*amqp.Channel, amqp.Queue, error) {
// // 	exchange := routing.ExchangePerilTopic
// // 	queueName := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
// // 	routingKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
// // 	var queueType pubsub.SimpleQueueType = pubsub.Transient
// // 	return pubsub.DeclareAndBind(conn, exchange, queueName, routingKey, queueType)
// // }

// // func connectToWarQueue(conn *amqp.Connection, username string) (*amqp.Channel, amqp.Queue, error) {
// // 	exchange := routing.ExchangePerilTopic
// // 	queueName := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, username)
// // 	routingKey := fmt.Sprintf("%s", routing.WarRecognitionsPrefix)
// // 	var queueType pubsub.SimpleQueueType = pubsub.Durable
// // 	return pubsub.DeclareAndBind(conn, exchange, queueName, routingKey, queueType)
// // }

// func processInput(pauseChannel *amqp.Channel, movesChannel *amqp.Channel, input []string, gameState *gamelogic.GameState, username string) bool {
// 	action := input[0]

// 	switch action {
// 	case "spawn":
// 		err := gameState.CommandSpawn(input)
// 		if err != nil {
// 			log.Println(err)
// 		}

// 	case "move":
// 		move, err := gameState.CommandMove(input)
// 		if err != nil {
// 			log.Println(err)
// 		}
// 		pubsub.PublishJSON(movesChannel, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username), move)

// 	case "status":
// 		gameState.CommandStatus()

// 	case "help":
// 		gamelogic.PrintClientHelp()

// 	case "spam":
// 		log.Println("Spamming not allowed yet!")

// 	case "quit":
// 		gamelogic.PrintQuit()
// 		return true

// 	default:
// 		log.Println("Did not understand")
// 	}

// 	return false
// }

// func gameloop(conn *amqp.Connection) bool {
// 	username, err := gamelogic.ClientWelcome()
// 	if err != nil {
// 		log.Fatal(err)
// 		return true
// 	}

// 	pauseChannel, _, err := connectToPauseQueue(conn, username)
// 	if err != nil {
// 		log.Fatal(err)
// 		return true
// 	}

// 	movesChannel, _, err := connectToMovesQueue(conn, username)
// 	if err != nil {
// 		log.Fatal(err)
// 		return true
// 	}

// 	// warChannel, _, err := connectToWarQueue(conn, username)
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// 	return true
// 	// }

// 	gameState := gamelogic.NewGameState(username)
// 	queueName := fmt.Sprintf("pause.%s", username)
// 	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.Transient, handlerPause(gameState))

// 	queueName = fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
// 	routingKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
// 	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueName, routingKey, pubsub.Transient, handlerMove(gameState, movesChannel))

// 	queueName = fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix)
// 	routingKey = fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix)
// 	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueName, routingKey, pubsub.Durable, handleWar(gameState))

// 	var input []string
// 	for len(input) == 0 {
// 		input = gamelogic.GetInput()

// 		if len(input) != 0 {
// 			shouldExit := processInput(pauseChannel, movesChannel, input, gameState, username)
// 			if shouldExit {
// 				return true
// 			}
// 			input = []string{}
// 		}
// 	}
// 	return false
// }

func main() {
	fmt.Println("Starting Peril client...")
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Peril game client connected to RabbitMQ!")

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}
	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gs.GetUsername(),
		routing.ArmyMovesPrefix+".*",
		pubsub.SimpleQueueTransient,
		handlerMove(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.SimpleQueueTransient,
		handlerWar(gs),
	)
	// if err != nil {
	// 	log.Fatalf("could not subscribe to war declarations: %v", err)
	// }
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "move":
			mv, err := gs.CommandMove(words)
			if err != nil {
				fmt.Println(err)
				continue
			}

			err = pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+mv.Player.Username,
				mv,
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				continue
			}
			fmt.Printf("Moved %v units to %s\n", len(mv.Units), mv.ToLocation)
		case "spawn":
			err = gs.CommandSpawn(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			// TODO: publish n malicious logs
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("unknown command")
		}
	}
}
