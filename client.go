package microservice

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/streadway/amqp"
)

type BaseRequest struct {
	Cmd string `json:"cmd"`
}

// RPCClient хранит состояние клиента микросервиса
type RPCClient struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	msgs <-chan amqp.Delivery
	q    amqp.Queue
}

// NewRPCClient создает новый объект клиента микросервиса.
func NewRPCClient() *RPCClient {
	var err error
	cl := &RPCClient{}
	cl.conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	FailOnError(err, "Failed to connect to RabbitMQ")

	cl.ch, err = cl.conn.Channel()
	FailOnError(err, "Failed to open a channel")

	cl.q, err = cl.ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // noWait
		nil,   // arguments
	)
	FailOnError(err, "Failed to declare a queue")

	cl.msgs, err = cl.ch.Consume(
		cl.q.Name, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	FailOnError(err, "Failed to register a consumer")

	return cl
}

// Request выполняет запрос к микросервису по имени `srvName`.
// Запрос квитируется уникальным идентификатором corrID.
// Поле `args` содержит JSON представление запроса.
func (cl *RPCClient) Request(srvName, correlationID string, args []byte) {
	err := cl.ch.Publish(
		"",      // exchange
		srvName, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       cl.q.Name,
			Body:          args,
		})
	FailOnError(err, "Failed to publish a message")
}

// Result блокирует ход исполнения до момента ответа микросервиса на запрос и
// возвращает сам ответ в виде JSON.
func (cl *RPCClient) Result(correlationID string) []byte {
	for d := range cl.msgs {
		if correlationID == d.CorrelationId {
			return d.Body
		}
	}
	return nil
}

// Close освобождает ресурсы клиента при его закрытии.
func (cl *RPCClient) Close() {
	cl.conn.Close()
	cl.ch.Close()
}

// CreateCmdRequest формирует данные для запроса команды `cmd` в виде CorrelationID и тела
// JSON.
func CreateCmdRequest(cmd string) (_ string, data []byte, err error) {
	correlationID, _ := uuid.NewV4()
	request := BaseRequest{cmd}
	data, err = json.Marshal(&request)
	if err != nil {
		return
	}
	return correlationID.String(), data, nil
}

// ParseErrorAnswer разбирает ответ с описанием ошибки.
func ParseErrorAnswer(data []byte) (_ *ErrorResponse, err error) {
	resp := &ErrorResponse{}
	if err = json.Unmarshal(data, &resp); err != nil {
		return
	}
	return resp, nil
}
