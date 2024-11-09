package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

type RabbitClient struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	url        string
}

func newConnection(url string) (*amqp.Connection, *amqp.Channel, error) {
	connection, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to rabbitmq: %v", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a channel: %v", err)
	}

	return connection, channel, nil
}

func NewRabbitClient(connectionUrl string) (*RabbitClient, error) {
	connection, channel, err := newConnection(connectionUrl)
	if err != nil {
		return nil, err
	}

	return &RabbitClient{
		connection: connection,
		channel:    channel,
		url:        connectionUrl,
	}, nil
}

func (rc *RabbitClient) ConsumeMessage(exchange, routingKey, queueName string) (<-chan amqp.Delivery, error) {
	err := rc.prepareExchangeAndQueue(exchange, routingKey, queueName)
	if err != nil {
		return nil, fmt.Errorf("error preparing exchange and queue: %v", err)
	}

	messages, err := rc.channel.Consume(
		queueName, // queue
		"goapp",   // consumer
		false,     // autoAck,
		false,     // exclusive
		false,     // noLocal
		false,     // noWait
		nil,       // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume messages from queue: %v", err)
	}

	return messages, nil
}

func (rc *RabbitClient) PublishMessage(exchange, routingKey, queueName string, message []byte) error {
	err := rc.prepareExchangeAndQueue(exchange, routingKey, queueName)
	if err != nil {
		return fmt.Errorf("error preparing exchange and queue: %v", err)
	}

	err = rc.channel.Publish(
		exchange,   // exchange
		routingKey, // key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		}, // msg
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}

func (rc *RabbitClient) prepareExchangeAndQueue(exchange, routingKey, queueName string) error {
	err := rc.channel.ExchangeDeclare(
		exchange, // name
		"direct", // kind
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	queue, err := rc.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // autoDelete
		false,     // exclusive
		false,     // noWait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	err = rc.channel.QueueBind(
		queue.Name, // name
		routingKey, // key
		exchange,   // exchange
		false,      // noWait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %v", err)
	}

	return nil
}

func (rc *RabbitClient) Close() {
	rc.channel.Close()
	rc.connection.Close()
}
