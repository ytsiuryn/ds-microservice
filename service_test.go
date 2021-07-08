package microservice

import (
	"context"
	"encoding/json"
	"testing"
)

const testServiceName = "test_service"

var testService *Service

func TestBaseServiceCommands(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTestService(ctx)

	cl := NewRPCClient()
	defer cl.Close()

	correlationID, data, _ := CreateCmdRequest("ping")
	cl.Request(testServiceName, correlationID, data)
	respData := cl.Result(correlationID)
	if len(respData) != 0 {
		t.Fail()
	}

	correlationID, data, _ = CreateCmdRequest("x")
	cl.Request(testServiceName, correlationID, data)
	vInfo, _ := ParseErrorAnswer(cl.Result(correlationID))
	// {"error": "Unknown command: x", "context": "Message dispatcher"}
	if vInfo.Error != "Unknown command: x" {
		t.Fail()
	}
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
