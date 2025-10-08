package ports

import "context"

type RMQConsumer interface {
	Connect(dsn string) error
	Init() error
	DeliveryWorker(ctx context.Context)
}
