package ports

import (
	"context"

	"github.com/rabbitmq/amqp091-go"

	"user-manager-api/internal/infrastructure/mq"
)

type RabbitMQ interface {
	Connect(ctx context.Context, dsn string) error
	Init() error
	PublisherWorker(ctx context.Context)
	GetInputChan() chan mq.Event
	GetConn() *amqp091.Connection
}
