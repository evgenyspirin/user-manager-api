package rmqconsumer

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"user-manager-api/config"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	defer func() { os.Stdout = old }()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func Test_delivery_Table(t *testing.T) {
	type tc struct {
		name       string
		routingKey string
		body       string
		wantOut    string
	}
	cases := []tc{
		{"POST -> UserCreated", "POST", `{"id":1}`, "Action=UserCreated EventBody={\"id\":1}\n"},
		{"PUT  -> UserUpdated", "PUT", `{"id":2}`, "Action=UserUpdated EventBody={\"id\":2}\n"},
		{"DELETE -> UserDeleted", "DELETE", `{"id":3}`, "Action=UserDeleted EventBody={\"id\":3}\n"},
		{"Unknown -> empty", "PATCH", `{"id":4}`, "Action= EventBody={\"id\":4}\n"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			c := &Consumer{}
			out := captureStdout(t, func() {
				msg := amqp091.Delivery{RoutingKey: tt.routingKey, Body: []byte(tt.body)}
				err := c.delivery(msg)
				require.NoError(t, err)
			})
			require.Equal(t, tt.wantOut, out)
		})
	}
}

func TestConnect_InvalidDSN(t *testing.T) {
	l := zap.NewNop()
	c := New(config.MQ{}, l, nil)

	err := c.Connect("amqp://bad:://dsn")
	require.Error(t, err)
	require.Nil(t, c.chConsume)
	require.Nil(t, c.conn)
}
