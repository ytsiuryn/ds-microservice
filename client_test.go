package microservice

import (
	"context"
	"encoding/json"
	"testing"
)

const (
	testServiceName = "test_microservice"
)

func TestServicePing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startTestService(ctx)

	cl := NewRPCClient()
	defer cl.Close()

	correlationID, data, _ := CreateCmdRequest("ping")
	cl.Request(testServiceName, correlationID, data)

	respData := cl.Result(correlationID)
	if len(respData) != 0 {
		t.Fail()
	}
}

func TestServiceInfo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startTestService(ctx)

	cl := NewRPCClient()
	defer cl.Close()

	correlationID, data, _ := CreateCmdRequest("info")
	cl.Request(testServiceName, correlationID, data)

	info := ServiceInfo{}
	respData := cl.Result(correlationID)
	json.Unmarshal(respData, &info)
	if info.Name != testServiceName {
		t.Fail()
	}
}

func TestUnknownServiceCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startTestService(ctx)

	cl := NewRPCClient()
	defer cl.Close()

	correlationID, data, _ := CreateCmdRequest("x")
	cl.Request(testServiceName, correlationID, data)

	info, _ := ParseErrorAnswer(cl.Result(correlationID))
	// {"error": "Unknown command: x", "context": "Message dispatcher"}

	if info.Error != "Unknown command: x" {
		t.Fail()
	}
}

func startTestService(ctx context.Context) {
	srv := NewService(testServiceName)
	defer srv.Cleanup()

	srv.ConnectToMessageBroker("amqp://guest:guest@localhost:5672/")

	srv.Dispatch()
}
