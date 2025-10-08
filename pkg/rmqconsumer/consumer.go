package rmqconsumer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"user-manager-api/config"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// can scale depends on a parallel worker count
const preFetchCount = 1

type Consumer struct {
	cfg        config.MQ
	log        *zap.Logger
	conn       *amqp091.Connection
	chConsume  *amqp091.Channel
	chDelivery <-chan amqp091.Delivery
}

func New(cfg config.MQ, logger *zap.Logger, conn *amqp091.Connection) *Consumer {
	return &Consumer{
		cfg:  cfg,
		log:  logger,
		conn: conn,
	}
}

var err error

func (c *Consumer) Connect(dsn string) error {
	c.conn, err = amqp091.Dial(dsn)
	if err != nil {
		return fmt.Errorf("amqp dial: %w", err)
	}
	c.chConsume, err = c.conn.Channel()
	if err != nil {
		_ = c.conn.Close()
		return fmt.Errorf("amqp channel: %w", err)
	}

	c.log.Info("rabbitmq consumer connected successfully")

	return err
}

func (c *Consumer) Init() error {
	if err = c.chConsume.ExchangeDeclare(
		c.cfg.Exchange,
		c.cfg.ExchangeType,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("exchange declare: %w", err)
	}
	if _, err = c.chConsume.QueueDeclare(
		c.cfg.QueueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("queue declare: %w", err)
	}
	for _, rk := range []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	} {
		if err = c.chConsume.QueueBind(
			c.cfg.QueueName,
			rk,
			c.cfg.Exchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("queue bind %s: %w", rk, err)
		}
	}

	if err = c.chConsume.Qos(preFetchCount, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	var cerr error
	c.chDelivery, cerr = c.chConsume.Consume(
		c.cfg.QueueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if cerr != nil {
		return fmt.Errorf("consume: %w", cerr)
	}

	return nil
}

func (c *Consumer) DeliveryWorker(ctx context.Context) {
	c.log.Info("starting delivery worker")

	defer func() {
		c.log.Info("delivery worker gracefully stopped")
	}()

	for {
		select {
		case msg := <-c.chDelivery:
			// we can also use "fan-out" chan here with "worker-pool"
			// in case of heavy logic processing of messages
			if err = c.delivery(msg); err != nil {
				// alert
				c.log.Error("mq read message error", zap.Error(err))
			}
		case <-ctx.Done():
			c.chConsume.Close()
			return
		}
	}
}

func (c *Consumer) delivery(msg amqp091.Delivery) error {
	// we are having simple delivery but in prod
	// we should implement also ack/nack procedures

	var action string
	switch msg.RoutingKey {
	case http.MethodPost:
		action = "UserCreated"
	case http.MethodPut:
		action = "UserUpdated"
	case http.MethodDelete:
		action = "UserDeleted"
	}

	fmt.Fprintf(os.Stdout,
		"Action=%s EventBody=%s\n",
		action,
		string(msg.Body),
	)

	return nil
}
