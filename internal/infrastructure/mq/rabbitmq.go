package mq

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"user-manager-api/config"
	"user-manager-api/internal/interface/api/rest/dto/user"
)

// "Rely on metrics, not guesses."
const bufferSize = 128

type (
	InputCh  = chan Event
	RabbitMQ struct {
		cfg   config.MQ
		log   *zap.Logger
		conn  *amqp091.Connection
		pubCh *amqp091.Channel
		in    InputCh
	}
	Event struct {
		Id      uuid.UUID `json:"event_id"`
		TS      time.Time `json:"time_stamp"`
		Method  string    `json:"event_action"`
		UserID  string    `json:"user_id"`
		Payload user.User `json:"user_payload"`
	}
)

func New(cfg config.MQ, logger *zap.Logger) *RabbitMQ {
	return &RabbitMQ{
		cfg: cfg,
		log: logger,
		in:  make(chan Event, bufferSize),
	}
}

func (r *RabbitMQ) Connect(ctx context.Context, dsn string) error {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	amqpCfg := amqp091.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
		Properties: amqp091.Table{
			"connection_name": "usermanagerapi",
		},
		Dial: func(network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: nil,
	}

	var err error
	r.conn, err = amqp091.DialConfig(dsn, amqpCfg)
	if err != nil {
		return err
	}
	r.pubCh, err = r.conn.Channel()
	if err != nil {
		_ = r.conn.Close()
		return err
	}

	r.log.Info("rabbitmq connected successfully")

	return err
}

func (r *RabbitMQ) Init() error {
	var err error
	if err = r.pubCh.ExchangeDeclare(
		r.cfg.Exchange,
		r.cfg.ExchangeType,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = r.pubCh.Close()
		return err
	}
	q, err := r.pubCh.QueueDeclare(
		r.cfg.QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for _, rk := range []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	} {
		if err = r.pubCh.QueueBind(q.Name, rk, r.cfg.Exchange, false, nil); err != nil {
			return err
		}
	}

	return nil
}

func (r *RabbitMQ) PublisherWorker(ctx context.Context) {
	r.log.Info("starting publisher worker ")

	defer func() {
		r.log.Info("publisher worker gracefully stopped")
	}()

	for {
		select {
		case e := <-r.in:
			if err := r.publish(ctx, e); err != nil {
				// alert
				r.log.Error("mq publish error", zap.Error(err))
			}
		case <-ctx.Done():
			close(r.in)
			r.pubCh.Close()
			return
		}
	}
}

func (r *RabbitMQ) publish(ctx context.Context, e Event) error {
	// for a good boost of performance(x3 minimum) and to avoid reflection under the hood
	// better to use codegen for marshal/unmarshal for example:
	// https://github.com/mailru/easyjson
	b, err := json.Marshal(e)
	if err != nil {
		// alert
		return err
	}

	pub := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		MessageId:    e.Id.String(),
		Timestamp:    e.TS,
		Type:         e.Method,
		Body:         b,
	}
	if err = r.pubCh.PublishWithContext(
		ctx,
		r.cfg.Exchange,
		e.Method,
		true,
		false,
		pub,
	); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) GetInputChan() chan Event     { return r.in }
func (r *RabbitMQ) GetConn() *amqp091.Connection { return r.conn }
