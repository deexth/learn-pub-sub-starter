package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print(">")

		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(mv gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print(">")

		moveOutcome := gs.HandleMove(mv)
		switch moveOutcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		case gamelogic.MoveOutcomeMakeWar:
			if err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: mv.Player,
					Defender: gs.GetPlayerSnap(),
				},
			); err != nil {
				fmt.Println(err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}

		fmt.Println("error: unknown moveOutcome")
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print(">")

		outcome, winner, loser := gs.HandleWar(rw)

		switch outcome {
		case gamelogic.WarOutcomeDraw:
			err := publishGob(
				fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser),
				gs.GetUsername(),
				ch,
			)
			if err != nil {
				fmt.Print(err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeOpponentWon:
			err := publishGob(
				fmt.Sprintf("%s won a war against %s", winner, loser),
				gs.GetUsername(),
				ch,
			)
			if err != nil {
				fmt.Print(err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			err := publishGob(
				fmt.Sprintf("%s won a war against %s", winner, loser),
				gs.GetUsername(),
				ch,
			)
			if err != nil {
				fmt.Print(err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}
		fmt.Print("error: unknown war outcome")
		return pubsub.NackDiscard
	}
}

func publishGob(username, msg string, ch *amqp.Channel) error {
	err := pubsub.PublishGob(
		ch,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			Message:     msg,
			CurrentTime: time.Now().UTC(),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func handlerSpam(inputs []string) (int, error) {
	if len(inputs) < 2 {
		fmt.Println("usage: spam <n>")
		return 0, nil
	}

	spamNum, err := strconv.ParseInt(inputs[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("no number provided, issue converting 2nd value to int: %v", err)
	}

	return int(spamNum), err
}
