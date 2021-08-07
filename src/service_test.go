package microservice

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testServiceName = "test_service"

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

func startTestService(ctx context.Context) {
	testService = NewService(testServiceName)
	msgs := testService.ConnectToMessageBroker("amqp://guest:guest@localhost:5672/")
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
