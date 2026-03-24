// Package pubsub contains all reusable functionality for peril
package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

type SimpleQueueType string

const (
	Ack AckType = iota
	NackDiscard
	NackRequeue
)

const (
	Durable   SimpleQueueType = "durable"
	Transient SimpleQueueType = "transient"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("issue marshaling value: %v", err)
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	})
	if err != nil {
		return fmt.Errorf("issue publishing from the channel: %v", err)
	}

	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // SimpleQueueType is an "enum" type I made to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("issue creating channel connection: %v", err)
	}

	var isDirect, isTransient = false, false

	if queueType == Durable {
		isDirect = true
	} else {
		isTransient = true
	}

	queue, err := ch.QueueDeclare(queueName, isDirect, isTransient, isTransient, false, amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	})
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("issue creating queue: %v", err)
	}

	err = ch.QueueBind(queue.Name, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("issue binding queue: %v", err)
	}

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	ch, queue, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)
	if err != nil {
		return err
	}

	ampqDeliveryChan, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("issue getting amqp.Delivery chan from ch.consume: %v", err)
	}

	go func() {
		defer ch.Close()
		for c := range ampqDeliveryChan {
			var data T
			err = json.Unmarshal(c.Body, &data)
			if err != nil {
				fmt.Printf("couldn't Unmarshal %s", c.MessageId)
				continue
			}

			switch handler(data) {
			case Ack:
				c.Ack(false)
				fmt.Print("Ack")
			case NackDiscard:
				c.Nack(false, false)
				fmt.Print("Nack discard")
			case NackRequeue:
				c.Nack(false, true)
				fmt.Print("Nack requeue")
			}
		}
	}()

	return nil
}
