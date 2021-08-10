package microservice

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testServiceName        = "test_service"
	defaultRabbitMQConnStr = "amqp://guest:guest@localhost:5672/"
)

var testService *Service

func TestBaseServiceCommands(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTestService(ctx)

	cl := NewRPCClient()
	defer cl.Close()

	correlationID, data, err := CreateCmdRequest("ping")
	require.NoError(t, err)
	cl.Request(testServiceName, correlationID, data)
	respData := cl.Result(correlationID)
	assert.Empty(t, respData)

	correlationID, data, _ = CreateCmdRequest("x")
	cl.Request(testServiceName, correlationID, data)
	resp, err := ParseErrorAnswer(cl.Result(correlationID))
	require.NoError(t, err)
	// {"error": "Unknown command: x", "context": "Message dispatcher"}
	assert.NotEmpty(t, resp.Error)
}

func TestSubscribing(t *testing.T) {
	testMsg := []byte("Hello")
	go func() {
		pub := NewPublisher("test")
		pub.Connect(defaultRabbitMQConnStr)
		defer pub.Close()
		time.Sleep(10 * time.Millisecond)
		err := pub.Emit("text/plain", testMsg)
		require.NoError(t, err)
	}()
	sub := NewSubscriber("test")
	sub.Connect(defaultRabbitMQConnStr)
	defer sub.Close()
	delivery := sub.ReceiveOnce()
	assert.Equal(t, delivery.Body, testMsg)
}

func startTestService(ctx context.Context) {
	testService = NewService(testServiceName)
	msgs := testService.ConnectToMessageBroker(defaultRabbitMQConnStr)
	go func() {
		var req BaseRequest
		for delivery := range msgs {
			if err := json.Unmarshal(delivery.Body, &req); err != nil {
				continue
			}
			testService.RunCmd(req.Cmd, &delivery)
		}
	}()
}
