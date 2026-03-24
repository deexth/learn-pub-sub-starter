package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	rabbit := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbit)
	if err != nil {
		log.Fatalf("Issue creating client connection: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("issue creating channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalln(err)
	}

	gs := gamelogic.NewGameState(username)

	queueName := routing.PauseKey + "." + username

	// // _, queue, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, "transient")
	// _, queue, err := pubsub.DeclareAndBind(
	// 	conn,
	// 	routing.ExchangePerilTopic,
	// 	routing.ArmyMovesPrefix+"."+username,
	// 	routing.ArmyMovesPrefix+".*",
	// 	pubsub.Transient,
	// )
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// fmt.Printf("Queue %v declared and bound!\n", queue.Name)
	fmt.Println("Subscribed to json...")

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username,
		routing.ArmyMovesPrefix+".*",
		pubsub.Transient,
		handlerMove(gs, ch),
	)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Subscribed to move...")

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(gs),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Subscribed to war...")

	for {
		inputs := gamelogic.GetInput()
		if len(inputs) == 0 {
			continue
		}

		switch inputs[0] {
		case "spawn":
			err = gs.CommandSpawn(inputs)
			if err != nil {
				fmt.Printf("Issue spawning unit: %v", err)
				continue
			}
		case "move":
			move, err := gs.CommandMove(inputs)
			if err != nil {
				fmt.Printf("Issue moving unit: %v", err)
				continue
			}

			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+username,
				move,
			)
			if err != nil {
				fmt.Printf("Issue publishing move: %v", err)
				continue
			}
			fmt.Printf("move to %s", move.ToLocation)
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("No such command")
		}
	}

	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, os.Interrupt)
	// <-signalChan
	// fmt.Print("peril client shutting down...")
}
