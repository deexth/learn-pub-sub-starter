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
		log.Fatalf("issue creating client connection: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalln(err)
	}

	queueName := routing.PauseKey + "." + username

	_, queue, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, "transient")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	gs := gamelogic.NewGameState(username)

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
			continue
		}
	}

	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, os.Interrupt)
	// <-signalChan
	// fmt.Print("peril client shutting down...")
}
